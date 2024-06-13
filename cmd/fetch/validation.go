package fetch

import (
	"fmt"
	"net/url"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

const (
	AuthTypeHTTP     = "http"
	AuthTypeSSHKey   = "ssh-key"
	AuthTypeSSHAgent = "ssh-agent"
)

// validateFetchArgs validates the arguments provided to the fetch command.
func validateFetchArgs(allArgumentsFetch *RunOptionsFetch, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("invalid argument(s) received, only one positional argument is allowed")
	}

	if allArgumentsFetch.VCSPluginName == "" {
		return fmt.Errorf("the 'vcs' flag must be specified")
	}

	if allArgumentsFetch.AuthType == "" {
		return fmt.Errorf("the 'auth-type' flag must be specified")
	}

	authTypesList := []string{AuthTypeHTTP, AuthTypeSSHKey, AuthTypeSSHAgent}
	if !shared.IsInList(allArgumentsFetch.AuthType, authTypesList) {
		return fmt.Errorf("unknown auth-type: %v", allArgumentsFetch.AuthType)
	}

	// TODO: add SSHKey verification
	if allArgumentsFetch.AuthType == AuthTypeSSHKey && allArgumentsFetch.SSHKey == "" {
		return fmt.Errorf("you must specify ssh-key with auth-type 'ssh-key'")
	}

	if len(args) == 0 && allArgumentsFetch.InputFile == "" {
		return fmt.Errorf("either 'input-file' flag or a target URL must be specified")
	}

	if allArgumentsFetch.InputFile != "" && len(args) != 0 {
		return fmt.Errorf("you cannot use both 'input-file' flag and a target URL at the same time")
	}

	if len(args) == 1 {
		_, err := url.ParseRequestURI(args[0])
		if err != nil {
			return fmt.Errorf("provided URL in not valid: %w", err)
		}
		return nil
	}

	// TODO: add validation for the input file format
	if allArgumentsFetch.InputFile == "" {
		return fmt.Errorf("the 'input-file' flag must be specified")
	}

	if allArgumentsFetch.Threads <= 0 {
		return fmt.Errorf("the 'threads' flag must be a positive integer")
	}

	return nil
}
