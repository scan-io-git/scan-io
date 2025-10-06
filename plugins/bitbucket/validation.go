package main

import (
	"fmt"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/validation"
)

// validateCommonCredentials checks for the presence of common credentials.
func (g *VCSBitbucket) validateCommonCredentials() error {
	return validation.ValidateCommonCredentials(g.globalConfig.BitbucketPlugin.Username, g.globalConfig.BitbucketPlugin.Token)
}

// validateFetch checks the necessary fields in VCSFetchRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateFetch(args *shared.VCSFetchRequest) error {
	if err := validation.ValidateFetchArgs(args); err != nil {
		return err
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

// validateList checks the necessary fields in VCSListRepositoriesRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateList(args *shared.VCSListRepositoriesRequest) error {
	if err := validation.ValidateListArgs(args); err != nil {
		return err
	}
	return g.validateCommonCredentials()
}

// validateRetrievePRInformation checks the necessary fields in VCSRetrievePRInformationRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateRetrievePRInformation(args *shared.VCSRetrievePRInformationRequest) error {
	if err := validation.ValidateRetrievePRInformationArgs(args); err != nil {
		return err
	}
	return g.validateCommonCredentials()
}

// validateAddRoleToPR checks the necessary fields in VCSAddRoleToPRRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateAddRoleToPR(args *shared.VCSAddRoleToPRRequest) error {
	roles := []string{"reviewer"}
	if err := validation.ValidateAddRoleToPRArgs(args, roles); err != nil {
		return err
	}
	return g.validateCommonCredentials()
}

// validateSetStatusOfPR checks the necessary fields in VCSSetStatusOfPRRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateSetStatusOfPR(args *shared.VCSSetStatusOfPRRequest) error {
	statuses := []string{"unapproved", "needs_work", "approved"}
	requiredFields := map[string]string{
		"login":  args.Login,
		"status": args.Status,
	}

	if err := validation.ValidateSetStatusOfPRArgs(args, requiredFields, statuses); err != nil {
		return err
	}

	return g.validateCommonCredentials()
}

// validateAddCommentToPR checks the necessary fields in VCSAddCommentToPRRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateAddCommentToPR(args *shared.VCSAddCommentToPRRequest) error {
	if err := validation.ValidateAddCommentToPRArgs(args); err != nil {
		return err
	}
	return g.validateCommonCredentials()
}

// validateAddSarifComments checks the necessary fields in VCSAddInLineCommentsListRequest and returns errors if they are not set.
func (g *VCSBitbucket) validateAddSarifComments(req *shared.VCSAddInLineCommentsListRequest) error {
	if err := validation.ValidateAddInLineCommentsListArgs(req); err != nil {
		return err
	}
	return g.validateCommonCredentials()
}
