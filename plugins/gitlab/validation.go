package main

import (
	"fmt"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/validation"
)

// validateCommonCredentials checks for the presence of common credentials.
func (g *VCSGitlab) validateCommonCredentials() error {
	return validation.ValidateCommonCredentials(g.globalConfig.GitlabPlugin.Username, g.globalConfig.GitlabPlugin.Token)
}

// validateAPICommonCredentials checks for the presence of common credentials for APU.
func (g *VCSGitlab) validateAPICommonCredentials() error {
	if len(g.globalConfig.GitlabPlugin.Token) == 0 {
		g.logger.Warn("No token provided. Anonymous Git access will be used. API rate limits may apply.")
	}
	return nil
}

// validateFetch checks the necessary fields in VCSFetchRequest and returns errors if they are not set.
func (g *VCSGitlab) validateFetch(args *shared.VCSFetchRequest) error {
	if err := validation.ValidateFetchArgs(args); err != nil {
		return err
	}

	switch args.AuthType {
	case "ssh-key":
		if g.globalConfig.GitlabPlugin.SSHKeyPassword == "" {
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
func (g *VCSGitlab) validateList(args *shared.VCSListRepositoriesRequest) error {
	if err := validation.ValidateListArgs(args); err != nil {
		return err
	}
	return g.validateAPICommonCredentials()
}

// validateRetrievePRInformation checks the necessary fields in VCSRetrievePRInformationRequest and returns errors if they are not set.
func (g *VCSGitlab) validateRetrievePRInformation(args *shared.VCSRetrievePRInformationRequest) error {
	if err := validation.ValidateRetrievePRInformationArgs(args); err != nil {
		return err
	}
	return g.validateAPICommonCredentials()
}

// validateAddRoleToPR checks the necessary fields in VCSAddRoleToPRRequest and returns errors if they are not set.
func (g *VCSGitlab) validateAddRoleToPR(args *shared.VCSAddRoleToPRRequest) error {
	roles := []string{"assignee", "reviewer"}
	if err := validation.ValidateAddRoleToPRArgs(args, roles); err != nil {
		return err
	}
	return g.validateAPICommonCredentials()
}

// validateSetStatusOfPR checks the necessary fields in VCSSetStatusOfPRRequest and returns errors if they are not set.
func (g *VCSGitlab) validateSetStatusOfPR(args *shared.VCSSetStatusOfPRRequest) error {
	statuses := []string{"approve", "unapprove"}
	requiredFields := map[string]string{
		"status": args.Status,
	}

	if err := validation.ValidateSetStatusOfPRArgs(args, requiredFields, statuses); err != nil {
		return err
	}
	return g.validateAPICommonCredentials()
}

// validateAddCommentToPR checks the necessary fields in VCSAddCommentToPRRequest and returns errors if they are not set.
func (g *VCSGitlab) validateAddCommentToPR(args *shared.VCSAddCommentToPRRequest) error {
	if err := validation.ValidateAddCommentToPRArgs(args); err != nil {
		return err
	}
	return g.validateAPICommonCredentials()
}

// validateAddSarifComments checks the necessary fields in VCSAddInLineCommentsListRequest and returns errors if they are not set.
func (g *VCSGitlab) validateAddSarifComments(req *shared.VCSAddInLineCommentsListRequest) error {
	if err := validation.ValidateAddInLineCommentsListArgs(req); err != nil {
		return err
	}
	return g.validateCommonCredentials()
}
