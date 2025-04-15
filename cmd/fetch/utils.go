package fetch

import (
	"fmt"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"

	utils "github.com/scan-io-git/scan-io/internal/utils"
)

// Mode constants
const (
	ModeSingleURL = "single-url"
	ModeInputFile = "input-file"
)

// determineMode determines the mode based on the provided arguments.
func determineMode(args []string) string {
	if len(args) > 0 {
		return ModeSingleURL
	}
	return ModeInputFile
}

// prepareFetchTargets prepares the targets for fetching based on the validated arguments.
func prepareFetchTargets(options *RunOptionsFetch, args []string, mode string) ([]shared.RepositoryParams, error) {
	var reposInfo []shared.RepositoryParams

	switch mode {
	case ModeSingleURL:
		targetURL := args[0]
		vcsType := vcsurl.StringToVCSType(options.VCSPluginName)
		url, err := vcsurl.ParseForVCSType(targetURL, vcsType)
		repoInfo := shared.RepositoryParams{
			Domain:        url.ParsedURL.Hostname(),
			Namespace:     url.Namespace,
			Repository:    url.Repository,
			Branch:        url.Branch,
			PullRequestID: url.PullRequestId,
			HTTPLink:      url.HTTPRepoLink,
			SSHLink:       url.SSHRepoLink,
		}

		if err != nil {
			return reposInfo, fmt.Errorf("failed to extract data from provided URL '%s': %w", targetURL, err)
		}

		if err = validationRepoInfo(repoInfo); err != nil {
			return reposInfo, err
		}
		reposInfo = append(reposInfo, repoInfo)

	case ModeInputFile:
		reposData, err := utils.ReadReposFile2(options.InputFile)
		if err != nil {
			return nil, fmt.Errorf("error parsing the input file %s: %v", options.InputFile, err)
		}
		for _, repoInfo := range reposData {
			if err = validationRepoInfo(repoInfo); err != nil {
				return reposInfo, err
			}
		}
		reposInfo = reposData
	default:
		return reposInfo, fmt.Errorf("invalid analysing mode: %s", mode)
	}

	return reposInfo, nil
}
