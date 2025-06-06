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
			Domain:     "",
			Namespace:  repo.Project.Key,
			Repository: repo.Name,
			HTTPLink:   httpLink,
			SSHLink:    sshLink,
		})
	}
	return repoParams
}

func convertToPRParams(pr *bitbucket.PullRequest) shared.PRParams {
	var selfLink string
	if len(pr.Links.Self) > 0 {
		selfLink = pr.Links.Self[0].Href
	} else {
		selfLink = "no-link-available"
	}
	return shared.PRParams{
		ID:          pr.ID,
		Title:       pr.Title,
		Description: pr.Description,
		State:       pr.State,
		Author:      shared.User{UserName: pr.Author.User.DisplayName, Email: pr.Author.User.EmailAddress},
		SelfLink:    selfLink,
		Source: shared.Reference{
			ID:           pr.FromReference.ID,
			DisplayID:    pr.FromReference.DisplayID,
			LatestCommit: pr.FromReference.LatestCommit,
		},
		Destination: shared.Reference{
			ID:           pr.ToReference.ID,
			DisplayID:    pr.ToReference.DisplayID,
			LatestCommit: pr.ToReference.LatestCommit,
		},
		CreatedDate: pr.CreatedDate,
		UpdatedDate: pr.UpdatedDate,
	}
}
