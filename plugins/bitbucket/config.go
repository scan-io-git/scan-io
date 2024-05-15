package main

import (
	"fmt"
	"os"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// UpdateConfigFromEnv sets configuration values from environment variables, if they are set.
func UpdateConfigFromEnv(cfg *config.Config) error {
	if username := os.Getenv("SCANIO_BITBUCKET_USERNAME"); username != "" {
		cfg.BitbucketPlugin.BitbucketUsername = username
	}
	if token := os.Getenv("SCANIO_BITBUCKET_TOKEN"); token != "" {
		cfg.BitbucketPlugin.BitbucketToken = token
	}
	if sshKeyPass := os.Getenv("SCANIO_BITBUCKET_SSH_KEY_PASSWORD"); sshKeyPass != "" {
		cfg.BitbucketPlugin.SSHKeyPassword = sshKeyPass
	}
	return nil
}

func (g *VCSBitbucket) validateCommonCredentials() error {
	if len(g.globalConfig.BitbucketPlugin.BitbucketUsername) == 0 || len(g.globalConfig.BitbucketPlugin.BitbucketToken) == 0 {
		return fmt.Errorf("both Bitbucket username and token are required")
	}
	return nil
}

func (g *VCSBitbucket) validateBaseArgs(args *shared.VCSRequestBase) error {
	if args.VCSURL == "" {
		return fmt.Errorf("repository URL is required")
	}
	if args.Namespace == "" {
		return fmt.Errorf("namespace name is required")
	}
	if args.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	// if args.Action == "" {
	// 	return fmt.Errorf("action is required")
	// }
	// TODO change to nil and the struct for using a pointer
	// if args.PullRequestId == 0 {
	// 	return fmt.Errorf("pull request ID is required")
	// }
	return nil
}

// validateFetch checks the necessary fields in fetchArgs and returns errors if they are not adequately set.
func (g *VCSBitbucket) validateFetch(args *shared.VCSFetchRequest) error {
	// Validate basic fields in fetchArgs, like non-empty repository URL
	if args.CloneURL == "" {
		return fmt.Errorf("repository URL is required")
	}
	if args.AuthType == "" {
		return fmt.Errorf("authentication type is required")
	}
	if args.TargetFolder == "" {
		return fmt.Errorf("target folder is required")
	}
	if args.Mode == "" {
		return fmt.Errorf("mode is required")
	}
	// TODO param validation
	// if fetchArgs.RepoParam = nil  {
	// 	return fmt.Errorf("repository URL is required")
	// }

	switch args.AuthType {
	case "ssh-key":
		if len(g.globalConfig.BitbucketPlugin.SSHKeyPassword) == 0 {
			return fmt.Errorf("SSH key password is required for SSH-key authentication")
		}
	case "http":
		if err := g.validateCommonCredentials(); err != nil {
			return err
		}
	}
	return nil
}

// validateList checks the necessary fields for listing repositories and ensures they are set.
func (g *VCSBitbucket) validateList(args *shared.VCSListReposRequest) error {
	if args.VCSURL == "" {
		return fmt.Errorf("repository URL is required")
	}

	if err := g.validateCommonCredentials(); err != nil {
		return err
	}
	return nil
}

// validateRetrivePRInformation checks the necessary fields for listing repositories and ensures they are set.
func (g *VCSBitbucket) validateRetrievePRInformation(args *shared.VCSRetrievePRInformationRequest) error {
	if err := g.validateBaseArgs(&args.VCSRequestBase); err != nil {
		return err
	}

	if err := g.validateCommonCredentials(); err != nil {
		return err
	}
	return nil
}

// validateAddRoleToPR checks the necessary fields for listing repositories and ensures they are set.
func (g *VCSBitbucket) validateAddRoleToPR(args *shared.VCSAddRoleToPRRequest) error {
	if err := g.validateBaseArgs(&args.VCSRequestBase); err != nil {
		return err
	}

	if args.Login == "" {
		return fmt.Errorf("login is required")
	}
	if args.Role == "" {
		return fmt.Errorf("role is required")
	}

	if err := g.validateCommonCredentials(); err != nil {
		return err
	}
	return nil
}

// validateSetStatusOfPR checks the necessary fields for listing repositories and ensures they are set.
func (g *VCSBitbucket) validateSetStatusOfPR(args *shared.VCSSetStatusOfPRRequest) error {
	if err := g.validateBaseArgs(&args.VCSRequestBase); err != nil {
		return err
	}

	if args.Login == "" {
		return fmt.Errorf("login is required")
	}
	if args.Status == "" {
		return fmt.Errorf("status is required")
	}

	if err := g.validateCommonCredentials(); err != nil {
		return err
	}
	return nil
}

// validateAddComment checks the necessary fields for listing repositories and ensures they are set.
func (g *VCSBitbucket) validateAddCommentToPR(args *shared.VCSAddCommentToPRRequest) error {
	if err := g.validateBaseArgs(&args.VCSRequestBase); err != nil {
		return err
	}

	if args.Comment == "" {
		return fmt.Errorf("comme is required")
	}

	if err := g.validateCommonCredentials(); err != nil {
		return err
	}
	return nil
}
