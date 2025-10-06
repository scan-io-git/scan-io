package main

import (
	"fmt"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/validation"
)

// validateCommonCredentials checks for the presence of common credentials.
func (g *VCSGithub) validateCommonCredentials() error {
	return validation.ValidateCommonCredentials(g.globalConfig.GithubPlugin.Username, g.globalConfig.GithubPlugin.Token)
}

// validateAPICommonCredentials checks for the presence of common credentials for APU.
func (g *VCSGithub) validateAPICommonCredentials() error {
	if len(g.globalConfig.GithubPlugin.Token) == 0 {
		g.logger.Warn("No token provided. Anonymous Git access will be used. API rate limits may apply.")
	}
	return nil
}

// validateFetch checks the necessary fields in VCSFetchRequest and returns errors if they are not set.
func (g *VCSGithub) validateFetch(args *shared.VCSFetchRequest) error {
	if err := validation.ValidateFetchArgs(args); err != nil {
		return err
	}

	switch args.AuthType {
	case "ssh-key":
		if g.globalConfig.GithubPlugin.SSHKeyPassword == "" {
			return fmt.Errorf("SSH key password is required for SSH-key authentication")
		}
	case "http":
		if err := g.validateCommonCredentials(); err != nil {
			return err
		}
	}
	return nil
}

// validateList checks the necessary fields in VCSListRepositoriesRequest and returns errors if they are not set.
func (g *VCSGithub) validateList(args *shared.VCSListRepositoriesRequest) error {
	if err := validation.ValidateListArgs(args); err != nil {
		return err
	}
	return g.validateAPICommonCredentials()
}

// validateRetrievePRInformation checks the necessary fields in VCSRetrievePRInformationRequest and returns errors if they are not set.
func (g *VCSGithub) validateRetrievePRInformation(args *shared.VCSRetrievePRInformationRequest) error {
	if err := validation.ValidateRetrievePRInformationArgs(args); err != nil {
		return err
	}
	return g.validateAPICommonCredentials()
}

// validateAddRoleToPR checks the necessary fields in VCSAddRoleToPRRequest and returns errors if they are not set.
func (g *VCSGithub) validateAddRoleToPR(args *shared.VCSAddRoleToPRRequest) error {
	roles := []string{"assignee", "reviewer"}
	if err := validation.ValidateAddRoleToPRArgs(args, roles); err != nil {
		return err
	}
	return g.validateAPICommonCredentials()
}

// validateSetStatusOfPR checks the necessary fields in VCSSetStatusOfPRRequest and returns errors if they are not set.
func (g *VCSGithub) validateSetStatusOfPR(args *shared.VCSSetStatusOfPRRequest) error {
	statuses := []string{"approve", "request_changes"}
	requiredFields := map[string]string{
		"status":  args.Status,
		"comment": args.Comment,
	}

	if err := validation.ValidateSetStatusOfPRArgs(args, requiredFields, statuses); err != nil {
		return err
	}
	return g.validateAPICommonCredentials()
}

// validateAddCommentToPR checks the necessary fields in VCSAddCommentToPRRequest and returns errors if they are not set.
func (g *VCSGithub) validateAddCommentToPR(args *shared.VCSAddCommentToPRRequest) error {
	if err := validation.ValidateAddCommentToPRArgs(args); err != nil {
		return err
	}

	if len(args.Comment.FilePaths) != 0 {
		g.logger.Warn("uploading files is not supported for github plugin. Uploading will be skipped!")
	}
	return g.validateAPICommonCredentials()
}

// validateAddSarifComments checks the necessary fields in VCSAddInLineCommentsListRequest and returns errors if they are not set.
func (g *VCSGithub) validateAddSarifComments(req *shared.VCSAddInLineCommentsListRequest) error {
	if err := validation.ValidateAddInLineCommentsListArgs(req); err != nil {
		return err
	}
	return g.validateCommonCredentials()
}
