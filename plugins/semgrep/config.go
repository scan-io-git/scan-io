package main

import (
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// getDefaultRuleSet returns the default rule set based on the configuration.
func getDefaultRuleSet(cfg *config.Config) string {
	if config.IsCI(cfg) {
		return "p/ci"
	}
	return "p/default"
}
