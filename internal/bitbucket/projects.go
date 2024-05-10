package bitbucket

import (
	"fmt"
	"strconv"
)

// projectsService implements the ProjectsService interface.
type projectsService struct {
	*service
}

// NewProjectsService initializes a new projects service.
func NewProjectsService(client *Client) ProjectsService {
	return &projectsService{
		service: &service{client},
	}
}

// List retrieves all projects with pagination handling.
func (ps *projectsService) List() (*[]Project, error) {
	var result []Project
	start := 0
	limit := 2000
	path := "/projects/"
	ps.client.Logger.Debug("fetching list of projects")

	for {
		ps.client.Logger.Debug("fetching page of repositories", "start", start, "limit", limit)
		query := map[string]string{
			"start": strconv.Itoa(start),
			"limit": strconv.Itoa(limit),
		}

		response, err := ps.client.get(path, query)
		if err != nil {
			return nil, fmt.Errorf("error fetching projects: %v", err)
		}

		var resp Response[Project]
		if err := unmarshalResponse(response, &resp); err != nil {
			return nil, err
		}

		result = append(result, resp.Values...)
		if resp.IsLastPage {
			ps.client.Logger.Debug("last page of projects reached", "totalFetched", len(result))
			break
		}

		start = resp.NextPageStart
	}

	return &result, nil
}
