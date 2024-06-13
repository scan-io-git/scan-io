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
func prepareFetchTargets(allArgumentsFetch *RunOptionsFetch, args []string, mode string) ([]shared.RepositoryParams, error) {
	var reposInfo []shared.RepositoryParams

	switch mode {
	case ModeSingleURL:
		targetURL := args[0]
		repoInfo, err := vcsurl.ExtractRepositoryInfoFromURL(targetURL, allArgumentsFetch.VCSPluginName)
		if err != nil {
			return reposInfo, fmt.Errorf("failed to extract data from provided URL '%s': %w", targetURL, err)
		}
		reposInfo = append(reposInfo, repoInfo)

	case ModeInputFile:
		reposData, err := utils.ReadReposFile2(allArgumentsFetch.InputFile)
		if err != nil {
			return nil, fmt.Errorf("error parsing the input file %s: %v", allArgumentsFetch.InputFile, err)
		}
		reposInfo = reposData
	}

	return reposInfo, nil
}
