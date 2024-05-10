package bitbucket

import (
	"fmt"
	"strconv"
)

// repositoriesService implements the RepositoriesService interface.
type repositoriesService struct {
	*service
}

// NewRepositoriesService initializes a new projects service.
func NewRepositoriesService(client *Client) RepositoriesService {
	return &repositoriesService{
		service: &service{client},
	}
}

// List retrieves all repositories for a given project with pagination handling.
func (rs *repositoriesService) List(project string) (*[]Repository, error) {
	var result []Repository
	start := 0
	limit := 2000
	path := fmt.Sprintf("/projects/%s/repos", project)
	rs.client.Logger.Info("fetching list of repositories", "project", project)

	for {
		rs.client.Logger.Debug("fetching page of repositories", "start", start, "limit", limit)
		query := map[string]string{
			"start": strconv.Itoa(start),
			"limit": strconv.Itoa(limit),
		}

		response, err := rs.client.get(path, query)
		if err != nil {
			return nil, fmt.Errorf("error fetching repositories: %v", err)
		}

		var resp Response[Repository]
		if err := unmarshalResponse(response, &resp); err != nil {
			return nil, err
		}

		result = append(result, resp.Values...)
		if resp.IsLastPage {
			rs.client.Logger.Debug("last page of repositories reached", "totalFetched", len(result))
			break
		}

		start = resp.NextPageStart
	}

	return &result, nil
}
