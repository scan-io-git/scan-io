package vcsintegrator

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/scan-io-git/scan-io/internal/sarif"
	"github.com/scan-io-git/scan-io/pkg/shared"
)

// PrepareSarifComments builds a VCSAddSarifCommentsRequest by parsing a SARIF
// file and converting findings into VCS comment structures.
func PrepareSarifComments(
	logger hclog.Logger,
	pluginName string,
	repo shared.RepositoryParams,
	sarifPath string,
	sourceRoot string,
	limit int,
	summary string,
) (shared.VCSAddSarifCommentsRequest, error) {
	collected, err := sarif.CollectIssuesFromFile(logger, sarifPath, sourceRoot, pluginName, repo.HTTPLink, true)
	if err != nil {
		return shared.VCSAddSarifCommentsRequest{}, fmt.Errorf("collect issues from sarif: %w", err)
	}

	comments := make([]shared.Comment, 0, len(collected))
	for _, issue := range collected {
		comment := shared.Comment{Body: issue.Body}
		if issue.Metadata.Filename != "" {
			comment.Path = issue.Metadata.Filename
		}
		if issue.Metadata.StartLine > 0 {
			comment.Line = issue.Metadata.StartLine
		}
		if issue.Metadata.EndLine > 0 {
			comment.EndLine = issue.Metadata.EndLine
		}
		comments = append(comments, comment)
	}

	total := len(comments)
	if limit > 0 && limit < total {
		comments = comments[:limit]
	}

	summaryText := strings.TrimSpace(summary)
	if total > len(comments) {
		info := fmt.Sprintf("Only the first %d findings out of %d were commented automatically.", len(comments), total)
		if summaryText != "" {
			summaryText = summaryText + "\n\n" + info
		} else {
			summaryText = info
		}
	}

	return shared.VCSAddSarifCommentsRequest{
		VCSRequestBase: shared.VCSRequestBase{
			RepoParam: repo,
			Action:    VCSAddCommentsSarif,
		},
		Comments: comments,
		Summary:  summaryText,
	}, nil
}
