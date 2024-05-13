package config

import (
	"fmt"
	"os"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// Config holds configuration settings parsed from a YAML config file.
type Config struct {
	BitbucketPlugin BitbucketPlugin `yaml:"bitbucket_plugin"` // Bitbucket plugin configuration settings.
	Logger          Logger          `yaml:"logger"`           // Logger configuration settings.
	HttpClient      HttpClient      `yaml:"http_client"`      // HTTP client configuration settings.
	GitClient       GitClient       `yaml:"git_client"`       // Git client configuration settings.
}

// BitbucketPlugin holds configuration specific to the Bitbucket plugin.
type BitbucketPlugin struct {
	BitbucketUsername string `yaml:"bitbucket_username"` // Username for Bitbucket integrations. No core-level validation needed.
	BitbucketToken    string `yaml:"bitbucket_token"`    // Access token for Bitbucket. No core-level validation needed.
	SSHKeyPassword    string `yaml:"ssh_key_password"`   // Password for the SSH key used in fetching operations. No core-level validation needed.
}

// Logger configures the hclog logging aspects of the application.
type Logger struct {
	Level           string `yaml:"level"`            // Logging level (e.g., DEBUG, INFO, WARN). Validated by logger.NewLogger.
	DisableTime     *bool  `yaml:"disable_time"`     // Flag to disable timestamp logging if true. No core-level validation needed.
	JSONFormat      *bool  `yaml:"json_format"`      // Flag to output logs in JSON format if true. No core-level validation needed.
	IncludeLocation *bool  `yaml:"include_location"` // Flag to include file and line number in logs if true. No core-level validation needed.
}

// HttpClient configures settings for the HTTP client used within the application.
type HttpClient struct {
	RetryCount       int             `yaml:"retry_count"`         // The number of times to retry an HTTP request before failing. Validated by config.HttpGlobalConfigValidation.
	RetryWaitTime    time.Duration   `yaml:"retry_wait_time"`     // The duration to wait before the first retry of a failed HTTP request. Validated by config.HttpGlobalConfigValidation.
	RetryMaxWaitTime time.Duration   `yaml:"retry_max_wait_time"` // The maximum duration to wait before subsequent retries of a failed HTTP request. Validated by config.HttpGlobalConfigValidation.
	Timeout          time.Duration   `yaml:"timeout"`             // The maximum duration for the HTTP request before timing it out. Validated by config.HttpGlobalConfigValidation.
	TlsClientConfig  TlsClientConfig `yaml:"tls_client_config"`   // TLS configuration for HTTPS connections. Not validated.
	Proxy            Proxy           `yaml:"proxy"`               // A proxy configuration. Validated by config.HttpGlobalConfigValidation.
}

// TlsClientConfig configures the TLS aspects of HTTP connections.
type TlsClientConfig struct {
	Verify *bool `yaml:"verify"` // Flag to verify SSL certificates if true.
}

// Proxy defines the parameters to set up a proxy settings for HTTP connections.
type Proxy struct {
	Host string `yaml:"host"` // Hostname or IP address of the proxy server with a scheme or without.
	Port int    `yaml:"port"` // Port number of the proxy server.
}

type GitClient struct {
	Depth       int   `yaml:"depth"`        // Level of depth for cloning and fetching.
	InsecureTls *bool `yaml:"insecure_tls"` // Flag to skip SSL certificates if true.
	// CABundle
}

// LoadConfig reads a YAML config file and decodes it into a Config struct.
func LoadConfig(configPath string) (*Config, error) {
	var appConfig *Config

	if err := ValidateConfigPath(configPath); err != nil {
		return nil, err
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err = d.Decode(&appConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return appConfig, nil
}

// ValidateConfigPath checks if the given path is a valid file path for reading the configuration.
// It returns an error if the file does not exist, is a directory, or is not a regular file.
func ValidateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("config path stat error: %w", err)
	}
	if s.IsDir() {
		return fmt.Errorf("config path '%s' is a directory, not a file", path)
	}

	if s.Mode()&os.ModeType != 0 {
		return fmt.Errorf("config path '%s' is not a regular file", path)
	}
	return nil
}
