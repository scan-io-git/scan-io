package vcs

import (
	"github.com/hashicorp/go-plugin"
	"net/rpc"
)

type RepositoryParams struct {
	Namespace string
	RepoName  string
	HttpLink  string
	SshLink   string
}

type ProjectParams struct {
	Key  string
	Name string
	Link string
}

type ListFuncResult struct {
	Args    VCSListReposRequest
	Result  []RepositoryParams
	Status  string
	Message string
}

type EvnVariables struct {
	Username, Token, VcsPort, SshKeyPassword string
}

type VCS interface {
	Fetch(req VCSFetchRequest) bool
	ListRepos(args VCSListReposRequest) ([]RepositoryParams, error)
}

type VCSFetchRequest struct {
	Project      string
	AuthType     string
	SSHKey       string
	VCSURL       string
	TargetFolder string
}

type VCSFetchResponse struct {
	Success bool
}

type VCSListReposRequest struct {
	Namespace string
	VCSURL    string
	Limit     int
}

type VCSListReposResponse struct {
	Repositories []RepositoryParams
}

type VCSRPCClient struct{ client *rpc.Client }

func (g *VCSRPCClient) Fetch(req VCSFetchRequest) bool {
	var resp VCSFetchResponse

	err := g.client.Call("Plugin.Fetch", req, &resp)

	if err != nil {
		panic(err)
	}

	return resp.Success
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

func (s *VCSRPCServer) Fetch(args VCSFetchRequest, resp *VCSFetchResponse) string {
	resp.Success = s.Impl.Fetch(args)
	return "asdf"
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
