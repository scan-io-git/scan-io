package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scan-io-git/scan-io/pkg/shared/files"
)

// ValidateConfig checks if the global configurations have valid values.
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("YAML global config: configuration object is nil")
	}
	if err := ValidateScanioConfig(cfg); err != nil {
		return fmt.Errorf("YAML global config: scanio directive is invalid: %w", err)
	}
	if err := ValidateHTTPConfig(&cfg.HTTPClient); err != nil {
		return fmt.Errorf("YAML global config: http_client directive is invalid: %w", err)
	}
	if err := ValidateGitConfig(&cfg.GitClient); err != nil {
		return fmt.Errorf("YAML global config: git_client directive is invalid: %w", err)
	}
	return nil
}

// ValidateScanioConfig checks if the Scanio configurations have valid values.
func ValidateScanioConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("scanio configuration is nil")
	}
	if err := updateHome(cfg); err != nil {
		return fmt.Errorf("failed to update home folder: %w", err)
	}
	if err := updateFolder(&cfg.Scanio.PluginsFolder, "SCANIO_PLUGINS_FOLDER", "plugins", cfg); err != nil {
		return fmt.Errorf("failed to update plugins folder: %w", err)
	}
	if err := updateFolder(&cfg.Scanio.ProjectsFolder, "SCANIO_PROJECTS_FOLDER", "projects", cfg); err != nil {
		return fmt.Errorf("failed to update projects folder: %w", err)
	}
	if err := updateFolder(&cfg.Scanio.ResultsFolder, "SCANIO_RESULTS_FOLDER", "results", cfg); err != nil {
		return fmt.Errorf("failed to update results folder: %w", err)
	}
	if err := updateFolder(&cfg.Scanio.TempFolder, "SCANIO_TEMP_FOLDER", "tmp", cfg); err != nil {
		return fmt.Errorf("failed to update temp folder: %w", err)
	}
	updateMode(cfg)

	return nil
}

// ValidateGitConfig checks if the Git configurations have valid values.
func ValidateGitConfig(gitConfig *GitClient) error {
	if gitConfig == nil {
		return fmt.Errorf("git configuration is nil")
	}

	if err := validateDuration(gitConfig.Timeout, "timeout", 1*time.Hour); err != nil {
		return err
	}
	return nil
}

// ValidateHTTPConfig checks if the HTTP configurations have valid values.
func ValidateHTTPConfig(httpConfig *HTTPClient) error {
	if httpConfig == nil {
		return fmt.Errorf("HTTP configuration is nil")
	}
	if httpConfig.RetryCount < 0 || httpConfig.RetryCount > 20 {
		return fmt.Errorf("retry_count must be between 0 and 20: %d", httpConfig.RetryCount)
	}

	durations := map[string]time.Duration{
		"RetryMaxWaitTime": httpConfig.RetryMaxWaitTime,
		"RetryWaitTime":    httpConfig.RetryWaitTime,
		"Timeout":          httpConfig.Timeout,
	}
	for name, duration := range durations {
		if err := validateDuration(duration, name, 100*time.Second); err != nil {
			return err
		}
	}

	if err := validateProxy(&httpConfig.Proxy); err != nil {
		return err
	}

	return nil
}

// validateDuration checks that a time.Duration is valid and within a specified maximum duration.
func validateDuration(d time.Duration, name string, max time.Duration) error {
	if d < 0 {
		return fmt.Errorf("invalid duration for %q: %v cannot be negative", name, d)
	}
	if d > max {
		return fmt.Errorf("%q duration is too long: %v exceeds maximum of %v", name, d, max)
	}
	return nil
}

// validateProxy checks if the given Proxy settings are valid.
func validateProxy(proxy *Proxy) error {
	if proxy == nil {
		return fmt.Errorf("proxy configuration is nil")
	}

	// If host or port is not set, skip further validation
	if proxy.Host == "" || proxy.Port == 0 {
		return nil
	}

	if err := validateHost(&proxy.Host); err != nil {
		return err
	}

	if err := validatePort(proxy.Port); err != nil {
		return err
	}

	return nil
}

// validateHost checks if the host part of the proxy configuration is valid.
// It ensures the host includes a scheme; adds "http" if missing.
func validateHost(host *string) error {
	if host == nil {
		return fmt.Errorf("host string pointer is nil")
	}

	if !strings.Contains(*host, "://") {
		*host = "http://" + *host
	}
	*host = strings.TrimRight(*host, "/")

	// TODO: Add domain or IP validation
	_, err := url.Parse(*host)
	if err != nil {
		return fmt.Errorf("invalid host URL: %w", err)
	}

	return nil
}

// validatePort checks if the port part of the proxy configuration is valid.
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}

// updateHome updates the HomeFolder in the Scanio config from environment variables or sets a default value.
func updateHome(cfg *Config) error {
	if scanioHomeFolder := os.Getenv("SCANIO_HOME"); scanioHomeFolder != "" {
		cfg.Scanio.HomeFolder = scanioHomeFolder
	} else if cfg.Scanio.HomeFolder == "" {
		homeFolder, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("unable to get user home folder: %w", err)
		}
		cfg.Scanio.HomeFolder = filepath.Join(homeFolder, ".scanio")
	}

	expandedHomePath, err := files.ExpandPath(cfg.Scanio.HomeFolder)
	if err != nil {
		return fmt.Errorf("failed to expand new home path %q: %w", cfg.Scanio.HomeFolder, err)
	}
	cfg.Scanio.HomeFolder = expandedHomePath

	if err := files.CreateFolderIfNotExists(expandedHomePath); err != nil {
		return fmt.Errorf("failed to create home folder %q: %w", cfg.Scanio.HomeFolder, err)
	}
	return nil
}

// updateFolder updates a folder path in the Scanio configuration.
func updateFolder(folder *string, envVar, defaultSubFolder string, cfg *Config) error {
	if envVarValue := os.Getenv(envVar); envVarValue != "" {
		*folder = envVarValue
	} else if *folder == "" {
		*folder = filepath.Join(GetScanioHome(cfg), defaultSubFolder)
	}

	expandedHomePath, err := files.ExpandPath(*folder)
	if err != nil {
		return fmt.Errorf("failed to expand new home path %q: %w", *folder, err)
	}
	*folder = expandedHomePath

	if err := files.CreateFolderIfNotExists(expandedHomePath); err != nil {
		return fmt.Errorf("failed to create folder %q: %w", expandedHomePath, err)
	}
	return nil
}

// updateMode updates the Mode field in the Scanio configuration based on environment variables.
func updateMode(cfg *Config) {
	if os.Getenv("SCANIO_MODE") == "CI" || os.Getenv("CI") == "true" {
		cfg.Scanio.Mode = "CI"
		return
	}

	if envVarValue := os.Getenv("SCANIO_MODE"); envVarValue != "" {
		cfg.Scanio.Mode = envVarValue
		return
	}

	cfg.Scanio.Mode = "user"
}
