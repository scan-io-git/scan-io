package git

import (
	"context"
	"fmt"
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
)

// type VCSFetchRequest struct {
// 	CloneURL     string
// 	Branch       string
// 	AuthType     string
// 	SSHKey       string
// 	TargetFolder string
// 	Mode         string
// 	RepoParam    shared.RepositoryParams
// }

// type RepositoryParams struct {
// 	Namespace string `json:"namespace"`
// 	RepoName  string `json:"repo_name"`
// 	PRID      string `json:"pr_id"`
// 	VCSURL    string `json:"vcs_url"`
// 	HttpLink  string `json:"http_link"`
// 	SshLink   string `json:"ssh_link"`
// }

// // CloneConfig represents the configuration needed to clone a repository.
// type CloneConfig struct {
//     RepositoryURL string
//     Branch        string
// 	AuthType string
// 	SSHKey       string
//     Destination   string
// 	Mode         string
// }

// getAuth configures the appropriate Git authentication method based on the provided credentials and environment variables.
func getAuth(args *shared.VCSFetchRequest, config *config.BitbucketPlugin, logger hclog.Logger) (transport.AuthMethod, error) {
	var auth transport.AuthMethod
	var err error

	switch args.AuthType {
	case "ssh-key":
		logger.Debug("Setting up SSH key authentication")

		sshKeyPath, err := shared.ExpandPath(args.SSHKey)
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
			HostKeyCallback: crssh.InsecureIgnoreHostKey(), // TODO fix
		}

	case "ssh-agent":
		logger.Debug("setting up SSH agent authentication")
		auth, err = ssh.NewSSHAgentAuth("git")
		if err != nil {
			logger.Error("failed to set up SSH agent authentication", "err", err)
			return nil, err
		}

		auth.(*ssh.PublicKeysCallback).HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
			HostKeyCallback: crssh.InsecureIgnoreHostKey(), // TODO fix
		}

	case "http":
		logger.Debug("setting up HTTP authentication")
		auth = &http.BasicAuth{
			Username: config.BitbucketUsername,
			Password: config.SSHKeyPassword,
		}

	default:
		err := fmt.Errorf("unknown auth type: %s", args.AuthType)
		logger.Error("unsupported authentication type", "error", err)
		return nil, err
	}
	return auth, err
}

// CloneRepository clones a Git repository based on the provided VCSFetchRequest and environment variables.
// It handles authentication, checks if the repository exists, and updates it if necessary.
func CloneRepository(logger hclog.Logger, globalConfig *config.Config, args *shared.VCSFetchRequest) (string, error) {
	timeout := time.Duration(600 * time.Second) //g.config
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	info, err := vcsurl.Parse(args.CloneURL)
	if err != nil {
		logger.Error("failed to parse VCS URL", "VCSURL", args.CloneURL, "err", err)
		return "", err
	}

	// Determine the branch reference, defaulting to a branch name with an additional prefix if not fully specified
	branch := plumbing.ReferenceName(args.Branch)
	if !branch.IsBranch() && !branch.IsRemote() && !branch.IsTag() && !branch.IsNote() {
		branch = plumbing.NewBranchReferenceName(args.Branch)
	}

	auth, err := getAuth(args, &globalConfig.BitbucketPlugin, logger)
	if err != nil {
		logger.Error("failed to set up Git authentication", "err", err)
		return "", err
	}

	// Prepare the logger for output from Git operations
	output := logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
		ForceLevel:  logger.GetLevel(), // use the same level as the logger
	})

	// Start the repository fetch operation
	repoPath := args.TargetFolder
	logger.Debug("starting repository fetch", "repository", info.Name, "branch", args.Branch, "targetFolder", args.TargetFolder)
	repo, err := git.PlainCloneContext(ctx, repoPath, false, &git.CloneOptions{
		Auth:            auth,
		URL:             args.CloneURL,
		ReferenceName:   branch,
		Progress:        output,
		Depth:           config.SetThen(globalConfig.GitClient.Depth, 1),
		InsecureSkipTLS: config.GetBoolValue(globalConfig.GitClient, "InsecureTLS", false),
	})
	if err != nil {
		if err != git.ErrRepositoryAlreadyExists {
			logger.Error("error occurred during clone", "error", err, "targetFolder", repoPath)
			return "", err
		}

		// Handle existing repository updates
		logger.Info("repository already exists, updating...", "targetFolder", repoPath)
		repo, err = git.PlainOpen(repoPath)
		if err != nil {
			logger.Error("cannot open existing repository", "error", err, "targetFolder", repoPath)
			return "", err
		}

		// TODO insecuretls from config

		// Fetch all updates from the remote
		if err = repo.FetchContext(ctx, &git.FetchOptions{
			RemoteName:      "origin",
			Auth:            auth,
			Progress:        output,
			RefSpecs:        []gitconfig.RefSpec{"+refs/*:refs/*"},
			Depth:           config.SetThen(globalConfig.GitClient.Depth, 1),
			InsecureSkipTLS: config.GetBoolValue(globalConfig.GitClient, "InsecureTLS", false),
		}); err != nil && err != git.NoErrAlreadyUpToDate {
			logger.Error("error occurred during fetch", "error", err, "targetFolder", repoPath)
			return "", err
		}
	}

	// Checkout and reset the branch to ensure it's up-to-date
	w, err := repo.Worktree()
	if err != nil {
		logger.Error("error accessing worktree", "err", err, "targetFolder", args.TargetFolder)
		return "", err
	}

	// Switching to a local branch
	logger.Debug("checking a branch", "repository", info.Name, "branch", args.Branch, "targetFolder", args.TargetFolder)
	if err = w.Checkout(&git.CheckoutOptions{Branch: branch, Force: true}); err != nil {
		logger.Error("error occurred during checkout", "error", err, "targetFolder", repoPath)
		return "", err
	}

	// Reset the worktree to ensure it is clean
	logger.Debug("reseting a local repository", "repository", info.Name, "targetFolder", args.TargetFolder)
	if err := w.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
		fmt.Println("error occurred during reset", "err", err, "targetFolder", args.TargetFolder)
		return "", err
	}

	// Pull if the repository was already present
	logger.Debug("attempting to pull the latest changes", "repository", repoPath)
	if err == git.ErrRepositoryAlreadyExists {
		if err = w.Pull(&git.PullOptions{Auth: auth, ReferenceName: branch, Progress: output}); err != nil {
			if err != git.NoErrAlreadyUpToDate {
				logger.Error("error occurred during pull", "error", err, "targetFolder", repoPath)
				return "", err
			}
		}
	}

	logger.Info("repository operation completed successfully", "repository", info.Name, "branch", args.Branch, "targetFolder", args.TargetFolder)
	return args.TargetFolder, nil
}
