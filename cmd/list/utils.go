package list

import (
	"fmt"

	"github.com/scan-io-git/scan-io/internal/vcsintegrator"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"
)

// Mode constants
const (
	ModeSingleURL = "single-url"
	ModeFlags     = "flags"
)

// determineMode determines the mode based on the provided arguments.
func determineMode(args []string) string {
	if len(args) > 0 {
		return ModeSingleURL
	}
	return ModeFlags
}

// prepareListTargets prepares the targets for listing based on the validated arguments.
func prepareListTarget(options *vcsintegrator.RunOptionsIntegrationVCS, args []string, mode string) (shared.RepositoryParams, error) {
	switch mode {
	case ModeSingleURL:
		targetURL := args[0]
		repoInfo, err := vcsurl.ExtractRepositoryInfoFromURL(targetURL, options.VCSPluginName)
		if err != nil {
			return repoInfo, fmt.Errorf("failed to extract data from provided URL '%s': %w", targetURL, err)
		}
		return repoInfo, nil

	case ModeFlags:
		return shared.RepositoryParams{
			Domain:    options.Domain,
			Namespace: options.Namespace,
		}, nil

	default:
		return shared.RepositoryParams{}, fmt.Errorf("invalid listing mode: %s", mode)
	}
}
