package main

import (
	"fmt"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

// validateCommonCredentials checks for the presence of common credentials.
func (g *VCSBitbucket) validateCommonCredentials() error {
	if len(g.globalConfig.BitbucketPlugin.Username) == 0 || len(g.globalConfig.BitbucketPlugin.Token) == 0 {
		return fmt.Errorf("both Bitbucket username and token are required")
	}
	return nil
}

// validateBaseArgs checks the common fields in VCSRequestBase and returns errors if they are not set.
func (g *VCSBitbucket) validateBaseArgs(args *shared.VCSRequestBase) error {
	requiredFields := map[string]string{
		"repository URL": args.VCSURL,
		"namespace":      args.Namespace,
		"repository":     args.Repository,
		// "Action": args.Action,
		// "PullRequestId": args.PullRequestId, // TODO: Change the struct for using a pointer
	}

	for field, value := range requiredFields {
		if value == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	return nil
}

// validateFetch checks the necessary fields in VCSFetchRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateFetch(args *shared.VCSFetchRequest) error {
	requiredFields := map[string]string{
		"repository URL":      args.CloneURL,
		"authentication type": args.AuthType,
		"target folder":       args.TargetFolder,
		"mode":                args.Mode,
		// "RepoParam": args.RepoParam, // TODO: Add params validation
	}

	for field, value := range requiredFields {
		if value == "" {
			return fmt.Errorf("%s is required", field)
		}
	}

	switch args.AuthType {
	case "ssh-key":
		if g.globalConfig.BitbucketPlugin.SSHKeyPassword == "" {
			return fmt.Errorf("SSH key password is required for SSH-key authentication")
		}
	case "http":
		if err := g.validateCommonCredentials(); err != nil {
			return err
		}
	}
	return nil
}

// validateList checks the necessary fields in VCSListReposRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateList(args *shared.VCSListRepositoriesRequest) error {
	if args.VCSURL == "" {
		return fmt.Errorf("repository URL is required")
	}
	return g.validateCommonCredentials()
}

// validateRetrievePRInformation checks the necessary fields in VCSRetrievePRInformationRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateRetrievePRInformation(args *shared.VCSRetrievePRInformationRequest) error {
	if err := g.validateBaseArgs(&args.VCSRequestBase); err != nil {
		return err
	}
	return g.validateCommonCredentials()
}

// validateAddRoleToPR checks the necessary fields in VCSAddRoleToPRRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateAddRoleToPR(args *shared.VCSAddRoleToPRRequest) error {
	if err := g.validateBaseArgs(&args.VCSRequestBase); err != nil {
		return err
	}

	requiredFields := map[string]string{
		"login": args.Login,
		"role":  args.Role,
	}

	for field, value := range requiredFields {
		if value == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	return g.validateCommonCredentials()
}

// validateSetStatusOfPR checks the necessary fields in VCSSetStatusOfPRRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateSetStatusOfPR(args *shared.VCSSetStatusOfPRRequest) error {
	if err := g.validateBaseArgs(&args.VCSRequestBase); err != nil {
		return err
	}

	requiredFields := map[string]string{
		"login":  args.Login,
		"status": args.Status,
	}

	for field, value := range requiredFields {
		if value == "" {
			return fmt.Errorf("%s is required", field)
		}
	}

	return g.validateCommonCredentials()
}

// validateAddCommentToPR checks the necessary fields in VCSAddCommentToPRRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateAddCommentToPR(args *shared.VCSAddCommentToPRRequest) error {
	if err := g.validateBaseArgs(&args.VCSRequestBase); err != nil {
		return err
	}

	if args.Comment == "" {
		return fmt.Errorf("comment is required")
	}
	return g.validateCommonCredentials()
}
