package bitbucket

import (
	"fmt"
	"net/http"

	//bitbucketv2 "github.com/ktrysmt/go-bitbucket"
	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	"github.com/go-resty/resty/v2"
	"github.com/mitchellh/mapstructure"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// Limit for Bitbucket v1 API page response
var opts = map[string]interface{}{
	"limit": maxLimitElements,
	"start": startElement,
}

func getProjectsResponse(r *bitbucketv1.APIResponse) ([]bitbucketv1.Project, error) {
	var m []bitbucketv1.Project
	err := mapstructure.Decode(r.Values["values"], &m)
	return m, err
}

// extractCloneLinks parses the clone links from the repository information to find HTTP and SSH URLs.
func extractCloneLinks(clones []bitbucketv1.CloneLink) (string, string) {
	var httpLink, sshLink string
	for _, clone := range clones {
		if clone.Name == "http" {
			httpLink = clone.Href
		} else if clone.Name == "ssh" {
			sshLink = clone.Href
		}
	}
	return httpLink, sshLink
}

// GetPullRequest retrieves a specific pull request by its ID within a project and repository.
func (c *Client) GetPullRequest(projectKey, repoSlug string, prID int) (*PullRequest, error) {
	apiResponse, err := c.APIClient.DefaultApi.GetPullRequest(projectKey, repoSlug, prID)
	if err != nil {
		return nil, fmt.Errorf("error fetching pull request %d: %w", prID, err)
	}

	pullRequestData, err := bitbucketv1.GetPullRequestResponse(apiResponse)
	if err != nil {
		return nil, fmt.Errorf("error Bitbucket response parsing: %w", err)
	}

	result := &PullRequest{
		ID:          pullRequestData.ID,
		Title:       pullRequestData.Title,
		Description: pullRequestData.Description,
		State:       pullRequestData.State,
		Author:      User{DisplayName: pullRequestData.Author.User.DisplayName, Email: pullRequestData.Author.User.EmailAddress},
		SelfLink:    pullRequestData.Links.Self[0].Href,
		Source:      Reference{ID: pullRequestData.FromRef.ID, DisplayId: pullRequestData.FromRef.DisplayID, LatestCommit: pullRequestData.FromRef.LatestCommit},
		Destination: Reference{ID: pullRequestData.ToRef.ID, DisplayId: pullRequestData.ToRef.DisplayID, LatestCommit: pullRequestData.ToRef.LatestCommit},
		CreatedDate: pullRequestData.CreatedDate,
		UpdatedDate: pullRequestData.UpdatedDate,
	}

	return result, nil
}

// Resolving information about all repositories in one project from Bitbucket v1 API
func (c *Client) GetProject(project string) ([]Repository, error) {
	apiResponse, err := c.APIClient.DefaultApi.GetRepositoriesWithOptions(project, opts)
	if err != nil {
		return nil, fmt.Errorf("error fetching Bitbucket projects: %w", err)
	}

	res, err := bitbucketv1.GetRepositoriesResponse(apiResponse)
	if err != nil {
		return nil, fmt.Errorf("error Bitbucket response parsing: %w", err)
	}

	var resultList []Repository
	for _, repo := range res {
		httpLink, sshLink := extractCloneLinks(repo.Links.Clone)
		resultList = append(resultList, Repository{
			Name:     repo.Name,
			Project:  Project{Name: repo.Project.Name},
			HTTPLink: httpLink,
			SSHLink:  sshLink,
		})
	}

	return resultList, nil
}

// GetProjects lists all projects available to the client
func (c *Client) GetProjects() ([]Project, error) {
	apiResponse, err := c.APIClient.DefaultApi.GetProjects(opts)
	if err != nil {
		return nil, fmt.Errorf("error fetching Bitbucket projects: %w", err)
	}

	res, err := getProjectsResponse(apiResponse)
	if err != nil {
		return nil, fmt.Errorf("error Bitbucket response parsing: %w", err)
	}

	var projects []Project
	for _, item := range res {
		projects = append(projects, Project{
			Key:  item.Key,
			Name: item.Name,
			Link: item.Links.Self[0].Href,
		})
	}

	return projects, nil
}

// AddCommentToPR adds a comment to a specific pull request.
func (c *Client) AddCommentToPR(namespace, repository string, pullRequestID int, commentText string) error {
	comment := bitbucketv1.Comment{
		Text: commentText,
	}
	_, err := c.APIClient.DefaultApi.CreatePullRequestComment(namespace, repository, pullRequestID, comment, []string{"application/json"})
	return err
}

// SetPRStatus updates the status of a pull request.
func (c *Client) SetPRStatus(namespace, repository string, pullRequestID int, login, status string) (bitbucketv1.UserWithMetadata, error) {
	//TODO add statuses verification
	approval := status == "APPROVED"
	userBB := bitbucketv1.UserWithMetadata{
		User: bitbucketv1.UserWithLinks{
			Name: login,
			Slug: login,
		},
		Approved: approval,
		Status:   status,
	}

	response, err := c.APIClient.DefaultApi.UpdateStatus(namespace, repository, int64(pullRequestID), login, userBB)
	if err != nil {
		return bitbucketv1.UserWithMetadata{}, fmt.Errorf("error fetching Bitbucket PR information: %w", err)
	}

	participant, err := bitbucketv1.GetUserWithMetadataResponse(response)
	if err != nil {
		return bitbucketv1.UserWithMetadata{}, fmt.Errorf("error Bitbucket response parsing: %w", err)
	}
	return participant, err
}

// AddReviewerToPR adds a reviewer to a pull request using direct HTTP requests.
func AddReviewerToPR(client *resty.Client, vcsURL, namespace, repository string, pullRequestID int, login, role string, authInfo config.BitbucketPlugin) error {
	url := fmt.Sprintf("https://%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/participants/", vcsURL, namespace, repository, pullRequestID)
	response, err := client.R().
		SetBasicAuth(authInfo.BitbucketUsername, authInfo.BitbucketToken).
		SetHeader("Content-Type", "application/json").
		SetBody(fmt.Sprintf(`{"user": {"name": "%s"}, "role": "%s", "approved": false}`, login, role)). // TODO need to hendle values safely
		Post(url)

	if err != nil {
		return err
	}

	if response.StatusCode() == http.StatusConflict {
		return fmt.Errorf("conflict error: User %s is an author of the PR", login)
	}

	if response.StatusCode() < 200 || response.StatusCode() >= 300 {
		return fmt.Errorf("failed to add role reviewer PR, status: %d, body: %s", response.StatusCode(), response.String())
	}
	return nil
}
