package validation

import (
	"fmt"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

// ValidateCommonCredentials checks for the presence of common credentials.
func ValidateCommonCredentials(username, token string) error {
	if len(username) == 0 || len(token) == 0 {
		return fmt.Errorf("both username and token are required")
	}
	return nil
}

// ValidateBaseArgs checks the common fields in VCSRequestBase and returns errors if they are not set.
func ValidateBaseArgs(args *shared.VCSRequestBase) error {
	requiredFields := map[string]string{
		"repository URL": args.RepoParam.Domain,
		"namespace":      args.RepoParam.Namespace,
		"repository":     args.RepoParam.Repository,
		"Action":         args.Action,
		"PullRequestID":  args.RepoParam.PullRequestID,
	}

	for field, value := range requiredFields {
		if value == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	return nil
}

// ValidateFetchArgs checks the necessary fields in VCSFetchRequest and returns errors if they are not set.
func ValidateFetchArgs(args *shared.VCSFetchRequest) error {
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

	return nil
}

// ValidateListArgs checks the necessary fields in VCSListRepositoriesRequest and returns errors if they are not set.
func ValidateListArgs(args *shared.VCSListRepositoriesRequest) error {
	if args.RepoParam.Domain == "" {
		return fmt.Errorf("repository URL is required")
	}
	return nil
}

// ValidateRetrievePRInformationArgs checks the necessary fields in VCSRetrievePRInformationRequest and returns errors if they are not set.
func ValidateRetrievePRInformationArgs(args *shared.VCSRetrievePRInformationRequest) error {
	if err := ValidateBaseArgs(&args.VCSRequestBase); err != nil {
		return err
	}
	return nil
}

// ValidateAddRoleToPRArgs checks the necessary fields in VCSAddRoleToPRRequest and returns errors if they are not set.
func ValidateAddRoleToPRArgs(args *shared.VCSAddRoleToPRRequest) error {
	if err := ValidateBaseArgs(&args.VCSRequestBase); err != nil {
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
	return nil
}

// ValidateSetStatusOfPRArgs checks the necessary fields in VCSSetStatusOfPRRequest and returns errors if they are not set.
func ValidateSetStatusOfPRArgs(args *shared.VCSSetStatusOfPRRequest) error {
	if err := ValidateBaseArgs(&args.VCSRequestBase); err != nil {
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

	return nil
}

// ValidateAddCommentToPRArgs checks the necessary fields in VCSAddCommentToPRRequest and returns errors if they are not set.
func ValidateAddCommentToPRArgs(args *shared.VCSAddCommentToPRRequest) error {
	if err := ValidateBaseArgs(&args.VCSRequestBase); err != nil {
		return err
	}

	if args.Comment == "" {
		return fmt.Errorf("comment is required")
	}
	return nil
}
