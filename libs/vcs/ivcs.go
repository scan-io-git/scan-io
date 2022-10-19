package vcs

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

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

type VCSListReposRequest struct {
	Namespace string
	VCSURL    string
}

type VCSFetchRequest struct {
	Repository   string
	AuthType     string
	SSHKey       string
	VCSURL       string
	TargetFolder string
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

type EvnVariables struct {
	Username, Token, VcsPort, SshKeyPassword string
}

type VCSFetchResponse struct {
	Dummy bool
}

type VCSListReposResponse struct {
	Repositories []RepositoryParams
}

type VCS interface {
	Fetch(req VCSFetchRequest) error
	ListRepos(args VCSListReposRequest) ([]RepositoryParams, error)
}

type VCSRPCClient struct{ client *rpc.Client }

func (g *VCSRPCClient) Fetch(req VCSFetchRequest) error {
	var resp VCSFetchResponse
	// var resp bool

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

type VCSRPCServer struct {
	Impl VCS
}

func (s *VCSRPCServer) Fetch(args VCSFetchRequest, resp *VCSFetchResponse) error {
	return s.Impl.Fetch(args)
	// if resp.Error != nil {
	// 	return resp.Error
	// }
	// return nil
}

func (s *VCSRPCServer) ListRepos(args VCSListReposRequest, resp *VCSListReposResponse) error {
	projects, err := s.Impl.ListRepos(args)
	resp.Repositories = projects
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
