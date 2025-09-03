package fetch

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"

	utils "github.com/scan-io-git/scan-io/internal/utils"
)

// Mode constants
const (
	CmdModeSingleURL = "single-url"
	CmdModeInputFile = "input-file"
)

// determineCmdMode determines the cmd mode based on the provided arguments.
func determineCmdMode(args []string) string {
	if len(args) > 0 {
		return CmdModeSingleURL
	}
	return CmdModeInputFile
}

func determineAddFlags(cmd *cobra.Command, cfg *config.Config, options *RunOptionsFetch) (git.TagMode, error) {
	ciMode := config.IsCI(cfg)
	tagMode := resolveTagsMode(cmd)
	df := cmd.Flags().Lookup("depth")
	if df != nil && df.Changed {
		if options.Depth < 0 {
			return tagMode, fmt.Errorf("invalid --depth %d (must be non-negative)", options.Depth)
		}
	} else {
		if ciMode {
			options.Depth = 1 // CI default
		} else {
			options.Depth = 0 // user default
		}
	}

	if options.Branch != "" {
		options.SingleBranch = true
	}

	return tagMode, nil
}

// ResolveTagsMode decides the final TagMode.
func resolveTagsMode(cmd *cobra.Command) git.TagMode {
	tagsSet := cmd.Flags().Changed("tags")
	noTagsSet := cmd.Flags().Changed("no-tags")

	if tagsSet && noTagsSet {
		return git.TagFollowing
	}
	if tagsSet {
		return git.AllTags
	}
	if noTagsSet {
		return git.NoTags
	}

	return git.TagFollowing
}

// prepareFetchTargets prepares the targets for fetching based on the validated arguments.
func prepareFetchTargets(options *RunOptionsFetch, args []string, cmdMode string) ([]shared.RepositoryParams, error) {
	var reposInfo []shared.RepositoryParams

	switch cmdMode {
	case CmdModeSingleURL:
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
			return reposInfo, fmt.Errorf("failed to extract data from provided URL '%q': %w", targetURL, err)
		}

		if err = validationRepoInfo(repoInfo); err != nil {
			return reposInfo, err
		}
		reposInfo = append(reposInfo, repoInfo)

	case CmdModeInputFile:
		reposData, err := utils.ReadReposFile2(options.InputFile)
		if err != nil {
			return nil, fmt.Errorf("error parsing the input file %q: %v", options.InputFile, err)
		}
		for _, repoInfo := range reposData {
			if err = validationRepoInfo(repoInfo); err != nil {
				return reposInfo, err
			}
		}
		reposInfo = reposData
	default:
		return reposInfo, fmt.Errorf("invalid analysing mode: %q", cmdMode)
	}

	return reposInfo, nil
}
