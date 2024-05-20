package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	gitconfig "github.com/go-git/go-git/v5/config"
	crssh "golang.org/x/crypto/ssh"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/hashicorp/go-hclog"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
)

// Authenticator defines an interface for different authentication methods.
type Authenticator interface {
	SetupAuth(args *shared.VCSFetchRequest, config *config.BitbucketPlugin, logger hclog.Logger) (transport.AuthMethod, error)
}

// SSHKeyAuthenticator provides SSH key-based authentication.
type SSHKeyAuthenticator struct{}

// SSHAgentAuthenticator provides SSH agent-based authentication.
type SSHAgentAuthenticator struct{}

// HTTPAuthenticator provides HTTP basic authentication.
type HTTPAuthenticator struct{}

// SetupAuth configures SSH key authentication.
func (s *SSHKeyAuthenticator) SetupAuth(args *shared.VCSFetchRequest, config *config.BitbucketPlugin, logger hclog.Logger) (transport.AuthMethod, error) {
	logger.Debug("setting up SSH key authentication")

	var auth transport.AuthMethod
	sshKeyPath, err := files.ExpandPath(args.SSHKey)
	if err != nil {
		logger.Error("failed to expand SSH key path", "path", args.SSHKey, "error", err)
		return nil, err
	}

	auth, err = ssh.NewPublicKeysFromFile("git", sshKeyPath, config.SSHKeyPassword)
	if err != nil {
		logger.Error("failed to set up SSH key authentication", "error", err.Error())
		return nil, err
	}

	auth.(*ssh.PublicKeys).HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
		HostKeyCallback: crssh.InsecureIgnoreHostKey(), // TODO: Fix this
	}

	return auth, nil
}

// SetupAuth configures SSH agent authentication.
func (s *SSHAgentAuthenticator) SetupAuth(args *shared.VCSFetchRequest, config *config.BitbucketPlugin, logger hclog.Logger) (transport.AuthMethod, error) {
	logger.Debug("setting up SSH agent authentication")

	var auth transport.AuthMethod
	var err error
	auth, err = ssh.NewSSHAgentAuth("git")
	if err != nil {
		logger.Error("failed to set up SSH agent authentication", "error", err)
		return nil, err
	}

	auth.(*ssh.PublicKeysCallback).HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
		HostKeyCallback: crssh.InsecureIgnoreHostKey(), // TODO: Fix this
	}

	return auth, nil
}

// SetupAuth configures HTTP basic authentication.
func (h *HTTPAuthenticator) SetupAuth(args *shared.VCSFetchRequest, config *config.BitbucketPlugin, logger hclog.Logger) (transport.AuthMethod, error) {
	logger.Debug("setting up HTTP authentication")
	return &http.BasicAuth{
		Username: config.Username,
		Password: config.SSHKeyPassword,
	}, nil
}

// getAuthenticator returns the appropriate Authenticator based on the authentication type.
func getAuthenticator(authType string) (Authenticator, error) {
	switch authType {
	case "ssh-key":
		return &SSHKeyAuthenticator{}, nil
	case "ssh-agent":
		return &SSHAgentAuthenticator{}, nil
	case "http":
		return &HTTPAuthenticator{}, nil
	default:
		return nil, fmt.Errorf("unknown auth type: %s", authType)
	}
}

// CloneRepository clones a Git repository based on the provided VCSFetchRequest and globalConfig.
func CloneRepository(logger hclog.Logger, globalConfig *config.Config, args *shared.VCSFetchRequest) (string, error) {
	targetFolder := args.TargetFolder

	info, err := vcsurl.Parse(args.CloneURL)
	if err != nil {
		logger.Error("failed to parse VCS URL", "VCSURL", args.CloneURL, "error", err)
		return "", fmt.Errorf("failed to parse VCS URL: %w", err)
	}

	branch := determineBranch(args.Branch)
	authenticator, err := getAuthenticator(args.AuthType)
	if err != nil {
		logger.Error("unsupported authentication type", "error", err)
		return "", fmt.Errorf("unsupported authentication type: %w", err)
	}

	auth, err := authenticator.SetupAuth(args, &globalConfig.BitbucketPlugin, logger)
	if err != nil {
		logger.Error("failed to set up Git authentication", "error", err)
		return "", fmt.Errorf("failed to set up Git authentication: %w", err)
	}

	output := getLoggerOutput(logger)
	timeout := config.SetThen(globalConfig.GitClient.Timeout, time.Duration(10*time.Minute))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	logger.Debug("starting repository fetch", "repository", info.Name, "branch", branch, "targetFolder", targetFolder)
	repo, err := git.PlainCloneContext(ctx, targetFolder, false, &git.CloneOptions{
		Auth:            auth,
		URL:             args.CloneURL,
		ReferenceName:   branch,
		Progress:        output,
		Depth:           config.SetThen(globalConfig.GitClient.Depth, 1),
		InsecureSkipTLS: config.GetBoolValue(globalConfig.GitClient, "InsecureTLS", false),
	})
	if err != nil {
		if err != git.ErrRepositoryAlreadyExists {
			logger.Error("error occurred during clone", "error", err, "targetFolder", targetFolder)
			return "", fmt.Errorf("error occurred during clone: %w", err)
		}

		// Handle existing repository updates
		logger.Info("repository already exists, updating...", "targetFolder", targetFolder)
		repo, err = git.PlainOpen(targetFolder)
		if err != nil {
			logger.Error("cannot open existing repository", "error", err, "targetFolder", targetFolder)
			return "", fmt.Errorf("cannot open existing repository: %w", err)
		}

		repo, err = updateRepository(ctx, repo, auth, logger, globalConfig, output, targetFolder, branch)
		if err != nil {
			return "", err
		}
	}
	if err = checkoutAndResetBranch(repo, branch, logger, targetFolder); err != nil {
		return "", err
	}

	if err == git.ErrRepositoryAlreadyExists {
		if err := pullLatestChanges(ctx, repo, globalConfig, auth, branch, logger, output); err != nil {
			return "", err
		}
	}

	logger.Info("repository operation completed successfully", "repository", info.Name, "branch", args.Branch, "targetFolder", targetFolder)
	return targetFolder, nil
}

// determineBranch returns the appropriate branch reference.
func determineBranch(branch string) plumbing.ReferenceName {
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

	if err := repo.FetchContext(ctx, fetchOptions); err != nil && err != git.NoErrAlreadyUpToDate {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			logger.Info("repository already up-to-date", "targetFolder", targetFolder)
		} else if err.Error() == "object not found" {
			logger.Error("object not found in the repository. Cleaning up the repo ...", "targetFolder", targetFolder, "error", err)
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
