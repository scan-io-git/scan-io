package bitbucket

import (
	"fmt"
	"net/http"
)

// pullRequestsService implements the PullRequestsService interface.
type pullRequestsService struct {
	*service
}

// NewPullRequestsService initializes a new pull requests service.
func NewPullRequestsService(client *Client) PullRequestsService {
	return &pullRequestsService{
		service: &service{client},
	}
}

// AddComment adds a comment to a specific pull request.
func (pr *PullRequest) AddComment(commentText string) (*PullRequest, error) {
	pr.client.Logger.Debug("leaving a comment to a pull request", "project", pr.ToReference.Repository.Project.Key, "repository", pr.ToReference.Repository.Slug, "id", pr.ID)

	path := pr.Links.Self[0].Href + "/comments" // works even without rest/api/1.0/ prefix
	body := map[string]interface{}{
		"text": commentText,
	}

	response, err := pr.client.post(path, nil, body)
	if err != nil {
		return nil, fmt.Errorf("error leaving a comment: %v", err)
	}

	var result PullRequest
	if err := unmarshalResponse(response, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SetStatus sets status to a specified PR.
func (pr *PullRequest) SetStatus(status, login string) (*PullRequest, error) {
	pr.client.Logger.Debug("setting a new status to a pull request", "project", pr.ToReference.Repository.Project.Key, "repository", pr.ToReference.Repository.Slug, "id", pr.ID)

	approval := status == "APPROVED"
	path := pr.Links.Self[0].Href + "/participants/" + login // works even without rest/api/1.0/ prefix
	body := map[string]interface{}{
		"status":   status,
		"approved": approval,
	}

	response, err := pr.client.put(path, nil, body)
	if err != nil {
		return nil, fmt.Errorf("error setting status: %v", err)
	}

	if response.StatusCode() == http.StatusConflict {
		return nil, fmt.Errorf("the PR is already merged")
	}
	if response.StatusCode() == http.StatusBadRequest {
		return nil, fmt.Errorf("conflict error: User %s is an author of the PR", login)
	}

	var result PullRequest
	if err := unmarshalResponse(response, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// AddRole adds a user to a PR with a specified role.
func (pr *PullRequest) AddRole(role, login string) (*UserData, error) {
	pr.client.Logger.Debug("adding a user to a pull request", "project", pr.ToReference.Repository.Project.Key, "repository", pr.ToReference.Repository.Slug, "id", pr.ID)
	path := pr.Links.Self[0].Href + "/participants" // works even without rest/api/1.0/ prefix
	body := map[string]interface{}{
		"user": map[string]string{
			"name": login,
		},
		"role":     role,
		"approved": false,
	}

	response, err := pr.client.post(path, nil, body)
	if err != nil {
		return nil, fmt.Errorf("error adding user to the PR: %v", err)
	}

	if response.StatusCode() == http.StatusConflict {
		return nil, fmt.Errorf("conflict error: User %s is an author of the PR", login)
	}

	if response.StatusCode() < 200 || response.StatusCode() >= 300 {
		return nil, fmt.Errorf("failed to add role reviewer PR, status: %d, body: %s", response.StatusCode(), response.String())
	}

	var result UserData
	if err := unmarshalResponse(response, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get retrieves pull request for a given project, repository, and id.
func (prs *pullRequestsService) Get(project, repository string, id int) (*PullRequest, error) {
	path := fmt.Sprintf("/projects/%s/repos/%s/pull-requests/%v", project, repository, id)
	prs.client.Logger.Debug("fetching pull request information", "project", project, "repository", repository, "id", id)

	response, err := prs.client.get(path, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching a pull request: %v", err)
	}

	var result PullRequest
	if err := unmarshalResponse(response, &result); err != nil {
		return nil, err
	}

	result.client = prs.client
	return &result, nil
}
