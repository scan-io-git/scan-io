package shared

import (
	"fmt"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// Scanner defines the interface for scanner-related operations.
type Scanner interface {
	Setup(configData config.Config) (bool, error)
	Scan(args ScannerScanRequest) (ScannerScanResponse, error)
}

// ScannerScanRequest represents a single scan request.
type ScannerScanRequest struct {
	TargetPath     string   `json:"target_path"`     // Path to the target to scan
	ResultsPath    string   `json:"results_path"`    // Path to save the results of the scan
	ConfigPath     string   `json:"config_path"`     // Path to the configuration file for the scanner
	ReportFormat   string   `json:"report_path"`     // Format of the report to generate (e.g., JSON, Sarif)
	AdditionalArgs []string `json:"additional_args"` // Additional arguments for the scanner
}

// ScannerScanResponse represents the response from a scan plugin.
type ScannerScanResponse struct {
	ResultsPath string `json:"results_path"` // Path to the saved results of the scan
}

// ScannerRPCClient implements the Scanner interface for RPC clients.
type ScannerRPCClient struct {
	client *rpc.Client
}

// Setup calls the Setup method on the RPC client.
func (c *ScannerRPCClient) Setup(configData config.Config) (bool, error) {
	var resp bool
	err := c.client.Call("Plugin.Setup", configData, &resp)
	if err != nil {
		return false, fmt.Errorf("RPC client Setup call failed: %w", err)
	}
	return resp, nil
}

// Scan calls the Scan method on the RPC client.
func (c *ScannerRPCClient) Scan(req ScannerScanRequest) (ScannerScanResponse, error) {
	var resp ScannerScanResponse
	err := c.client.Call("Plugin.Scan", req, &resp)
	if err != nil {
		return resp, fmt.Errorf("RPC client Scan call failed: %w", err)
	}
	return resp, nil
}

// ScannerRPCServer wraps a Scanner implementation to provide an RPC server.
type ScannerRPCServer struct {
	Impl Scanner
}

// Setup calls the Setup method on the Scanner implementation.
func (s *ScannerRPCServer) Setup(configData config.Config, resp *bool) error {
	var err error
	*resp, err = s.Impl.Setup(configData)
	if err != nil {
		return fmt.Errorf("Scanner Setup failed: %w", err)
	}
	return nil
}

// Scan calls the Scan method on the Scanner implementation.
func (s *ScannerRPCServer) Scan(args ScannerScanRequest, resp *ScannerScanResponse) error {
	var err error
	*resp, err = s.Impl.Scan(args)
	if err != nil {
		return fmt.Errorf("Scanner Scan failed: %w", err)
	}
	return nil
}

// ScannerPlugin is the implementation of the plugin.Plugin interface for scanners.
type ScannerPlugin struct {
	Impl Scanner
}

// Server returns an RPC server for the Scanner plugin.
func (p *ScannerPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &ScannerRPCServer{Impl: p.Impl}, nil
}

// Client returns an RPC client for the Scanner plugin.
func (p *ScannerPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ScannerRPCClient{client: c}, nil
}
