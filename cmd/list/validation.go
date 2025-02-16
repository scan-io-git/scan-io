package list

import (
	"fmt"
	"net/url"

	"github.com/scan-io-git/scan-io/internal/vcsintegrator"
)

// validateListArgs validates the arguments provided to the list command.
func validateListArgs(options *vcsintegrator.RunOptionsIntegrationVCS, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("invalid argument(s) received, only one positional argument is allowed")
	}

	if options.VCSPluginName == "" {
		return fmt.Errorf("the 'vcs' flag must be specified")
	}

	if len(args) == 1 {
		if options.Domain != "" || options.Namespace != "" {
			return fmt.Errorf("you cannot use 'domain', 'namespace' flags with a target URL")
		}
		_, err := url.ParseRequestURI(args[0])
		if err != nil {
			return fmt.Errorf("provided URL is not valid: %w", err)
		}
	} else if options.Domain == "" {
		return fmt.Errorf("the 'domain' flag must be specified")
	}

	if options.Language != "" && options.VCSPluginName != "gitlab" {
		return fmt.Errorf("the 'language' feature is supported only for the gitlab plugin")
	}

	// TODO: add a path validation
	if options.OutputPath == "" {
		return fmt.Errorf("the 'output' flag must be specified")
	}

	return nil
}
