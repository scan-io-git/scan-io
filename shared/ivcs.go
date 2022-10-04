package shared

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

// VCS is the interface that we're exposing as a plugin.
type VCS interface {
	Fetch(project string) bool
	ListAllRepos(organization string) []string
}

type VCSFetchResponse struct {
	Success bool
}

// type VCSListProjectsRequest struct {
// 	Organization string
// 	VCSURL       string
// }

type VCSListAllReposResponse struct {
	Projects []string
}

// Here is an implementation that talks over RPC
type VCSRPCClient struct{ client *rpc.Client }

func (g *VCSRPCClient) Fetch(project string) bool {
	var resp VCSFetchResponse

	err := g.client.Call("Plugin.Fetch", map[string]interface{}{
		"project": project,
	}, &resp)

	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		panic(err)
	}

	return resp.Success
}

func (g *VCSRPCClient) ListAllRepos(organization string) []string {
	var resp VCSListAllReposResponse

	err := g.client.Call("Plugin.ListAllRepos", map[string]interface{}{
		"organization": organization,
	}, &resp)

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

func (s *VCSRPCServer) Fetch(args map[string]interface{}, resp *VCSFetchResponse) error {
	resp.Success = s.Impl.Fetch(args["project"].(string))
	return nil
}

func (s *VCSRPCServer) ListAllRepos(args map[string]interface{}, resp *VCSListAllReposResponse) error {
	resp.Projects = s.Impl.ListAllRepos(args["organization"].(string))
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
