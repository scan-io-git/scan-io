package bitbucket

import (
	"fmt"
	"strconv"
)

// repositoriesService implements the RepositoriesService interface.
type repositoriesService struct {
	*service
	limit int
}

// NewRepositoriesService initializes a new repositories service with a given pagination limit.
func NewRepositoriesService(client *Client, limit int) RepositoriesService {
	if limit <= 0 {
		limit = 2000 // Default limit if not provided
	}
	return &repositoriesService{
		service: &service{client},
		limit:   limit,
	}
}

// List retrieves all repositories for a given project with pagination handling.
func (rs *repositoriesService) List(project string) (*[]Repository, error) {
	var result []Repository
	start := 0
	path := fmt.Sprintf("/projects/%s/repos", project)
	rs.client.Logger.Info("fetching list of repositories",
		"project", project,
	)

	for {
		rs.client.Logger.Debug("fetching page of repositories",
			"start", start, "limit",
			rs.limit,
		)
		query := map[string]string{
			"start": strconv.Itoa(start),
			"limit": strconv.Itoa(rs.limit),
		}

		response, err := rs.client.get(path, query)
		if err != nil {
			return nil, fmt.Errorf("error fetching repositories: %w", err)
		}

		var resp Response[Repository]
		if err := unmarshalResponse(response, &resp); err != nil {
			return nil, err
		}

		result = append(result, resp.Values...)
		if resp.IsLastPage || resp.NextPageStart == nil {
			rs.client.Logger.Debug("last page of repositories reached")
			break
		}

		start = *resp.NextPageStart
	}

	rs.client.Logger.Debug("successfully fetched all repositories",
		"totalRepositories", len(result),
	)
	return &result, nil
}

// List retrieves all repositories for a given user with pagination handling.
func (rs *repositoriesService) ListUserRepos(username string) (*[]Repository, error) {
	var result []Repository
	start := 0
	path := fmt.Sprintf("/users/%s/repos", username)
	rs.client.Logger.Info("fetching list of repositories",
		"username", username,
	)

	for {
		rs.client.Logger.Debug("fetching page of repositories",
			"start", start, "limit",
			rs.limit,
		)
		query := map[string]string{
			"start": strconv.Itoa(start),
			"limit": strconv.Itoa(rs.limit),
		}

		response, err := rs.client.get(path, query)
		if err != nil {
			return nil, fmt.Errorf("error fetching repositories: %w", err)
		}

		var resp Response[Repository]
		if err := unmarshalResponse(response, &resp); err != nil {
			return nil, err
		}

		result = append(result, resp.Values...)
		if resp.IsLastPage || resp.NextPageStart == nil {
			rs.client.Logger.Debug("last page of repositories reached")
			break
		}

		start = *resp.NextPageStart
	}

	rs.client.Logger.Debug("successfully fetched all repositories",
		"totalRepositories", len(result),
	)
	return &result, nil
}
