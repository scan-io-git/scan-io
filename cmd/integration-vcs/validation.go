package integrationvcs

import (
	"fmt"
	"net/url"

	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/internal/vcsintegrator"
)

// validateIntegrationVCSArgs validates the arguments provided to the list command.
func validateIntegrationVCSArgs(options *vcsintegrator.RunOptionsIntegrationVCS, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("invalid argument(s) received, only one positional argument is allowed")
	}

	if options.VCSPluginName == "" {
		return fmt.Errorf("the 'vcs' flag must be specified")
	}

	if options.Action == "" {
		return fmt.Errorf("the 'action' flag must be specified")
	}

	if len(args) == 1 {
		if options.Domain != "" || options.Namespace != "" || options.PullRequestID != "" {
			return fmt.Errorf("you cannot use 'domain', 'namespace', and 'pull-request-id' flags with a target URL")
		}
		if _, err := url.ParseRequestURI(args[0]); err != nil {
			return fmt.Errorf("provided URL is not valid: %w", err)
		}
		return nil
	} else if options.Domain == "" {
		return fmt.Errorf("the 'domain' flag must be specified")
	}

	switch options.Action {
	case vcsintegrator.VCSCheckPR:
		return validateCommonArgs(options, args)
	case vcsintegrator.VCSCommentPR:
		if err := validateCommonArgs(options, args); err != nil {
			return err
		}
		if options.Comment == "" && options.CommentFile == "" {
			return fmt.Errorf("either 'comment' or 'comment-file' flag must be specified")
		}
		if options.Comment != "" && options.CommentFile != "" {
			return fmt.Errorf("only one of 'comment' or 'comment-file' flag can be specified, not both")
		}
	case vcsintegrator.VCSAddRoleToPR:
		if err := validateCommonArgs(options, args); err != nil {
			return err
		}
		if options.Login == "" {
			return fmt.Errorf("the 'login' flag must be specified")
		}
		if options.Role == "" {
			return fmt.Errorf("the 'role' flag must be specified")
		}
	case vcsintegrator.VCSSetStatusOfPR:
		if err := validateCommonArgs(options, args); err != nil {
			return err
		}
		if options.Status == "" {
			return fmt.Errorf("the 'status' flag must be specified")
		}
		if options.LocalTipCommit != "" {
			normalized, err := git.NormalizeFullHash(options.LocalTipCommit)
			if err != nil {
				return fmt.Errorf("invalid --require-head-sha value: %w", err)
			}
			options.LocalTipCommit = normalized
		}
	default:
		return fmt.Errorf("the action '%v' is not implemented", options.Action)
	}

	return nil
}

// validateCommonArgs validates the common arguments for PR actions.
func validateCommonArgs(options *vcsintegrator.RunOptionsIntegrationVCS, args []string) error {
	if len(args) != 1 && (options.Namespace == "" || options.Repository == "" || options.PullRequestID == "") {
		return fmt.Errorf("the 'namespace', 'repository', and 'pull-request-id' flags must be specified")
	}
	return nil
}
