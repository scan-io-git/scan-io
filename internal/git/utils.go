package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	gitconfig "github.com/go-git/go-git/v5/config"
	crssh "golang.org/x/crypto/ssh"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// determineBranch returns the appropriate branch reference.
func determineBranch(branch, defaultBranch string) plumbing.ReferenceName {
	if branch == "" {
		branch = defaultBranch
	}
	ref := plumbing.ReferenceName(branch)
	if !ref.IsBranch() && !ref.IsRemote() && !ref.IsTag() && !ref.IsNote() {
		return plumbing.NewBranchReferenceName(branch)
	}
	return ref
}

// getLoggerOutput prepares the logger output.
func getLoggerOutput(logger hclog.Logger) io.Writer {
	return logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
		ForceLevel:  logger.GetLevel(),
	})
}

// updateRepository fetches updates from the remote repository and handles errors.
func updateRepository(ctx context.Context, repo *git.Repository, auth transport.AuthMethod, logger hclog.Logger, globalConfig *config.Config, output io.Writer, targetFolder string, branch plumbing.ReferenceName) (*git.Repository, error) {
	fetchOptions := &git.FetchOptions{
		RemoteName:      "origin",
		Auth:            auth,
		Progress:        output,
		RefSpecs:        []gitconfig.RefSpec{"+refs/*:refs/*"},
		Depth:           config.SetThen(globalConfig.GitClient.Depth, 1),
		InsecureSkipTLS: config.GetBoolValue(globalConfig.GitClient, "InsecureTLS", false),
	}

	if err := repo.FetchContext(ctx, fetchOptions); err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			logger.Info("repository already up-to-date", "targetFolder", targetFolder)
		} else if err.Error() == "object not found" || err.Error() == "reference not found" {
			logger.Error("object/reference not found in the repository. Cleaning up the repo ...", "targetFolder", targetFolder, "error", err)
			// TODO: double test it
			if err := os.RemoveAll(targetFolder); err != nil {
				logger.Error("failed to remove repository", "error", err)
				return nil, fmt.Errorf("failed to remove repository: %w", err)
			}

			// TODO: should we send the new repo to global context?
			_, err := git.PlainCloneContext(ctx, targetFolder, false, &git.CloneOptions{
				Auth:            auth,
				URL:             fetchOptions.RemoteName,
				ReferenceName:   branch,
				Progress:        output,
				Depth:           fetchOptions.Depth,
				InsecureSkipTLS: fetchOptions.InsecureSkipTLS,
			})
			if err != nil {
				logger.Error("retrying clone failed", "error", err)
				return nil, fmt.Errorf("retrying clone failed: %w", err)
			}
		} else {
			logger.Error("error occurred during fetch", "error", err, "targetFolder", targetFolder)
			return nil, fmt.Errorf("error occurred during fetch: %w", err)
		}
	}
	return repo, nil
}

// checkoutAndResetBranch checks out and resets the branch.
func checkoutAndResetBranch(repo *git.Repository, branch plumbing.ReferenceName, logger hclog.Logger, targetFolder string) error {
	w, err := repo.Worktree()
	if err != nil {
		logger.Error("error accessing worktree", "error", err, "targetFolder", targetFolder)
		return fmt.Errorf("error accessing worktree: %w", err)
	}

	logger.Debug("checking out branch", "branch", branch, "targetFolder", targetFolder)
	if err := w.Checkout(&git.CheckoutOptions{
		Branch: branch,
		Force:  true,
	}); err != nil {
		logger.Error("error occurred during checkout", "error", err, "targetFolder", targetFolder)
		return fmt.Errorf("error occurred during checkout: %w", err)
	}

	logger.Debug("resetting local repository", "targetFolder", targetFolder)
	if err := w.Reset(&git.ResetOptions{
		Mode: git.HardReset,
	}); err != nil {
		logger.Error("error occurred during reset", "error", err, "targetFolder", targetFolder)
		return fmt.Errorf("error occurred during reset: %w", err)
	}
	return nil
}

func pullLatestChanges(ctx context.Context, repo *git.Repository, cfg *config.Config, auth transport.AuthMethod, branch plumbing.ReferenceName, logger hclog.Logger, output io.Writer) error {
	w, err := repo.Worktree()
	if err != nil {
		logger.Error("error accessing worktree", "error", err)
		return fmt.Errorf("error accessing worktree: %w", err)
	}

	logger.Debug("attempting to pull the latest changes", "branch", branch)
	err = w.PullContext(ctx, &git.PullOptions{
		Auth:            auth,
		ReferenceName:   branch,
		Progress:        output,
		Force:           true,
		InsecureSkipTLS: config.GetBoolValue(cfg.GitClient, "InsecureTLS", false),
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		logger.Error("error occurred during pull", "error", err)
		return fmt.Errorf("error occurred during pull: %w", err)
	}
	return nil
}

// findGitRepositoryPath function finds a git repository path for a given source folder
func findGitRepositoryPath(sourceFolder string) (string, error) {
	if sourceFolder == "" {
		return "", fmt.Errorf("source folder is not set")
	}

	// check if source folder is a subfolder of a git repository
	for {
		_, err := git.PlainOpen(sourceFolder)
		if err == nil {
			return sourceFolder, nil
		}

		// move up one level
		sourceFolder = filepath.Dir(sourceFolder)

		// check if reached the root folder
		if sourceFolder == filepath.Dir(sourceFolder) {
			break
		}
	}

	return "", fmt.Errorf("source folder is not a git repository")
}
