package shared

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

type Args interface {
}

type RepositoryParams struct {
	Namespace string `json:"namespace"`
	RepoName  string `json:"repo_name"`
	HttpLink  string `json:"http_link"`
	SshLink   string `json:"ssh_link"`
}

type ProjectParams struct {
	Key  string
	Name string
	Link string
}

type PRParams struct {
	PullRequestId int
	Title         string
	Description   string
	State         string
	AuthorEmail   string
	AuthorName    string
	SelfLink      string
	CreatedDate   int64
	UpdatedDate   int64
	FromRef       RefPRInf
	ToRef         RefPRInf
}

type RefPRInf struct {
	ID           string
	LatestCommit string
}

type VCSListReposRequest struct {
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
}

type VCSRequestBase struct {
	Namespace     string
	VCSURL        string
	Action        string
	Repository    string
	PullRequestId int
}

type VCSRetrivePRInformationRequest struct {
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
	Comment string
}

type Result interface {
}

type ListFuncResult struct {
	Args    VCSListReposRequest `json:"args"`
	Result  []RepositoryParams  `json:"result"`
	Status  string              `json:"status"`
	Message string              `json:"message"`
}

type FetchFuncResult struct {
	Args    VCSFetchRequest
	Result  []string
	Status  string
	Message string
}

type GenericResult struct {
	Args    interface{} `json:"args"`
	Result  interface{} `json:"result"`
	Status  string      `json:"status"`
	Message string      `json:"message"`
}

type EvnVariables struct {
	Username, Token, VcsPort, SshKeyPassword string
}

type VCSFetchResponse struct {
	Dummy bool
}

type VCSListReposResponse struct {
	Repositories []RepositoryParams
}

type VCSRetrivePRInformationResponse struct {
	PR PRParams
}

type VCS interface {
	Fetch(req VCSFetchRequest) error
	ListRepos(args VCSListReposRequest) ([]RepositoryParams, error)
	RetrivePRInformation(req VCSRetrivePRInformationRequest) (PRParams, error)
	AddRoleToPR(req VCSAddRoleToPRRequest) (bool, error)
	SetStatusOfPR(req VCSSetStatusOfPRRequest) (bool, error)
	AddComment(req VCSAddCommentToPRRequest) (bool, error)
}

type VCSRPCClient struct{ client *rpc.Client }

func (g *VCSRPCClient) Fetch(req VCSFetchRequest) error {
	var resp VCSFetchResponse

	err := g.client.Call("Plugin.Fetch", req, &resp)

	if err != nil {
		return err
	}

	return nil
}

func (g *VCSRPCClient) ListRepos(req VCSListReposRequest) ([]RepositoryParams, error) {
	var resp VCSListReposResponse

	err := g.client.Call("Plugin.ListRepos", req, &resp)

	if err != nil {
		return resp.Repositories, err
	}

	return resp.Repositories, nil
}

func (g *VCSRPCClient) RetrivePRInformation(req VCSRetrivePRInformationRequest) (PRParams, error) {
	var resp VCSRetrivePRInformationResponse

	err := g.client.Call("Plugin.RetrivePRInformation", req, &resp)

	if err != nil {
		return resp.PR, err
	}

	return resp.PR, nil
}

func (g *VCSRPCClient) AddRoleToPR(req VCSAddRoleToPRRequest) (bool, error) {
	var resp VCSRetrivePRInformationResponse

	err := g.client.Call("Plugin.AddRoleToPR", req, &resp)

	if err != nil {
		return false, err
	}

	return true, nil
}

func (g *VCSRPCClient) SetStatusOfPR(req VCSSetStatusOfPRRequest) (bool, error) {
	var resp VCSRetrivePRInformationResponse

	err := g.client.Call("Plugin.SetStatusOfPR", req, &resp)

	if err != nil {
		return false, err
	}

	return true, nil
}

func (g *VCSRPCClient) AddComment(req VCSAddCommentToPRRequest) (bool, error) {
	var resp VCSRetrivePRInformationResponse

	err := g.client.Call("Plugin.AddComment", req, &resp)

	if err != nil {
		return false, err
	}

	return true, nil
}

type VCSRPCServer struct {
	Impl VCS
}

func (s *VCSRPCServer) Fetch(args VCSFetchRequest, resp *VCSFetchResponse) error {
	return s.Impl.Fetch(args)
}

func (s *VCSRPCServer) ListRepos(args VCSListReposRequest, resp *VCSListReposResponse) error {
	projects, err := s.Impl.ListRepos(args)
	resp.Repositories = projects
	return err
}

func (s *VCSRPCServer) RetrivePRInformation(args VCSRetrivePRInformationRequest, resp *VCSRetrivePRInformationResponse) error {
	pr, err := s.Impl.RetrivePRInformation(args)
	resp.PR = pr
	return err
}

func (s *VCSRPCServer) AddRoleToPR(args VCSAddRoleToPRRequest, resp *VCSRetrivePRInformationResponse) error {
	a, err := s.Impl.AddRoleToPR(args)
	if a == false {

	}
	return err
}

func (s *VCSRPCServer) SetStatusOfPR(args VCSSetStatusOfPRRequest, resp *VCSRetrivePRInformationResponse) error {
	a, err := s.Impl.SetStatusOfPR(args)
	if a == false {

	}
	return err
}

func (s *VCSRPCServer) AddComment(args VCSAddCommentToPRRequest, resp *VCSRetrivePRInformationResponse) error {
	a, err := s.Impl.AddComment(args)
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
