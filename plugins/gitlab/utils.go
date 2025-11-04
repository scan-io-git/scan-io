package main

import (
	"sort"
	"strings"
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
			Namespace:  repo.Namespace.FullPath,
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

func verdictPriority(verdict string) int {
	switch verdict {
	case "APPROVED":
		return 3
	case "CHANGES_REQUESTED", "REJECTED":
		return 2
	default:
		return 1
	}
}

func convertGitlabReviewers(mr *gitlab.MergeRequest, approvals *gitlab.MergeRequestApprovals) []shared.PRReview {
	if mr == nil {
		return nil
	}

	type reviewerState struct {
		verdict string
		review  shared.PRReview
	}

	reviewers := make(map[int]reviewerState)

	addOrUpdate := func(user *gitlab.BasicUser, verdict string, approved bool) {
		if user == nil {
			return
		}

		normalizedVerdict := strings.ToUpper(strings.TrimSpace(verdict))
		if normalizedVerdict == "" {
			if approved {
				normalizedVerdict = "APPROVED"
			} else {
				normalizedVerdict = "PENDING"
			}
		}

		state := reviewerState{
			verdict: normalizedVerdict,
			review: shared.PRReview{
				Reviewer: safeUser(user),
				Verdict:  normalizedVerdict,
			},
		}

		if existing, ok := reviewers[user.ID]; ok {
			if verdictPriority(normalizedVerdict) > verdictPriority(existing.verdict) {
				reviewers[user.ID] = state
			}
			return
		}

		reviewers[user.ID] = state
	}

	for _, reviewer := range mr.Reviewers {
		addOrUpdate(reviewer, "PENDING", false)
	}

	if approvals != nil {
		for _, approver := range approvals.Approvers {
			if approver == nil {
				continue
			}
			addOrUpdate(approver.User, "PENDING", false)
		}

		for _, approved := range approvals.ApprovedBy {
			if approved == nil {
				continue
			}
			addOrUpdate(approved.User, "APPROVED", true)
		}
	}

	if len(reviewers) == 0 {
		return nil
	}

	result := make([]shared.PRReview, 0, len(reviewers))
	for _, reviewer := range reviewers {
		result = append(result, reviewer.review)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Reviewer.UserName < result[j].Reviewer.UserName
	})

	return result
}
