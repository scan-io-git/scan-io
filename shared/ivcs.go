package shared

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/library/vcs"
)

// VCS is the interface that we're exposing as a plugin.
type VCS interface {
	Fetch(req VCSFetchRequest) bool
	ListRepos(args VCSListReposRequest) vcs.ListFuncResult
	// ListOrgs(args VCSListReposRequest) []string
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
	Projects vcs.ListFuncResult
}

// Here is an implementation that talks over RPC
type VCSRPCClient struct{ client *rpc.Client }

func (g *VCSRPCClient) Fetch(req VCSFetchRequest) bool {
	var resp VCSFetchResponse

	err := g.client.Call("Plugin.Fetch", req, &resp)

	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		panic(err)
	}

	return resp.Success
}

func (g *VCSRPCClient) ListRepos(req VCSListReposRequest) vcs.ListFuncResult {
	var resp VCSListReposResponse

	err := g.client.Call("Plugin.ListRepos", req, &resp)

	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		panic(err)
	}

	return resp.Projects
}

// Here is the RPC server that VCSRPCClient talks to, conforming to
// the requirements of net/rpc
type VCSRPCServer struct {
	// This is the real implementation
	Impl VCS
}

func (s *VCSRPCServer) Fetch(args VCSFetchRequest, resp *VCSFetchResponse) string {
	resp.Success = s.Impl.Fetch(args)
	return "asdf"
}

func (s *VCSRPCServer) ListRepos(args VCSListReposRequest, resp *VCSListReposResponse) error {
	resp.Projects = s.Impl.ListRepos(args)
	return nil
}

// This is the implementation of plugin.Plugin so we can serve/consume this
//
// This has two methods: Server must return an RPC server for this plugin
// type. We construct a VCSRPCServer for this.
//
// Client must return an implementation of our interface that communicates
// over an RPC client. We return VCSRPCClient for this.
//
// Ignore MuxBroker. That is used to create more multiplexed streams on our
// plugin connection and is a more advanced use case.
type VCSPlugin struct {
	// Impl Injection
	Impl VCS
}

func (p *VCSPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &VCSRPCServer{Impl: p.Impl}, nil
}

func (VCSPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &VCSRPCClient{client: c}, nil
}
