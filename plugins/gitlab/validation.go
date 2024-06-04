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
	return g.validateCommonCredentials()
}

// validateRetrievePRInformation checks the necessary fields in VCSRetrievePRInformationRequest and returns errors if they are not set.
func (g *VCSGitlab) validateRetrievePRInformation(args *shared.VCSRetrievePRInformationRequest) error {
	if err := validation.ValidateRetrievePRInformationArgs(args); err != nil {
		return err
	}
	return g.validateCommonCredentials()
}

// validateAddRoleToPR checks the necessary fields in VCSAddRoleToPRRequest and returns errors if they are not set.
func (g *VCSGitlab) validateAddRoleToPR(args *shared.VCSAddRoleToPRRequest) error {
	if err := validation.ValidateAddRoleToPRArgs(args); err != nil {
		return err
	}
	return g.validateCommonCredentials()
}

// validateSetStatusOfPR checks the necessary fields in VCSSetStatusOfPRRequest and returns errors if they are not set.
func (g *VCSGitlab) validateSetStatusOfPR(args *shared.VCSSetStatusOfPRRequest) error {
	if err := validation.ValidateSetStatusOfPRArgs(args); err != nil {
		return err
	}
	return g.validateCommonCredentials()
}

// validateAddCommentToPR checks the necessary fields in VCSAddCommentToPRRequest and returns errors if they are not set.
func (g *VCSGitlab) validateAddCommentToPR(args *shared.VCSAddCommentToPRRequest) error {
	if err := validation.ValidateAddCommentToPRArgs(args); err != nil {
		return err
	}
	return g.validateCommonCredentials()
}
