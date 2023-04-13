package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/pkg/shared"
)

type ScannerSemgrep struct {
	logger hclog.Logger
}

func (g *ScannerSemgrep) Scan(args shared.ScannerScanRequest) error {
	g.logger.Info("Scan is starting", "project", args.RepoPath)
	g.logger.Debug("Debug info", "args", args)

	var commandArgs []string
	var cmd *exec.Cmd
	var reportFormat string
	var stdBuffer bytes.Buffer

	commandArgs = []string{"scan"}

	// Add additional arguments
	if len(args.AdditionalArgs) != 0 {
		commandArgs = append(commandArgs, args.AdditionalArgs...)
	}

	if args.ReportFormat != "" {
		reportFormat = fmt.Sprintf("--%v", args.ReportFormat)
		commandArgs = append(commandArgs, reportFormat)
	}

	if args.ConfigPath == "" {
		commandArgs = append(commandArgs, "-f", "auto", "--output", args.ResultsPath, args.RepoPath)
	} else {
		commandArgs = append(commandArgs, "--metrics", "off", "-f", args.ConfigPath, "--output", args.ResultsPath, args.RepoPath)
	}

	cmd = exec.Command("semgrep", commandArgs...)
	g.logger.Debug("Debug info", "cmd", cmd.Args)

	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}), &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	err := cmd.Run()
	if err != nil {
		err := fmt.Errorf(stdBuffer.String())
		g.logger.Error("Semgrep execution error", "error", err)
		return err
	}
	g.logger.Info("Scan finished for", "project", args.RepoPath)
	g.logger.Info("Result is saved to", "path to a result file", args.ResultsPath)
	g.logger.Debug("Debug info", "project", args.RepoPath, "config", args.ConfigPath, "resultsFile", args.ResultsPath, "cmd", cmd.Args)
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

	var pluginMap = map[string]plugin.Plugin{
		shared.PluginTypeScanner: &shared.ScannerPlugin{Impl: Scanner},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
