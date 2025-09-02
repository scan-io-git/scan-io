package shared

import (
	"fmt"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// RepositoryParams holds the details of a repository.
type RepositoryParams struct {
	Domain        string `json:"domain,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	Repository    string `json:"repository,omitempty"`
	Branch        string `json:"branch,omitempty"`
	PullRequestID string `json:"pull_request_id,omitempty"`
	HTTPLink      string `json:"http_link,omitempty"`
	SSHLink       string `json:"ssh_link,omitempty"`
}

// PRParams holds the details of a pull request.
type PRParams struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	Author      User      `json:"author"`
	SelfLink    string    `json:"self_link"`
	Source      Reference `json:"source"`
	Destination Reference `json:"destination"`
	CreatedDate int64     `json:"created_date"`
	UpdatedDate int64     `json:"updated_date"`
}

// IssueParams holds the details of an issue.
type IssueParams struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Body        string `json:"body,omitempty"`
	State       string `json:"state"`
	Author      User   `json:"author"`
	URL         string `json:"url"`
	CreatedDate int64  `json:"created_date"`
	UpdatedDate int64  `json:"updated_date"`
}

// User holds the details of a user.
type User struct {
	UserName string `json:"user_name"`
	Email    string `json:"email"`
}

// Reference holds the details of a reference in a repository.
type Reference struct {
	ID           string `json:"id"`
	DisplayID    string `json:"display_id"`
	LatestCommit string `json:"latest_commit"`
}

// VCSFetchRequest represents a fetch request for a VCS.
type VCSFetchRequest struct {
	CloneURL     string           `json:"clone_url"`
	Branch       string           `json:"branch"`
	AuthType     string           `json:"auth_type"`
	SSHKey       string           `json:"ssh_key"`
	TargetFolder string           `json:"target_folder"`
	Mode         string           `json:"mode"`
	RepoParam    RepositoryParams `json:"repo_param"`
}

// VCSRequestBase is the base structure for VCS requests.
type VCSRequestBase struct {
	RepoParam RepositoryParams `json:"repo_param"`
	Action    string           `json:"action"`
}

// VCSListRepositoriesRequest represents a request to list repositories.
type VCSListRepositoriesRequest struct {
	VCSRequestBase
	Language string `json:"language"`
}

// VCSRetrievePRInformationRequest represents a request to retrieve PR information.
type VCSRetrievePRInformationRequest struct {
	VCSRequestBase
}

// VCSAddRoleToPRRequest represents a request to add a role to a PR.
type VCSAddRoleToPRRequest struct {
	VCSRequestBase
	Login string `json:"login"`
	Role  string `json:"role"`
}

// VCSSetStatusOfPRRequest represents a request to set the status of a PR.
type VCSSetStatusOfPRRequest struct {
	VCSRequestBase
	Login   string `json:"login"`
	Status  string `json:"status"`
	Comment string `json:"comment"`
}

// VCSIssueCreationRequest represents a request to create a new issue.
type VCSIssueCreationRequest struct {
	VCSRequestBase
	Title string `json:"title"`
	Body  string `json:"body"`
}

// VCSIssueUpdateRequest represents a request to update an existing issue.
type VCSIssueUpdateRequest struct {
	VCSRequestBase
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"` // optional: "open" or "closed"
}

// VCSAddCommentToPRRequest represents a request to add a comment to a PR.
type VCSAddCommentToPRRequest struct {
	VCSRequestBase
	Comment   string   `json:"comment"`
	FilePaths []string `json:"file_paths"`
}

// VCSListIssuesRequest represents a request to list issues in a repository.
type VCSListIssuesRequest struct {
	VCSRequestBase
	State string `json:"state"` // open, closed, all; default open
}

// ListFuncResult holds the result of a list function.
type ListFuncResult struct {
	Args    VCSListRepositoriesRequest `json:"args"`
	Result  []RepositoryParams         `json:"result"`
	Status  string                     `json:"status"`
	Message string                     `json:"message"`
}

// VCSFetchResponse represents a response from a fetch request.
type VCSFetchResponse struct {
	Path string `json:"path"`
}

// VCSListRepositoriesResponse represents a response from listing repositories.
type VCSListRepositoriesResponse struct {
	Repositories []RepositoryParams `json:"repositories"`
}

// VCSRetrievePRInformationResponse represents a response from retrieving PR information.
type VCSRetrievePRInformationResponse struct {
	PR PRParams `json:"pr"`
}

// VCSListIssuesResponse represents a response from listing issues.
type VCSListIssuesResponse struct {
	Issues []IssueParams `json:"issues"`
}

// VCS defines the interface for VCS-related operations.
type VCS interface {
	Setup(configData config.Config) (bool, error)
	Fetch(req VCSFetchRequest) (VCSFetchResponse, error)
	ListRepositories(req VCSListRepositoriesRequest) ([]RepositoryParams, error)
	RetrievePRInformation(req VCSRetrievePRInformationRequest) (PRParams, error)
	AddRoleToPR(req VCSAddRoleToPRRequest) (bool, error)
	SetStatusOfPR(req VCSSetStatusOfPRRequest) (bool, error)
	AddCommentToPR(req VCSAddCommentToPRRequest) (bool, error)
	CreateIssue(req VCSIssueCreationRequest) (int, error)
	ListIssues(req VCSListIssuesRequest) ([]IssueParams, error)
	UpdateIssue(req VCSIssueUpdateRequest) (bool, error)
}

// VCSRPCClient implements the VCS interface for RPC clients.
type VCSRPCClient struct {
	client *rpc.Client
}

// Setup calls the Setup method on the RPC client.
func (c *VCSRPCClient) Setup(configData config.Config) (bool, error) {
	var resp bool
	err := c.client.Call("Plugin.Setup", configData, &resp)
	if err != nil {
		return false, fmt.Errorf("RPC client Setup call failed: %w", err)
	}
	return resp, nil
}

// ListRepositories calls the ListRepositories method on the RPC client.
func (c *VCSRPCClient) ListRepositories(req VCSListRepositoriesRequest) ([]RepositoryParams, error) {
	var resp VCSListRepositoriesResponse
	err := c.client.Call("Plugin.ListRepositories", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("RPC client ListRepositories call failed: %w", err)
	}
	return resp.Repositories, nil
}

// Fetch calls the Fetch method on the RPC client.
func (c *VCSRPCClient) Fetch(req VCSFetchRequest) (VCSFetchResponse, error) {
	var resp VCSFetchResponse
	err := c.client.Call("Plugin.Fetch", req, &resp)
	if err != nil {
		return resp, fmt.Errorf("RPC client Fetch call failed: %w", err)
	}
	return resp, nil
}

// RetrievePRInformation calls the RetrievePRInformation method on the RPC client.
func (c *VCSRPCClient) RetrievePRInformation(req VCSRetrievePRInformationRequest) (PRParams, error) {
	var resp VCSRetrievePRInformationResponse
	err := c.client.Call("Plugin.RetrievePRInformation", req, &resp)
	if err != nil {
		return resp.PR, fmt.Errorf("RPC client RetrievePRInformation call failed: %w", err)
	}
	return resp.PR, nil
}

// AddRoleToPR calls the AddRoleToPR method on the RPC client.
func (c *VCSRPCClient) AddRoleToPR(req VCSAddRoleToPRRequest) (bool, error) {
	var resp bool
	err := c.client.Call("Plugin.AddRoleToPR", req, &resp)
	if err != nil {
		return false, fmt.Errorf("RPC client AddRoleToPR call failed: %w", err)
	}
	return resp, nil
}

// SetStatusOfPR calls the SetStatusOfPR method on the RPC client.
func (c *VCSRPCClient) SetStatusOfPR(req VCSSetStatusOfPRRequest) (bool, error) {
	var resp bool
	err := c.client.Call("Plugin.SetStatusOfPR", req, &resp)
	if err != nil {
		return false, fmt.Errorf("RPC client SetStatusOfPR call failed: %w", err)
	}
	return resp, nil
}

// AddCommentToPR calls the AddCommentToPR method on the RPC client.
func (c *VCSRPCClient) AddCommentToPR(req VCSAddCommentToPRRequest) (bool, error) {
	var resp bool
	err := c.client.Call("Plugin.AddCommentToPR", req, &resp)
	if err != nil {
		return false, fmt.Errorf("RPC client AddCommentToPR call failed: %w", err)
	}
	return resp, nil
}

// CreateIssue calls the CreateIssue method on the RPC client.
func (c *VCSRPCClient) CreateIssue(req VCSIssueCreationRequest) (int, error) {
	var resp int
	err := c.client.Call("Plugin.CreateIssue", req, &resp)
	if err != nil {
		return 0, fmt.Errorf("RPC client CreateIssue call failed: %w", err)
	}
	return resp, nil
}

// ListIssues calls the ListIssues method on the RPC client.
func (c *VCSRPCClient) ListIssues(req VCSListIssuesRequest) ([]IssueParams, error) {
	var resp VCSListIssuesResponse
	err := c.client.Call("Plugin.ListIssues", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("RPC client ListIssues call failed: %w", err)
	}
	return resp.Issues, nil
}

// UpdateIssue calls the UpdateIssue method on the RPC client.
func (c *VCSRPCClient) UpdateIssue(req VCSIssueUpdateRequest) (bool, error) {
	var resp bool
	err := c.client.Call("Plugin.UpdateIssue", req, &resp)
	if err != nil {
		return false, fmt.Errorf("RPC client UpdateIssue call failed: %w", err)
	}
	return resp, nil
}

// VCSRPCServer wraps a VCS implementation to provide an RPC server.
type VCSRPCServer struct {
	Impl VCS
}

// Setup calls the Setup method on the VCS implementation.
func (s *VCSRPCServer) Setup(configData config.Config, resp *bool) error {
	var err error
	*resp, err = s.Impl.Setup(configData)
	if err != nil {
		return fmt.Errorf("VCS Setup failed: %w", err)
	}
	return nil
}

// Fetch calls the Fetch method on the VCS implementation.
func (s *VCSRPCServer) Fetch(args VCSFetchRequest, resp *VCSFetchResponse) error {
	var err error
	*resp, err = s.Impl.Fetch(args)
	if err != nil {
		return fmt.Errorf("VCS Fetch failed: %w", err)
	}
	return nil
}

// ListRepositories calls the ListRepositories method on the VCS implementation.
func (s *VCSRPCServer) ListRepositories(args VCSListRepositoriesRequest, resp *VCSListRepositoriesResponse) error {
	projects, err := s.Impl.ListRepositories(args)
	if err != nil {
		return fmt.Errorf("VCS ListRepositories failed: %w", err)
	}
	resp.Repositories = projects
	return nil
}

// RetrievePRInformation calls the RetrievePRInformation method on the VCS implementation.
func (s *VCSRPCServer) RetrievePRInformation(args VCSRetrievePRInformationRequest, resp *VCSRetrievePRInformationResponse) error {
	pr, err := s.Impl.RetrievePRInformation(args)
	if err != nil {
		return fmt.Errorf("VCS RetrievePRInformation failed: %w", err)
	}
	resp.PR = pr
	return nil
}

// AddRoleToPR calls the AddRoleToPR method on the VCS implementation.
func (s *VCSRPCServer) AddRoleToPR(args VCSAddRoleToPRRequest, resp *bool) error {
	var err error
	*resp, err = s.Impl.AddRoleToPR(args)
	if err != nil {
		return fmt.Errorf("VCS AddRoleToPR failed: %w", err)
	}
	return nil
}

// SetStatusOfPR calls the SetStatusOfPR method on the VCS implementation.
func (s *VCSRPCServer) SetStatusOfPR(args VCSSetStatusOfPRRequest, resp *bool) error {
	var err error
	*resp, err = s.Impl.SetStatusOfPR(args)
	if err != nil {
		return fmt.Errorf("VCS SetStatusOfPR failed: %w", err)
	}
	return nil
}

// AddCommentToPR calls the AddCommentToPR method on the VCS implementation.
func (s *VCSRPCServer) AddCommentToPR(args VCSAddCommentToPRRequest, resp *bool) error {
	var err error
	*resp, err = s.Impl.AddCommentToPR(args)
	if err != nil {
		return fmt.Errorf("VCS AddCommentToPR failed: %w", err)
	}
	return err
}

// CreateIssue calls the CreateIssue method on the VCS implementation.
func (s *VCSRPCServer) CreateIssue(args VCSIssueCreationRequest, resp *int) error {
	var err error
	*resp, err = s.Impl.CreateIssue(args)
	if err != nil {
		return fmt.Errorf("VCS CreateIssue failed: %w", err)
	}
	return nil
}

// ListIssues calls the ListIssues method on the VCS implementation.
func (s *VCSRPCServer) ListIssues(args VCSListIssuesRequest, resp *VCSListIssuesResponse) error {
	issues, err := s.Impl.ListIssues(args)
	if err != nil {
		return fmt.Errorf("VCS ListIssues failed: %w", err)
	}
	resp.Issues = issues
	return nil
}

// UpdateIssue calls the UpdateIssue method on the VCS implementation.
func (s *VCSRPCServer) UpdateIssue(args VCSIssueUpdateRequest, resp *bool) error {
	var err error
	*resp, err = s.Impl.UpdateIssue(args)
	if err != nil {
		return fmt.Errorf("VCS UpdateIssue failed: %w", err)
	}
	return nil
}

// VCSPlugin is the implementation of the plugin.Plugin interface for VCS.
type VCSPlugin struct {
	Impl VCS
}

// Server returns an RPC server for the VCS plugin.
func (p *VCSPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &VCSRPCServer{Impl: p.Impl}, nil
}

// Client returns an RPC client for the VCS plugin.
func (p *VCSPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &VCSRPCClient{client: c}, nil
}
