package main

import (
	"os"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// UpdateConfigFromEnv sets configuration values from environment variables, if they are set.
func UpdateConfigFromEnv(cfg *config.Config) error {
	envVars := map[string]*string{
		"SCANIO_GITHUB_USERNAME":         &cfg.GithubPlugin.Username,
		"SCANIO_GITHUB_TOKEN":            &cfg.GithubPlugin.Token,
		"SCANIO_GITHUB_SSH_KEY_PASSWORD": &cfg.GithubPlugin.SSHKeyPassword,
	}

	for env, val := range envVars {
		if v := os.Getenv(env); v != "" {
			*val = v
		}
	}
	return nil
}
