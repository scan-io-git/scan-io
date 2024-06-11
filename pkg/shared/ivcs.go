package shared

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

type Args interface {
}

type RepositoryParams struct {
	Namespace string `json:"namespace"`
	RepoName  string `json:"repo_name"`
	PRID      string `json:"pr_id,omitempty"`
	VCSURL    string `json:"vcs_url,omitempty"`
	HttpLink  string `json:"http_link,omitempty"`
	SshLink   string `json:"ssh_link"`
}

type ProjectParams struct {
	Key  string
	Name string
	Link string
}

type PRParams struct {
	Id          int       `json:"id"`
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

type User struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type Reference struct {
	ID           string
	DisplayId    string
	LatestCommit string
}

type VCSListRepositoriesRequest struct {
	Namespace  string
	VCSURL     string
	Repository string
	Language   string
}

type VCSFetchRequest struct {
	CloneURL     string
	Branch       string
	AuthType     string
	SSHKey       string
	TargetFolder string
	Mode         string
	RepoParam    RepositoryParams
}

type VCSRequestBase struct {
	Namespace     string
	VCSURL        string
	Action        string
	Repository    string
	PullRequestId int
}

type VCSRetrievePRInformationRequest struct {
	VCSRequestBase
}

type VCSAddRoleToPRRequest struct {
	VCSRequestBase
	Login string
	Role  string
}

type VCSSetStatusOfPRRequest struct {
	VCSRequestBase
	Login  string
	Status string
}

type VCSAddCommentToPRRequest struct {
	VCSRequestBase
	Comment   string
	FilePaths []string
}

type Result interface {
}

type ListFuncResult struct {
	Args    VCSListRepositoriesRequest `json:"args"`
	Result  []RepositoryParams         `json:"result"`
	Status  string                     `json:"status"`
	Message string                     `json:"message"`
}

type VCSFetchResponse struct {
	Path string
}

type VCSListRepositoriesResponse struct {
	Repositories []RepositoryParams
}

type VCSRetrievePRInformationResponse struct {
	PR PRParams
}

type VCS interface {
	Setup(configData config.Config) (bool, error)
	Fetch(req VCSFetchRequest) (VCSFetchResponse, error)
	ListRepositories(args VCSListRepositoriesRequest) ([]RepositoryParams, error)
	RetrievePRInformation(req VCSRetrievePRInformationRequest) (PRParams, error)
	AddRoleToPR(req VCSAddRoleToPRRequest) (bool, error)
	SetStatusOfPR(req VCSSetStatusOfPRRequest) (bool, error)
	AddCommentToPR(req VCSAddCommentToPRRequest) (bool, error)
}

type VCSRPCClient struct{ client *rpc.Client }

func (g *VCSRPCClient) Setup(configData config.Config) (bool, error) {
	var resp bool
	err := g.client.Call("Plugin.Setup", configData, &resp)
	if err != nil {
		return false, err
	}
	return resp, nil
}

func (g *VCSRPCClient) ListRepositories(req VCSListRepositoriesRequest) ([]RepositoryParams, error) {
	var resp VCSListRepositoriesResponse

	err := g.client.Call("Plugin.ListRepositories", req, &resp)

	if err != nil {
		return resp.Repositories, err
	}

	return resp.Repositories, nil
}

func (g *VCSRPCClient) Fetch(req VCSFetchRequest) (VCSFetchResponse, error) {
	var resp VCSFetchResponse

	err := g.client.Call("Plugin.Fetch", req, &resp)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (g *VCSRPCClient) RetrievePRInformation(req VCSRetrievePRInformationRequest) (PRParams, error) {
	var resp VCSRetrievePRInformationResponse

	err := g.client.Call("Plugin.RetrievePRInformation", req, &resp)

	if err != nil {
		return resp.PR, err
	}

	return resp.PR, nil
}

func (g *VCSRPCClient) AddRoleToPR(req VCSAddRoleToPRRequest) (bool, error) {
	var resp VCSRetrievePRInformationResponse

	err := g.client.Call("Plugin.AddRoleToPR", req, &resp)

	if err != nil {
		return false, err
	}

	return true, nil
}

func (g *VCSRPCClient) SetStatusOfPR(req VCSSetStatusOfPRRequest) (bool, error) {
	var resp VCSRetrievePRInformationResponse

	err := g.client.Call("Plugin.SetStatusOfPR", req, &resp)

	if err != nil {
		return false, err
	}

	return true, nil
}

func (g *VCSRPCClient) AddCommentToPR(req VCSAddCommentToPRRequest) (bool, error) {
	var resp VCSRetrievePRInformationResponse

	err := g.client.Call("Plugin.AddCommentToPR", req, &resp)

	if err != nil {
		return false, err
	}

	return true, nil
}

type VCSRPCServer struct {
	Impl VCS
}

func (s *VCSRPCServer) Setup(configData config.Config, resp *bool) error {
	var err error
	*resp, err = s.Impl.Setup(configData)
	return err
}

func (s *VCSRPCServer) Fetch(args VCSFetchRequest, resp *VCSFetchResponse) error {
	var err error
	*resp, err = s.Impl.Fetch(args)
	return err
}

func (s *VCSRPCServer) ListRepositories(args VCSListRepositoriesRequest, resp *VCSListRepositoriesResponse) error {
	projects, err := s.Impl.ListRepositories(args)
	resp.Repositories = projects
	return err
}

func (s *VCSRPCServer) RetrievePRInformation(args VCSRetrievePRInformationRequest, resp *VCSRetrievePRInformationResponse) error {
	pr, err := s.Impl.RetrievePRInformation(args)
	resp.PR = pr
	return err
}

func (s *VCSRPCServer) AddRoleToPR(args VCSAddRoleToPRRequest, resp *VCSRetrievePRInformationResponse) error {
	_, err := s.Impl.AddRoleToPR(args)
	if err != nil {
		return err
	}
	return err
}

func (s *VCSRPCServer) SetStatusOfPR(args VCSSetStatusOfPRRequest, resp *VCSRetrievePRInformationResponse) error {
	a, err := s.Impl.SetStatusOfPR(args)
	if a == false {

	}
	return err
}

func (s *VCSRPCServer) AddCommentToPR(args VCSAddCommentToPRRequest, resp *VCSRetrievePRInformationResponse) error {
	a, err := s.Impl.AddCommentToPR(args)
	if a == false {

	}
	return err
}

type VCSPlugin struct {
	Impl VCS
}

func (p *VCSPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &VCSRPCServer{Impl: p.Impl}, nil
}

func (VCSPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &VCSRPCClient{client: c}, nil
}
