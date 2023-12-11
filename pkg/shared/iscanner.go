package shared

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

type Scanner interface {
	Scan(args ScannerScanRequest) (ScannerScanResponse, error)
}

type ScannerScanResult struct {
	Args    ScannerScanRequest
	Result  []string
	Status  string
	Message string
}

type ScannerScanRequest struct {
	RepoPath       string
	ReportFormat   string
	ConfigPath     string
	ResultsPath    string
	AdditionalArgs []string
}

type ScannerScanResponse struct {
	ResultsPath string
}

type ScannerRPCClient struct{ client *rpc.Client }

func (g *ScannerRPCClient) Scan(req ScannerScanRequest) (ScannerScanResponse, error) {
	var resp ScannerScanResponse

	err := g.client.Call("Plugin.Scan", req, &resp)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

type ScannerRPCServer struct {
	Impl Scanner
}

func (s *ScannerRPCServer) Scan(args ScannerScanRequest, resp *ScannerScanResult) (ScannerScanResponse, error) {
	result, err := s.Impl.Scan(args)
	return result, err
}

type ScannerPlugin struct {
	Impl Scanner
}

func (p *ScannerPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &ScannerRPCServer{Impl: p.Impl}, nil
}

func (ScannerPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ScannerRPCClient{client: c}, nil
}
