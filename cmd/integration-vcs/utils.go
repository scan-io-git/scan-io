package integrationvcs

import (
	"fmt"
	"os"

	"github.com/scan-io-git/scan-io/internal/vcsintegrator"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
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

// prepareIntegrationVCSTarget prepares the targets for integration VCS based on the validated arguments.
func prepareIntegrationVCSTarget(options *vcsintegrator.RunOptionsIntegrationVCS, args []string, mode string) (shared.RepositoryParams, error) {
	commentContent, err := getCommentContent(options)
	if err != nil {
		return shared.RepositoryParams{}, err
	}
	options.Comment = commentContent

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
			Domain:        options.Domain,
			Namespace:     options.Namespace,
			Repository:    options.Repository,
			PullRequestID: options.PullRequestID,
		}, nil

	default:
		return shared.RepositoryParams{}, fmt.Errorf("invalid integration VCS mode: %s", mode)
	}
}

// getCommentContent reads the comment content from a file or directly from options.
func getCommentContent(options *vcsintegrator.RunOptionsIntegrationVCS) (string, error) {
	if options.CommentFile == "" {
		return options.Comment, nil
	}

	expandedPath, err := files.ExpandPath(options.CommentFile)
	if err != nil {
		return "", fmt.Errorf("failed to expand path '%s': %w", options.CommentFile, err)
	}

	if err := files.ValidatePath(expandedPath); err != nil {
		return "", fmt.Errorf("failed to validate path '%s': %w", expandedPath, err)
	}

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read comment file: %v", err)
	}
	return string(data), nil
}
