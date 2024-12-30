package main

import (
	// "strings"
	"time"

	"gitlab.com/gitlab-org/api/client-go"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

// safeTime safely dereferences a time pointer, returning 0 if the pointer is nil.
func safeTime(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.Unix()
}

// safeUser converts a GitHub user to a shared.User, handling nil safely.
func safeUser(user *gitlab.BasicUser) shared.User {
	if user == nil || len(user.Username) == 0 {
		return shared.User{UserName: "unknown"}
	}
	return shared.User{UserName: user.Username}
}

// toRepositoryParams converts a slice of internal Project type to a slice of external RepositoryParams type.
func toRepositoryParams(repos []*gitlab.Project) []shared.RepositoryParams {
	var repoParams []shared.RepositoryParams
	for _, repo := range repos {
		if repo == nil {
			continue
		}

		repoParams = append(repoParams, shared.RepositoryParams{
			Domain:     "",
			Namespace:  repo.Namespace.Path,
			Repository: repo.Path,
			HTTPLink:   repo.HTTPURLToRepo,
			SSHLink:    repo.SSHURLToRepo,
		})
	}
	return repoParams
}

// convertToPRParams converts a GitHub PullRequest object to shared.PRParams.
func convertToPRParams(mr *gitlab.MergeRequest) shared.PRParams {

	if mr == nil {
		return shared.PRParams{}
	}

	selfLink := "no-link-available"
	if len(mr.WebURL) > 0 {
		selfLink = mr.WebURL
	}

	return shared.PRParams{
		ID:          mr.IID,
		Title:       mr.Title,
		Description: mr.Description,
		State:       mr.State,
		Author:      safeUser(mr.Author),
		SelfLink:    selfLink,
		Source: shared.Reference{
			ID:           mr.SourceBranch, // Gitlab doesn't provide a reference as BB and Gitlab
			DisplayID:    mr.SourceBranch,
			LatestCommit: mr.DiffRefs.HeadSha,
		},
		Destination: shared.Reference{
			ID:           mr.TargetBranch, // Gitlab doesn't provide a reference as BB and Gitlab
			DisplayID:    mr.TargetBranch,
			LatestCommit: mr.DiffRefs.BaseSha,
		},
		CreatedDate: safeTime(mr.CreatedAt),
		UpdatedDate: safeTime(mr.UpdatedAt),
	}
}
