package shared

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

// VCS is the interface that we're exposing as a plugin.
type VCS interface {
	Fetch(projects []string) string
	ListProjects(organization string) string
}

// Here is an implementation that talks over RPC
type VCSRPCClient struct{ client *rpc.Client }

func (g *VCSRPCClient) Fetch(projects []string) string {
	var resp string
	// err := g.client.Call("Plugin.Fetch", new(interface{}), &resp)
	err := g.client.Call("Plugin.Fetch", map[string]interface{}{
		"projects": projects,
	}, &resp)

	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		panic(err)
	}

	return resp
}

func (g *VCSRPCClient) ListProjects(organization string) string {
	var resp string

	err := g.client.Call("Plugin.ListProjects", map[string]interface{}{
		"organization": organization,
	}, &resp)

	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		panic(err)
	}

	return resp
}

// Here is the RPC server that VCSRPCClient talks to, conforming to
// the requirements of net/rpc
type VCSRPCServer struct {
	// This is the real implementation
	Impl VCS
}

func (s *VCSRPCServer) Fetch(args map[string]interface{}, resp *string) error {
	*resp = s.Impl.Fetch(args["projects"].([]string))
	return nil
}

func (s *VCSRPCServer) ListProjects(args map[string]interface{}, resp *string) error {
	*resp = s.Impl.ListProjects(args["organization"].(string))
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
