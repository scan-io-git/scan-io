package main

import (
	"github.com/scan-io-git/scan-io/internal/bitbucket"
	"github.com/scan-io-git/scan-io/pkg/shared"
)

// toRepositoryParams converts a slice of internal Repository type to a slice of external RepositoryParams type.
func toRepositoryParams(repos *[]bitbucket.Repository) []shared.RepositoryParams {
	var repoParams []shared.RepositoryParams
	for _, repo := range *repos {
		httpLink, sshLink := bitbucket.ExtractCloneLinks(repo.Links.Clone)
		repoParams = append(repoParams, shared.RepositoryParams{
			Namespace: repo.Project.Name,
			RepoName:  repo.Name,
			HttpLink:  httpLink,
			SshLink:   sshLink,
		})
	}
	return repoParams
}

// convertToPRParams converts the internal PullRequest type to the external PRParams type.
func convertToPRParams(pr *bitbucket.PullRequest) shared.PRParams {
	return shared.PRParams{
		Id:          pr.ID,
		Title:       pr.Title,
		Description: pr.Description,
		State:       pr.State,
		Author:      shared.User{DisplayName: pr.Author.User.DisplayName, Email: pr.Author.User.EmailAddress},
		SelfLink:    pr.Links.Self[0].Href,
		Source: shared.Reference{
			ID:           pr.FromReference.ID,
			DisplayId:    pr.FromReference.DisplayID,
			LatestCommit: pr.FromReference.LatestCommit,
		},
		Destination: shared.Reference{
			ID:           pr.ToReference.ID,
			DisplayId:    pr.ToReference.DisplayID,
			LatestCommit: pr.ToReference.LatestCommit,
		},
		CreatedDate: pr.CreatedDate,
		UpdatedDate: pr.UpdatedDate,
	}
}
