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
func validateFetchArgs(options *RunOptionsFetch, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("invalid argument(s) received, only one positional argument is allowed")
	}

	if options.VCSPluginName == "" {
		return fmt.Errorf("the 'vcs' flag must be specified")
	}

	if options.AuthType == "" {
		return fmt.Errorf("the 'auth-type' flag must be specified")
	}

	authTypesList := []string{AuthTypeHTTP, AuthTypeSSHKey, AuthTypeSSHAgent}
	if !shared.IsInList(options.AuthType, authTypesList) {
		return fmt.Errorf("unknown auth-type: %v", options.AuthType)
	}

	// TODO: add SSHKey verification
	if options.AuthType == AuthTypeSSHKey && options.SSHKey == "" {
		return fmt.Errorf("you must specify ssh-key with auth-type 'ssh-key'")
	}

	if len(args) == 0 && options.InputFile == "" {
		return fmt.Errorf("either 'input-file' flag or a target URL must be specified")
	}

	if options.InputFile != "" && len(args) != 0 {
		return fmt.Errorf("you cannot use 'input-file' flag with a target URL")
	}

	if len(args) == 1 {
		_, err := url.ParseRequestURI(args[0])
		if err != nil {
			return fmt.Errorf("provided URL in not valid: %w", err)
		}
		return nil
	}

	// TODO: add validation for the input file format
	if options.InputFile == "" {
		return fmt.Errorf("the 'input-file' flag must be specified")
	}

	if options.Threads <= 0 {
		return fmt.Errorf("the 'threads' flag must be a positive integer")
	}

	return nil
}

// validationRepoInfo validates the provided RepositoryParams struct.
func validationRepoInfo(repo shared.RepositoryParams) error {
	if repo.Namespace == "" {
		return fmt.Errorf("fetch all projects across VCS is not supported. Use the list command first")
	}
	if repo.Repository == "" {
		return fmt.Errorf("fetch an entire project is not supported. Use the list command first")
	}
	return nil
}
