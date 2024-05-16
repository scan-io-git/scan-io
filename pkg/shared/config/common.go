package config

import (
	"crypto/tls"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// BaseHTTPConfig holds common HTTP client configuration settings.
type BaseHTTPConfig struct {
	RetryCount       int           // Number of retries for failed requests
	RetryWaitTime    time.Duration // Wait time between retries
	RetryMaxWaitTime time.Duration // Maximum wait time for retries
	Timeout          time.Duration // Timeout for requests
	TLSClientConfig  *tls.Config   // TLS configuration
	Proxy            string        // Proxy address
}

// RestyHTTPClientConfig holds additional configuration settings for the Resty HTTP client.
type RestyHTTPClientConfig struct {
	BaseHTTPConfig
	Debug bool // Flag to enable Resty debug mode
}

// DefaultHTTPConfig returns a base configuration for HTTP clients with default values.
func DefaultHTTPConfig() BaseHTTPConfig {
	return BaseHTTPConfig{
		RetryCount:       5,
		RetryWaitTime:    1 * time.Second,
		RetryMaxWaitTime: 5 * time.Second,
		Timeout:          30 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12, // Enforce a minimum TLS version
			InsecureSkipVerify: false,            // Ensure TLS certificates are verified
		},
		Proxy: "", // No proxy by default
	}
}

// DefaultRestyConfig returns a default configuration for the Resty HTTP client, extending the base HTTP configuration.
func DefaultRestyConfig() RestyHTTPClientConfig {
	baseConfig := DefaultHTTPConfig()
	return RestyHTTPClientConfig{
		BaseHTTPConfig: baseConfig,
		Debug:          false, // Debug mode is disabled by default
	}
}

// GetScanioHome returns the Scanio home directory from the configuration.
func GetScanioHome(cfg *Config) string {
	return cfg.Scanio.HomeFolder
}

// GetScanioPluginsHome returns the Scanio plugins directory from the configuration.
func GetScanioPluginsHome(cfg *Config) string {
	return cfg.Scanio.PluginsFolder
}

// GetScanioProjectsHome returns the Scanio projects directory from the configuration.
func GetScanioProjectsHome(cfg *Config) string {
	return cfg.Scanio.ProjectsFolder
}

// GetScanioResultsHome returns the Scanio results directory from the configuration.
func GetScanioResultsHome(cfg *Config) string {
	return cfg.Scanio.ResultsFolder
}

// GetScanioTempHome returns the Scanio temporary directory from the configuration.
func GetScanioTempHome(cfg *Config) string {
	return cfg.Scanio.TempFolder
}

// GetRepositoryPath constructs the path to a repository based on the VCS URL and repository namespace.
func GetRepositoryPath(cfg *Config, VCSURL, repoWithNamespace string) string {
	return filepath.Join(GetScanioProjectsHome(cfg), strings.ToLower(VCSURL), strings.ToLower(repoWithNamespace))
}

// GetPRTempPath constructs the path to the temporary folder for a pull request based on the VCS URL, namespace, and repository name.
func GetPRTempPath(cfg *Config, VCSURL, Namespace, RepoName string, PRId int) string {
	rawStartTime := time.Now().UTC()
	startTime := rawStartTime.Format(time.RFC3339)
	return filepath.Join(GetScanioTempHome(cfg), strings.ToLower(VCSURL), strings.ToLower(Namespace),
		strings.ToLower(RepoName), "scanio-pr-tmp", strconv.Itoa(PRId), startTime)
}
