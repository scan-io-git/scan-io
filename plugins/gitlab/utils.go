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

// repoToParams converts a single *gitlab.Repository into RepositoryParams.
// ok=false if repo is nil.
func repoToParams(repo *gitlab.Project) (shared.RepositoryParams, bool) {
	if repo == nil {
		return shared.RepositoryParams{}, false
	}

	return shared.RepositoryParams{
		Domain:     "",
		Namespace:  repo.Namespace.FullPath,
		Repository: repo.Path,
		HTTPLink:   repo.HTTPURLToRepo,
		SSHLink:    repo.SSHURLToRepo,
	}, true
}

// toNamespaceParams converts a slice of internal Repository type to a slice of external RepositoryParams type.
func toNamespaceParams(repos []*gitlab.Project) ([]shared.NamespaceParams, int) {
	var gRepoCount int
	npMap := make(map[string][]shared.RepositoryParams)

	for _, repo := range repos {
		if repo == nil {
			continue
		}

		groupName := repo.Namespace.FullPath
		if rp, ok := repoToParams(repo); ok {
			npMap[groupName] = append(npMap[groupName], rp)
		}
	}

	result := make([]shared.NamespaceParams, 0, len(npMap))
	for ns, repos := range npMap {
		repoCount := len(repos)
		gRepoCount += repoCount
		result = append(result, shared.NamespaceParams{
			Namespace:       ns,
			RepositoryCount: len(repos),
			Repositories:    repos,
		})
	}

	return result, gRepoCount
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
