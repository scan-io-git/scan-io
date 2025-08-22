package main

import (
	"strings"
	"time"

	"github.com/google/go-github/v47/github"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

// safeString safely dereferences a string pointer, returning an empty string if the pointer is nil.
func safeString(s *string) string {
    if s == nil {
        return ""
    }
    return *s
}

// safeInt safely dereferences an int pointer, returning 0 if the pointer is nil.
func safeInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// safeTime safely dereferences a time pointer, returning 0 if the pointer is nil.
func safeTime(t *time.Time) int64 {
    if t == nil {
        return 0
    }
    return t.Unix()
}

// convertToIssueParams converts a GitHub Issue object to shared.IssueParams.
func convertToIssueParams(iss *github.Issue) shared.IssueParams {
    if iss == nil {
        return shared.IssueParams{}
    }

    return shared.IssueParams{
        Number:      safeInt(iss.Number),
        Title:       safeString(iss.Title),
        State:       safeString(iss.State),
        Author:      safeUser(iss.User),
        URL:         safeString(iss.HTMLURL),
        CreatedDate: safeTime(iss.CreatedAt),
        UpdatedDate: safeTime(iss.UpdatedAt),
    }
}

// safeUser converts a GitHub user to a shared.User, handling nil safely.
func safeUser(user *github.User) shared.User {
	if user == nil || user.Login == nil {
		return shared.User{UserName: "unknown"}
	}
	return shared.User{UserName: *user.Login}
}

// safeReference converts a GitHub reference to a shared.Reference, handling nil safely.
func safeReference(ref *github.PullRequestBranch, latestCommit *string) shared.Reference {
	if ref == nil {
		return shared.Reference{}
	}
	return shared.Reference{
		ID:           safeString(ref.Label),
		DisplayID:    safeString(ref.Ref),
		LatestCommit: safeString(latestCommit),
	}
}

// toRepositoryParams converts a slice of internal Repository type to a slice of external RepositoryParams type.
func toRepositoryParams(repos []*github.Repository) []shared.RepositoryParams {
	var repoParams []shared.RepositoryParams
	for _, repo := range repos {
		if repo == nil {
			continue
		}

		fullName := safeString(repo.FullName)
		parts := strings.Split(fullName, "/")

		repoParams = append(repoParams, shared.RepositoryParams{
			Domain:     safeString(repo.Homepage),
			Namespace:  strings.Join(parts[:len(parts)-1], "/"),
			Repository: safeString(repo.Name),
			HTTPLink:   safeString(repo.CloneURL),
			SSHLink:    safeString(repo.SSHURL),
		})
	}
	return repoParams
}

// convertToPRParams converts a GitHub PullRequest object to shared.PRParams.
func convertToPRParams(pr *github.PullRequest) shared.PRParams {

	if pr == nil {
		return shared.PRParams{}
	}

	selfLink := "no-link-available"
	if pr.Links != nil && pr.Links.Self != nil && len(*pr.Links.Self.HRef) > 0 {
		selfLink = *pr.Links.Self.HRef
	}

	return shared.PRParams{
		ID:          *pr.Number,
		Title:       safeString(pr.Title),
		Description: safeString(pr.Body),
		State:       safeString(pr.State),
		Author:      safeUser(pr.User),
		SelfLink:    selfLink,
		Source:      safeReference(pr.Head, pr.Head.SHA),
		Destination: safeReference(pr.Base, pr.Base.SHA),
		CreatedDate: safeTime(pr.CreatedAt),
		UpdatedDate: safeTime(pr.UpdatedAt),
	}
}
