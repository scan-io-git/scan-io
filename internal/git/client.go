package git

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	crssh "golang.org/x/crypto/ssh"

	"github.com/hashicorp/go-hclog"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
)

// GitClient represents a Git client with configuration and authentication.
type Client struct {
	logger       hclog.Logger
	auth         transport.AuthMethod
	timeout      time.Duration
	globalConfig *config.Config
}

// Authenticator defines an interface for different authentication methods.
type Authenticator interface {
	SetupAuth(args *shared.VCSFetchRequest, config map[string]string, logger hclog.Logger) (transport.AuthMethod, error)
	ValidateConfig(config map[string]string) error
}

// SSHKeyAuthenticator provides SSH key-based authentication.
type SSHKeyAuthenticator struct{}

// SSHAgentAuthenticator provides SSH agent-based authentication.
type SSHAgentAuthenticator struct{}

// HTTPAuthenticator provides HTTP basic authentication.
type HTTPAuthenticator struct{}

// SetupAuth configures SSH key authentication.
func (s *SSHKeyAuthenticator) SetupAuth(args *shared.VCSFetchRequest, config map[string]string, logger hclog.Logger) (transport.AuthMethod, error) {
	logger.Debug("setting up SSH key authentication")

	var auth transport.AuthMethod
	sshKeyPath, err := files.ExpandPath(args.SSHKey)
	if err != nil {
		logger.Error("failed to expand SSH key path", "path", args.SSHKey, "error", err)
		return nil, err
	}

	auth, err = ssh.NewPublicKeysFromFile("git", sshKeyPath, config["SSHKeyPassword"])
	if err != nil {
		logger.Error("failed to set up SSH key authentication", "error", err.Error())
		return nil, err
	}

	auth.(*ssh.PublicKeys).HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
		HostKeyCallback: crssh.InsecureIgnoreHostKey(), // TODO: Fix this
	}

	return auth, nil
}

// SetupAuth configures SSH agent authentication.
func (s *SSHAgentAuthenticator) SetupAuth(args *shared.VCSFetchRequest, config map[string]string, logger hclog.Logger) (transport.AuthMethod, error) {
	logger.Debug("setting up SSH agent authentication")

	var auth transport.AuthMethod
	var err error
	auth, err = ssh.NewSSHAgentAuth("git")
	if err != nil {
		logger.Error("failed to set up SSH agent authentication", "error", err)
		return nil, err
	}

	auth.(*ssh.PublicKeysCallback).HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
		HostKeyCallback: crssh.InsecureIgnoreHostKey(), // TODO: Fix this
	}

	return auth, nil
}

// ValidateConfig validates the configuration for SSHKeyAuthenticator.
func (s *SSHKeyAuthenticator) ValidateConfig(config map[string]string) error {
	if _, ok := config["SSHKeyPassword"]; !ok {
		return fmt.Errorf("SSHKeyPassword is required for SSHKeyAuthenticator")
	}
	return nil
}

// ValidateConfig validates the configuration for SSHAgentAuthenticator.
func (s *SSHAgentAuthenticator) ValidateConfig(config map[string]string) error {
	return nil
}

// SetupAuth configures HTTP basic authentication.
func (h *HTTPAuthenticator) SetupAuth(args *shared.VCSFetchRequest, config map[string]string, logger hclog.Logger) (transport.AuthMethod, error) {
	logger.Debug("setting up HTTP authentication")

	return &http.BasicAuth{
		Username: config["Username"],
		Password: config["Token"],
	}, nil
}

// ValidateConfig validates the configuration for HTTPAuthenticator.
func (h *HTTPAuthenticator) ValidateConfig(config map[string]string) error {
	if _, ok := config["Username"]; !ok {
		return fmt.Errorf("username is required for HTTPAuthenticator")
	}
	if _, ok := config["Token"]; !ok {
		return fmt.Errorf("token is required for HTTPAuthenticator")
	}
	return nil
}

// getAuthenticator returns the appropriate Authenticator based on the authentication type.
func getAuthenticator(authType string) (Authenticator, error) {
	switch authType {
	case "ssh-key":
		return &SSHKeyAuthenticator{}, nil
	case "ssh-agent":
		return &SSHAgentAuthenticator{}, nil
	case "http":
		return &HTTPAuthenticator{}, nil
	default:
		return nil, fmt.Errorf("unknown auth type: %s", authType)
	}
}

// New initializes a new Git Client with the given parameters.
func New(logger hclog.Logger, globalConfig *config.Config, pluginConfig interface{}, args *shared.VCSFetchRequest) (*Client, error) {

	cfg, ok := pluginConfig.(map[string]string)
	if !ok {
		return nil, fmt.Errorf("invalid config type for Git client")
	}

	authenticator, err := getAuthenticator(args.AuthType)
	if err != nil {
		logger.Error("unsupported authentication type", "error", err)
		return nil, fmt.Errorf("unsupported authentication type: %w", err)
	}

	// Validate the configuration
	if err := authenticator.ValidateConfig(cfg); err != nil {
		logger.Error("invalid configuration", "error", err)
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	auth, err := authenticator.SetupAuth(args, cfg, logger)
	if err != nil {
		logger.Error("failed to set up Git authentication", "error", err)
		return nil, fmt.Errorf("failed to set up Git authentication: %w", err)
	}

	timeout := config.SetThen(globalConfig.GitClient.Timeout, time.Duration(10*time.Minute))

	return &Client{
		logger:       logger,
		auth:         auth,
		timeout:      timeout,
		globalConfig: globalConfig,
	}, nil
}
