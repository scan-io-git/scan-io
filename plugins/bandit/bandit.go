package main

import (
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/pkg/shared"
)

// Here is a real implementation of Scanner
type ScannerSemgrep struct {
	logger hclog.Logger
}

func (g *ScannerSemgrep) Scan(args shared.ScannerScanRequest) error {

	cmd := exec.Command("bandit", "-r", "-f", "json", "-o", args.ResultsPath, args.RepoPath)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	g.logger.Info("Scan finished", "RepoPath", args.RepoPath, "ResultsPath", args.ResultsPath)

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() != 1 {
			g.logger.Error("scanner execution error", "err", err, "RepoPath", args.RepoPath, "exitError.ExitCode()", exitError.ExitCode())
			return err
		}
	}

	return nil
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
		shared.PluginTypeScanner: &shared.ScannerPlugin{Impl: Scanner},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
