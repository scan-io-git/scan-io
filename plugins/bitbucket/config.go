package main

import (
	"os"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// UpdateConfigFromEnv sets configuration values from environment variables, if they are set.
func UpdateConfigFromEnv(cfg *config.Config) error {
	envVars := map[string]*string{
		"SCANIO_BITBUCKET_USERNAME":         &cfg.BitbucketPlugin.Username,
		"SCANIO_BITBUCKET_TOKEN":            &cfg.BitbucketPlugin.Token,
		"SCANIO_BITBUCKET_SSH_KEY_PASSWORD": &cfg.BitbucketPlugin.SSHKeyPassword,
	}

	for env, val := range envVars {
		if v := os.Getenv(env); v != "" {
			*val = v
		}
	}
	return nil
}
