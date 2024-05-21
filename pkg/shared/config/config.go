package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/scan-io-git/scan-io/pkg/shared/files"
)

// Config holds configuration settings parsed from a YAML config file.
type Config struct {
	Scanio          Scanio          `yaml:"scanio"`           // Scanio configuration settings.
	BitbucketPlugin BitbucketPlugin `yaml:"bitbucket_plugin"` // Bitbucket plugin configuration settings.
	Logger          Logger          `yaml:"logger"`           // Logger configuration settings.
	HTTPClient      HTTPClient      `yaml:"http_client"`      // HTTP client configuration settings.
	GitClient       GitClient       `yaml:"git_client"`       // Git client configuration settings.
}

// Scanio holds configuration specific to the Scanio application.
type Scanio struct {
	Mode           string `yaml:"mode"`            // Scanio mode cofiguration.
	HomeFolder     string `yaml:"home_folder"`     // The home directory for Scanio.
	PluginsFolder  string `yaml:"plugins_folder"`  // The directory where Scanio plugins are stored.
	ProjectsFolder string `yaml:"projects_folder"` // The directory where Scanio project files are stored.
	ResultsFolder  string `yaml:"results_folder"`  // The directory where Scanio results are stored.
	TempFolder     string `yaml:"temp_folder"`     // The directory for temporary files used by Scanio.

}

// BitbucketPlugin holds configuration specific to the Bitbucket plugin.
type BitbucketPlugin struct {
	Username       string `yaml:"username"`         // Username for Bitbucket integrations.
	Token          string `yaml:"token"`            // Access token for Bitbucket.
	SSHKeyPassword string `yaml:"ssh_key_password"` // Password for the SSH key used in fetching operations.
}

// Logger configures the hclog logging aspects of the application.
type Logger struct {
	Level           string `yaml:"level"`            // Logging level (e.g., DEBUG, INFO, WARN).
	DisableTime     *bool  `yaml:"disable_time"`     // Flag to disable timestamp logging if true.
	JSONFormat      *bool  `yaml:"json_format"`      // Flag to output logs in JSON format if true.
	IncludeLocation *bool  `yaml:"include_location"` // Flag to include file and line number in logs if true.
}

// HTTPClient configures settings for the HTTP client used within the application.
type HTTPClient struct {
	RetryCount       int             `yaml:"retry_count"`         // The number of times to retry an HTTP request before failing.
	RetryWaitTime    time.Duration   `yaml:"retry_wait_time"`     // The duration to wait before the first retry of a failed HTTP request.
	RetryMaxWaitTime time.Duration   `yaml:"retry_max_wait_time"` // The maximum duration to wait before subsequent retries of a failed HTTP request.
	Timeout          time.Duration   `yaml:"timeout"`             // The maximum duration for the HTTP request before timing it out.
	TLSClientConfig  TLSClientConfig `yaml:"tls_client_config"`   // TLS configuration for HTTPS connections.
	Proxy            Proxy           `yaml:"proxy"`               // A proxy configuration.
}

// TLSClientConfig configures the TLS aspects of HTTP connections.
type TLSClientConfig struct {
	Verify *bool `yaml:"verify"` // Flag to verify SSL certificates if true.
}

// Proxy defines the parameters to set up proxy settings for HTTP connections.
type Proxy struct {
	Host string `yaml:"host"` // Hostname or IP address of the proxy server with a scheme or without.
	Port int    `yaml:"port"` // Port number of the proxy server.
}

// GitClient configures settings for Git operations.
type GitClient struct {
	Depth       int           `yaml:"depth"`        // Level of depth for cloning and fetching.
	InsecureTLS *bool         `yaml:"insecure_tls"` // Flag to skip SSL certificates if true.
	Timeout     time.Duration `yaml:"timeout"`      // The maximum duration for the Git request before timing it out.
	// CABundle
}

// LoadConfig reads a YAML config file and decodes it into a Config struct.
// If configPath is empty, it searches for a config file in default paths.
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{}

	if configPath != "" {
		if err := config.loadConfig(configPath); err != nil {
			return config, err
		}
	} else {
		if err := config.searchDefaultConfig(); err != nil {
			return config, err
		}
	}
	return config, nil
}

// searchDefaultConfig searches for a config file in default paths.
func (c *Config) searchDefaultConfig() error {
	defaultPaths := []string{
		"config.yml",           // Development default path
		"~/.scanio/config.yml", // Local install default path
		"/scanio/config.yml",   // Docker default path
	}

	var lastErr error
	for _, path := range defaultPaths {
		if err := c.loadConfig(path); err == nil {
			return nil
		} else {
			lastErr = fmt.Errorf("failed to load config from path '%s': %w", path, err)
		}
	}
	return fmt.Errorf("no valid config file found in default paths: %w", lastErr)
}

// loadConfig reads and parses the YAML config file at the given path.
func (c *Config) loadConfig(path string) error {
	expandedPath, err := files.ExpandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path '%s': %w", path, err)
	}

	if err := files.ValidatePath(expandedPath); err != nil {
		return fmt.Errorf("failed to validate path '%s': %w", expandedPath, err)
	}

	file, err := os.Open(expandedPath)
	if err != nil {
		return fmt.Errorf("failed to read config file '%s': %w", expandedPath, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err = decoder.Decode(&c); err != nil {
		return fmt.Errorf("failed to unmarshal config '%s': %w", expandedPath, err)
	}
	return nil
}
