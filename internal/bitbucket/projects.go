package bitbucket

import (
	"fmt"
	"strconv"
)

// projectsService implements the ProjectsService interface.
type projectsService struct {
	*service
	limit int
}

// NewProjectsService initializes a new projects service with a given pagination limit.
func NewProjectsService(client *Client, limit int) ProjectsService {
	if limit <= 0 {
		limit = 2000 // Default limit if not provided
	}
	return &projectsService{
		service: &service{client},
		limit:   limit,
	}
}

// List retrieves all projects with pagination handling.
func (ps *projectsService) List() (*[]Project, error) {
	var result []Project
	start := 0
	path := "/projects/"
	ps.client.Logger.Debug("fetching list of projects")

	for {
		ps.client.Logger.Debug("fetching page of projects",
			"start", start,
			"limit", ps.limit,
		)
		query := map[string]string{
			"start": strconv.Itoa(start),
			"limit": strconv.Itoa(ps.limit),
		}

		response, err := ps.client.get(path, query)
		if err != nil {
			return nil, fmt.Errorf("error fetching projects: %w", err)
		}

		var resp Response[Project]
		if err := unmarshalResponse(response, &resp); err != nil {
			return nil, err
		}

		result = append(result, resp.Values...)
		if resp.IsLastPage {
			ps.client.Logger.Debug("last page of projects reached")
			break
		}

		start = resp.NextPageStart
	}

	ps.client.Logger.Debug("successfully fetched all projects",
		"totalProjects", len(result),
	)
	return &result, nil
}
