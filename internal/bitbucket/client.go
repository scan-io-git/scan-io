package bitbucket

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-hclog"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/httpclient"
)

// service wraps a client to access different services.
type service struct {
	client *Client
}

// Client configures and manages access to the API, holding service implementations and an HTTP client.
type Client struct {
	HTTPClient   *httpclient.Client
	BaseURL      string
	Logger       hclog.Logger
	Repositories RepositoriesService
	Projects     ProjectsService
	PullRequests PullRequestsService
}

// RepositoriesService defines the interface for repository-related operations.
type RepositoriesService interface {
	List(project string) (*[]Repository, error)
	ListUserRepos(username string) (*[]Repository, error)
}

// ProjectsService defines the interface for project-related operations.
type ProjectsService interface {
	List() (*[]Project, error)
}

// PullRequestsService defines the interface for pull request-related operations.
type PullRequestsService interface {
	Get(project, repository string, id int) (*PullRequest, error)
}

// AuthInfo holds authentication details for Bitbucket access.
type AuthInfo struct {
	Username string // Username for Bitbucket access
	Token    string // Token for basic authentication
}

// resolveURL constructs the full URL by checking if the path is absolute or relative.
func (c *Client) resolveURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.BaseURL + path
}

// requestBuilder returns a common request builder with the necessary headers and authentication.
func (c *Client) headersBuilder() *resty.Request {
	return c.HTTPClient.RestyClient.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
}

// get sends a GET request using the client's base URL, path, and query parameters provided.
func (c *Client) get(path string, queryParams map[string]string) (*resty.Response, error) {
	fullURL := c.resolveURL(path)
	return c.headersBuilder().
		SetQueryParams(queryParams).
		Get(fullURL)
}

// post sends a POST request using the client's base URL, path, query parameters, and body provided.
func (c *Client) post(path string, queryParams map[string]string, body interface{}) (*resty.Response, error) {
	fullURL := c.resolveURL(path)
	return c.headersBuilder().
		SetQueryParams(queryParams).
		SetBody(body).
		Post(fullURL)
}

// put sends a PUT request using the client's base URL, path, query parameters, and body provided.
func (c *Client) put(path string, queryParams map[string]string, body interface{}) (*resty.Response, error) {
	fullURL := c.resolveURL(path)
	return c.headersBuilder().
		SetQueryParams(queryParams).
		SetBody(body).
		Put(fullURL)
}

// upload sends a multipart form-data request using the client's base URL, path, query parameters, and file provided.
func (c *Client) upload(path string, queryParams map[string]string, filePath, fileName string) (*resty.Response, error) {
	if fileName == "" {
		fileName = "files"
	}
	fullURL := c.resolveURL(path)
	return c.HTTPClient.RestyClient.R().
		SetQueryParams(queryParams).
		SetFile(fileName, filePath).
		Post(fullURL)
}

// unmarshalResponse is a generic function to parse JSON body from response into the provided type.
// It also checks the HTTP response code and API error messages.
func unmarshalResponse[T any](resp *resty.Response, out *T) error {
	if resp.StatusCode() >= 400 {
		var errorList ErrorList

		if err := json.Unmarshal(resp.Body(), &errorList); err == nil && len(errorList.Errors) > 0 {
			return fmt.Errorf("API error(s) occurred with status code %d: %+v", resp.StatusCode(), errorList.Errors)
		}
		return fmt.Errorf("API request failed with status code %d and response: %s", resp.StatusCode(), resp.String())
	}

	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return nil
}

// New initializes a new API client with configured services.
func New(globalConfig *config.Config, logger hclog.Logger, domain string, auth AuthInfo) (*Client, error) {
	httpClient, err := httpclient.New(logger, globalConfig)
	if err != nil {
		logger.Error("failed to initialize HTTP client", "error", err)
		return nil, err
	}

	httpClient.RestyClient.
		SetBasicAuth(auth.Username, auth.Token)

	client := &Client{
		HTTPClient: httpClient,
		BaseURL:    fmt.Sprintf("https://%s/rest/api/1.0", domain),
		Logger:     logger,
	}

	client.Repositories = NewRepositoriesService(client, 0)
	client.Projects = NewProjectsService(client, 0)
	client.PullRequests = NewPullRequestsService(client, 0)

	return client, nil
}
