package main

import (
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/shared"
)

// Here is a real implementation of Scanner
type ScannerSemgrep struct {
	logger hclog.Logger
}

func (g *ScannerSemgrep) Scan(args shared.ScannerScanRequest) bool {

	cmd := exec.Command("semgrep", "--config", "auto", "-o", args.ResultsPath, "--sarif", args.RepoPath)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	g.logger.Info("Scan finished", "project", args.RepoPath, "resultsFile", args.ResultsPath)

	if err != nil {
		g.logger.Error("semgrep execution error", "err", err, "projectFolder", args.RepoPath)
		return false
	}

	return true
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	Scanner := &ScannerSemgrep{
		logger: logger,
	}
	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		"scanner": &shared.ScannerPlugin{Impl: Scanner},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
