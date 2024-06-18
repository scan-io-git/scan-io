package main

import (
	"os"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// UpdateConfigFromEnv sets configuration values from environment variables, if they are set.
func UpdateConfigFromEnv(cfg *config.Config) error {
	envVars := map[string]*string{
		"SCANIO_GITLAB_USERNAME":         &cfg.GitlabPlugin.Username,
		"SCANIO_GITLAB_TOKEN":            &cfg.GitlabPlugin.Token,
		"SCANIO_GITLAB_SSH_KEY_PASSWORD": &cfg.GitlabPlugin.SSHKeyPassword,
	}

	for env, val := range envVars {
		if v := os.Getenv(env); v != "" {
			*val = v
		}
	}
	return nil
}
