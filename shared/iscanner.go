package shared

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

// Scanner is the interface that we're exposing as a plugin.
type Scanner interface {
	Scan(project string) bool
}

type ScannerFetchResponse struct {
	Success bool
}

// Here is an implementation that talks over RPC
type ScannerRPCClient struct{ client *rpc.Client }

func (g *ScannerRPCClient) Scan(project string) bool {
	var resp ScannerFetchResponse

	err := g.client.Call("Plugin.Scan", map[string]interface{}{
		"project": project,
	}, &resp)

	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		panic(err)
	}

	return resp.Success
}

// Here is the RPC server that ScannerRPCClient talks to, conforming to
// the requirements of net/rpc
type ScannerRPCServer struct {
	// This is the real implementation
	Impl Scanner
}

func (s *ScannerRPCServer) Scan(args map[string]interface{}, resp *ScannerFetchResponse) error {
	resp.Success = s.Impl.Scan(args["project"].(string))
	return nil
}

// This is the implementation of plugin.Plugin so we can serve/consume this
//
// This has two methods: Server must return an RPC server for this plugin
// type. We construct a ScannerRPCServer for this.
//
// Client must return an implementation of our interface that communicates
// over an RPC client. We return ScannerRPCClient for this.
//
// Ignore MuxBroker. That is used to create more multiplexed streams on our
// plugin connection and is a more advanced use case.
type ScannerPlugin struct {
	// Impl Injection
	Impl Scanner
}

func (p *ScannerPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &ScannerRPCServer{Impl: p.Impl}, nil
}

func (ScannerPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ScannerRPCClient{client: c}, nil
}
