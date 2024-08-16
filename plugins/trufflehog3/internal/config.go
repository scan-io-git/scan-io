package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
	"gopkg.in/yaml.v2"
)

// Config represents the entire YAML configuration.
type Config struct {
	Exclude        []*Exclude `yaml:"exclude"`
	Severity       string     `yaml:"severity,omitempty"`
	IgnoreNoSecret bool       `yaml:"ignore_nosecret,omitempty"`
	NoEntropy      bool       `yaml:"no_entropy,omitempty"`
	NoPattern      bool       `yaml:"no_pattern,omitempty"`
	Branch         string     `yaml:"branch,omitempty"`
	Depth          int        `yaml:"depth,omitempty"`
	Since          string     `yaml:"since,omitempty"`
	NoCurrent      bool       `yaml:"no_current,omitempty"`
	NoHistory      bool       `yaml:"no_history,omitempty"`
	Context        int        `yaml:"context,omitempty"`
}

// Exclude represents each exclusion rule in the configuration.
type Exclude struct {
	Message string   `yaml:"message"`
	Paths   []string `yaml:"paths,omitempty"`
	Pattern string   `yaml:"pattern,omitempty"`
	ID      string   `yaml:"id,omitempty"`
}

// DefaultConfig returns the default configuration for Trufflehog3.
func DefaultConfig() Config {
	return Config{
		Exclude: []*Exclude{
			{
				Message: "3rd-party dependencies",
				Paths: []string{
					"vendor",
					"node_modules",
				},
			},
			{
				Message: "Go lock files",
				Paths: []string{
					"go.mod",
					"go.sum",
				},
			},
			{
				Message: "JS lock files",
				Paths: []string{
					"package.json",
					"package-lock.json",
				},
			},
			{
				Message: "Python lock files",
				Paths: []string{
					"Pipfile.lock",
				},
			},
			{
				Message: "PHP lock files",
				Paths: []string{
					"composer.lock",
				},
			},
			{
				Message: "Git subproject commit hash",
				Pattern: "[0-9a-f]{40}",
			},
		},
	}
}

// LoadConfig loads the YAML configuration from the specified file.
func LoadConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open .trufflehog3.yml file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("failed to decode .trufflehog3.yml file: %w", err)
	}
	return config, nil
}

// SaveConfig saves the YAML configuration to the specified file.
func SaveConfig(path string, config Config) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .trufflehog3.yml file for writing: %w", err)
	}
	defer file.Close()

	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate .trufflehog3.yml file: %w", err)
	}

	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to the beginning of the file: %w", err)
	}

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode updated .trufflehog3.yml file: %w", err)
	}
	return nil
}

// WriteDefaultTrufflehogConfigIfMissing writes the default configuration if the file is missing.
func WriteDefaultTrufflehogConfigIfMissing(logger hclog.Logger, configFilePath string) error {
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		logger.Warn("Config file not found. Creating a new one with default settings.")
		config := DefaultConfig()
		if err := SaveConfig(configFilePath, config); err != nil {
			return fmt.Errorf("failed to write default config: %w", err)
		}
	}
	return nil
}

// ForceOverwriteTrufflehogConfig forcefully overwrites the existing configuration with the provided configuration.
func ForceOverwriteTrufflehogConfig(configFilePath string, config Config) error {
	if err := SaveConfig(configFilePath, config); err != nil {
		return fmt.Errorf("failed to overwrite config with default settings: %w", err)
	}
	return nil
}

// HandleScannerConfig processes the scanner configuration, including writing default configs or overwriting existing ones.
func HandleScannerConfig(logger hclog.Logger, excludePaths []string, targetFolder string, writeDefaultIfMissing bool, forceOverwrite bool) error {
	configFilePath := filepath.Join(targetFolder, ".trufflehog3.yml")

	if writeDefaultIfMissing {
		if err := WriteDefaultTrufflehogConfigIfMissing(logger, configFilePath); err != nil {
			return err
		}
	}

	if forceOverwrite {
		logger.Warn("Overwriting .trufflehog3.yml config file with default settings.")
		if err := ForceOverwriteTrufflehogConfig(configFilePath, DefaultConfig()); err != nil {
			return err
		}
	}

	if len(excludePaths) > 0 {
		logger.Info("Processed excluded paths for Trufflehog3 plugin")
		logger.Debug("Debug info", "paths", excludePaths)
		config, err := LoadConfig(configFilePath)
		if err != nil {
			return err
		}

		newExclusion := Exclude{
			Message: "Custom Scanio config exclusions",
			Paths:   excludePaths,
		}
		config.Exclude = append(config.Exclude, &newExclusion)

		if err := SaveConfig(configFilePath, config); err != nil {
			return err
		}
	}
	return nil
}
