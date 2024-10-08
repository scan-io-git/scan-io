package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/hashicorp/go-hclog"

	gitconfig "github.com/go-git/go-git/v5/config"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"

	log "github.com/scan-io-git/scan-io/pkg/shared/logger"
)

func (c *Client) CloneRepository(args *shared.VCSFetchRequest) (string, error) {
	targetFolder := args.TargetFolder
	cloneURL := args.CloneURL

	info, err := vcsurl.Parse(cloneURL)
	if err != nil {
		c.logger.Error("failed to parse VCS URL", "VCSURL", cloneURL, "error", err)
		return "", fmt.Errorf("failed to parse VCS URL: %w", err)
	}

	reference, err := determineBranch(args.Branch, cloneURL, &c.auth)
	if err != nil {
		c.logger.Error("failed to determine branch", "error", err, "cloneURL", cloneURL)
		return "", err
	}
	output := log.GetLoggerOutput(c.logger)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	c.logger.Debug("starting repository fetch", "repository", info.Name, "branch", reference.Branch, "commit", reference.IsCommit, "cloneURL", cloneURL, "targetFolder", targetFolder)
	repo, err := git.PlainCloneContext(ctx, targetFolder, false, &git.CloneOptions{
		Auth:            c.auth,
		URL:             cloneURL,
		ReferenceName:   reference.Branch,
		Progress:        output,
		Depth:           config.SetThen(c.globalConfig.GitClient.Depth, 0),
		InsecureSkipTLS: config.GetBoolValue(c.globalConfig.GitClient, "InsecureTLS", false),
	})
	if err != nil {
		if err != git.ErrRepositoryAlreadyExists {
			c.logger.Error("error occurred during clone", "error", err, "targetFolder", targetFolder)
			return "", fmt.Errorf("error occurred during clone: %w", err)
		}

		c.logger.Info("repository already exists, updating...", "targetFolder", targetFolder)
		repo, err = git.PlainOpen(targetFolder)
		if err != nil {
			c.logger.Error("cannot open existing repository", "error", err, "targetFolder", targetFolder)
			return "", fmt.Errorf("cannot open existing repository: %w", err)
		}
	}

	repo, err = updateRepository(ctx, repo, c.auth, c.logger, c.globalConfig, output, targetFolder, reference.Branch, args)
	if err != nil {
		return "", err
	}

	if reference.IsCommit {
		c.logger.Warn("found commit fetching", "targetFolder", targetFolder)
		err = checkoutCommit(repo, reference.Hash, c.logger, targetFolder)
	} else {
		err = checkoutAndResetBranch(repo, reference.Branch, c.logger, targetFolder)
	}
	if err != nil {
		return "", err
	}

	if err == git.ErrRepositoryAlreadyExists {
		if err := pullLatestChanges(ctx, repo, c.globalConfig, c.auth, reference.Branch, c.logger, output); err != nil {
			return "", err
		}
	}

	c.logger.Info("repository operation completed successfully", "repository", info.Name, "branch", reference.Branch, "targetFolder", targetFolder)
	return targetFolder, nil
}

// updateRepository fetches updates from the remote repository and handles errors.
func updateRepository(ctx context.Context, repo *git.Repository, auth transport.AuthMethod, logger hclog.Logger, globalConfig *config.Config, output io.Writer, targetFolder string, branch plumbing.ReferenceName, args *shared.VCSFetchRequest) (*git.Repository, error) {
	fetchOptions := &git.FetchOptions{
		RemoteName:      "origin",
		Auth:            auth,
		Progress:        output,
		RefSpecs:        []gitconfig.RefSpec{"+refs/*:refs/*"},
		Depth:           config.SetThen(globalConfig.GitClient.Depth, 0),
		InsecureSkipTLS: config.GetBoolValue(globalConfig.GitClient, "InsecureTLS", false),
	}
	logger.Debug("update repo by using fetch", "depth", fetchOptions.Depth, "targetFolder", targetFolder)

	if err := repo.FetchContext(ctx, fetchOptions); err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			logger.Info("repository already up-to-date", "targetFolder", targetFolder)
		} else if err.Error() == "object not found" || err.Error() == "reference not found" {
			logger.Error("object/reference not found in the repository. Cleaning up the repo ...", "targetFolder", targetFolder, "error", err)
			if err := os.RemoveAll(targetFolder); err != nil {
				logger.Error("failed to remove repository", "error", err)
				return nil, fmt.Errorf("failed to remove repository: %w", err)
			}

			// TODO: should we send the new repo to global context?
			repo, err = git.PlainCloneContext(ctx, targetFolder, false, &git.CloneOptions{
				Auth:            auth,
				URL:             args.CloneURL,
				ReferenceName:   branch,
				Progress:        output,
				Depth:           fetchOptions.Depth,
				InsecureSkipTLS: fetchOptions.InsecureSkipTLS,
			})
			logger.Debug("reclone of corrupted", "url", args.CloneURL, "referenceName", branch, "depth", fetchOptions.Depth)
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

// checkoutCommit checks out a specific commit in the repository.
func checkoutCommit(repo *git.Repository, commitHash plumbing.Hash, logger hclog.Logger, targetFolder string) error {
	logger.Debug("checking out revision", "revision", commitHash.String(), "targetFolder", targetFolder)
	w, err := repo.Worktree()
	if err != nil {
		logger.Error("error accessing worktree", "error", err, "targetFolder", targetFolder)
		return fmt.Errorf("error accessing worktree: %w", err)
	}
	h, err := repo.ResolveRevision(plumbing.Revision(commitHash.String()))
	if err != nil {
		logger.Error("error resolving revision", "error", err, "revision", commitHash.String(), "targetFolder", targetFolder)
		return fmt.Errorf("error resolving revision: %w", err)
	}

	if err := w.Checkout(&git.CheckoutOptions{
		Hash:  *h,
		Force: true,
	}); err != nil {
		logger.Error("error occurred during checkout", "error", err, "targetFolder", targetFolder)
		return fmt.Errorf("error occurred during checkout: %w", err)
	}
	return nil
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
