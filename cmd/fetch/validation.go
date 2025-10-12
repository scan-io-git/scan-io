package fetch

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"net/url"
	"os"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
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

	if options.AuthType == AuthTypeSSHKey && options.SSHKey == "" {
		return fmt.Errorf("you must specify ssh-key with auth-type 'ssh-key'")
	}

	if options.AuthType == AuthTypeSSHKey && options.SSHKey != "" {
		expandedPath, err := files.ExpandPath(options.SSHKey)
		if err != nil {
			return fmt.Errorf("failed to expand path %q: %w", options.SSHKey, err)
		}

		if err := files.ValidatePath(expandedPath); err != nil {
			return fmt.Errorf("failed to validate path %q: %w", expandedPath, err)
		}

		keyData, err := os.ReadFile(expandedPath)
		if err != nil {
			return fmt.Errorf("failed to read SSH key file: %w", err)
		}

		_, err = ssh.ParsePrivateKey(keyData)
		if err == nil {
			return nil
		}

		if _, ok := err.(*ssh.PassphraseMissingError); !ok {
			return fmt.Errorf("invalid SSH key format: %w", err)
		}

		// TODO: the check takes pass only from the global config and ignores env variables for plugins
		// should be fixed with moving to Viper
		// _, err = ssh.ParsePrivateKeyWithPassphrase(keyData, pass)
		// if err != nil {
		// 	return fmt.Errorf("invalid SSH key format or incorrect passphrase: %w", err)
		// }
		return nil
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
