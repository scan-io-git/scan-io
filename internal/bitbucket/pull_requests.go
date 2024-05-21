package bitbucket

import (
	"fmt"
	"net/http"
	"strconv"
)

// pullRequestsService implements the PullRequestsService interface.
type pullRequestsService struct {
	*service
	limit int
}

// NewPullRequestsService initializes a new pull requests service with a given pagination limit.
func NewPullRequestsService(client *Client, limit int) PullRequestsService {
	if limit <= 0 {
		limit = 2000 // Default limit if not provided
	}
	return &pullRequestsService{
		service: &service{client},
		limit:   limit,
	}
}

// Get retrieves a pull request for a given project, repository, and ID.
func (prs *pullRequestsService) Get(project, repository string, id int) (*PullRequest, error) {
	path := fmt.Sprintf("/projects/%s/repos/%s/pull-requests/%d", project, repository, id)
	prs.client.Logger.Debug("fetching pull request information", "project", project, "repository", repository, "id", id)

	response, err := prs.client.get(path, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching pull request: %w", err)
	}

	var result PullRequest
	if err := unmarshalResponse(response, &result); err != nil {
		return nil, err
	}

	result.client = prs.client
	return &result, nil
}

// GetChanges retrieves the changes for a pull request.
func (pr *PullRequest) GetChanges() (*[]Change, error) {
	pr.client.Logger.Debug("getting changes for a pull request", "project", pr.ToReference.Repository.Project.Key, "repository", pr.ToReference.Repository.Slug, "id", pr.ID)
	return pr.paginateChanges(pr.Links.Self[0].Href+"/changes", pr.client)
}

// AddComment adds a comment to a specific pull request.
func (pr *PullRequest) AddComment(commentText string) (*PullRequest, error) {
	pr.client.Logger.Debug("leaving a comment on a pull request", "project", pr.ToReference.Repository.Project.Key, "repository", pr.ToReference.Repository.Slug, "id", pr.ID)

	path := pr.Links.Self[0].Href + "/comments" // works even without rest/api/1.0/ prefix
	body := map[string]interface{}{
		"text": commentText,
	}

	response, err := pr.client.post(path, nil, body)
	if err != nil {
		return nil, fmt.Errorf("error leaving a comment: %w", err)
	}

	var result PullRequest
	if err := unmarshalResponse(response, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SetStatus sets the status of a specified pull request.
func (pr *PullRequest) SetStatus(status, login string) (*PullRequest, error) {
	pr.client.Logger.Debug("setting a new status for a pull request", "project", pr.ToReference.Repository.Project.Key, "repository", pr.ToReference.Repository.Slug, "id", pr.ID)

	approval := status == "APPROVED"
	path := pr.Links.Self[0].Href + "/participants/" + login // works even without rest/api/1.0/ prefix
	body := map[string]interface{}{
		"status":   status,
		"approved": approval,
	}

	response, err := pr.client.put(path, nil, body)
	if err != nil {
		return nil, fmt.Errorf("error setting status: %w", err)
	}

	if response.StatusCode() == http.StatusConflict {
		return nil, fmt.Errorf("the pull request is already merged")
	}
	if response.StatusCode() == http.StatusBadRequest {
		return nil, fmt.Errorf("conflict error: User %s is an author of the pull request", login)
	}

	var result PullRequest
	if err := unmarshalResponse(response, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// AddRole adds a user to a pull request with a specified role.
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
		return nil, fmt.Errorf("error adding user to the pull request: %w", err)
	}

	if response.StatusCode() == http.StatusConflict {
		return nil, fmt.Errorf("conflict error: User %s is an author of the pull request", login)
	}

	if response.StatusCode() < 200 || response.StatusCode() >= 300 {
		return nil, fmt.Errorf("failed to add role to pull request, status: %d, body: %s", response.StatusCode(), response.String())
	}

	var result UserData
	if err := unmarshalResponse(response, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// paginateChanges handles pagination for pull request changes.
func (pr *PullRequest) paginateChanges(path string, client *Client) (*[]Change, error) {
	var result []Change
	start := 0
	limit := 2000

	for {
		client.Logger.Debug("fetching page of changes", "start", start, "limit", limit)
		query := map[string]string{
			"start":        strconv.Itoa(start),
			"limit":        strconv.Itoa(limit),
			"withComments": "false",
		}

		response, err := client.get(path, query)
		if err != nil {
			return nil, fmt.Errorf("error getting changes: %w", err)
		}

		var resp ChangesResponse[Change]
		if err := unmarshalResponse(response, &resp); err != nil {
			return nil, err
		}

		result = append(result, resp.Values...)
		if resp.IsLastPage {
			client.Logger.Debug("last page of changes reached")
			break
		}

		start = resp.NextPageStart
	}

	client.Logger.Debug("successfully fetched all changes", "totalChanges", len(result))
	return &result, nil
}
