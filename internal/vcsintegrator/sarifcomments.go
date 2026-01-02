package vcsintegrator

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/scan-io-git/scan-io/internal/sarif"
	"github.com/scan-io-git/scan-io/pkg/shared"
)

// SarifCommentStats describes how many findings were processed versus included.
type SarifCommentStats struct {
	Total    int
	Included int
}

// PrepareSarifComments builds a VCSAddInLineCommentsListRequest by parsing a SARIF
// file and converting findings into VCS comment structures. It also returns
// statistics about how many findings were processed versus included in the
// resulting request.
func prepareSarifComments(
	logger hclog.Logger,
	pluginName string,
	repo shared.RepositoryParams,
	sarifPath string,
	sourceRoot string,
	limit int,
	summary string,
	sarifLevels []string,
) (shared.VCSAddInLineCommentsListRequest, SarifCommentStats, error) {
	collected, err := sarif.CollectIssuesFromFile(logger, sarifPath, sourceRoot, pluginName, repo.HTTPLink, sarifLevels, true)
	if err != nil {
		return shared.VCSAddInLineCommentsListRequest{}, SarifCommentStats{}, fmt.Errorf("collect issues from sarif: %w", err)
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

	stats := SarifCommentStats{Total: len(comments)}
	if limit > 0 && limit < stats.Total {
		comments = comments[:limit]
	}
	stats.Included = len(comments)

	summaryText := strings.TrimSpace(summary)
	if stats.Total > stats.Included {
		info := fmt.Sprintf("Only the first %d findings out of %d were commented automatically.", stats.Included, stats.Total)
		if summaryText != "" {
			summaryText = summaryText + "\n\n" + info
		} else {
			summaryText = info
		}
	}

	return shared.VCSAddInLineCommentsListRequest{
		VCSRequestBase: shared.VCSRequestBase{
			RepoParam: repo,
			Action:    VCSAddInLineCommentsSarif,
		},
		Comments: comments,
		Summary:  summaryText,
	}, stats, nil
}
