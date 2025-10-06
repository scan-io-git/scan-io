package sarifcomments

import (
	"fmt"
	"strings"

	cmdutil "github.com/scan-io-git/scan-io/internal/cmd"
	"github.com/scan-io-git/scan-io/internal/vcsintegrator"
)

// validateSarifCommentsArgs validates the required command options for the selected mode.
func validateSarifCommentsArgs(options *vcsintegrator.RunOptionsIntegrationVCS, args []string, mode string) error {
	var (
		missing []string
		issues  []string
	)

	switch mode {
	case cmdutil.ModeSingleURL:
		if len(args) != 1 {
			issues = append(issues, "provide exactly one repository URL")
		}
		if strings.TrimSpace(options.VCSPluginName) == "" {
			missing = append(missing, "vcs")
		}
	case cmdutil.ModeFlags:
		if len(args) > 0 {
			issues = append(issues, fmt.Sprintf("unexpected positional arguments: %s", strings.Join(args, ", ")))
		}
		if strings.TrimSpace(options.VCSPluginName) == "" {
			missing = append(missing, "vcs")
		}
		if strings.TrimSpace(options.Domain) == "" {
			missing = append(missing, "domain")
		}
		if strings.TrimSpace(options.Namespace) == "" {
			missing = append(missing, "namespace")
		}
		if strings.TrimSpace(options.Repository) == "" {
			missing = append(missing, "repository")
		}
		if strings.TrimSpace(options.PullRequestID) == "" {
			missing = append(missing, "pull-request-id")
		}
	default:
		issues = append(issues, fmt.Sprintf("invalid sarif comments mode: %q", mode))
	}

	if strings.TrimSpace(options.SarifInput) == "" {
		missing = append(missing, "sarif")
	}
	if strings.TrimSpace(options.SourceFolder) == "" {
		missing = append(missing, "source")
	}

	if len(missing) > 0 {
		issues = append(issues, fmt.Sprintf("missing required flags: %s", strings.Join(missing, ", ")))
	}

	if options.SarifIssuesLimit < 0 {
		issues = append(issues, "'limit' cannot be negative")
	}

	if len(issues) > 0 {
		return fmt.Errorf(strings.Join(issues, "; "))
	}

	return nil
}
