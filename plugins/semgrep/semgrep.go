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
	g.logger.Info("Scan is starting", "project", args.RepoPath, "config", args.ConfigPath, "resultsFile", args.ResultsPath)
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
	}

	if args.ConfigPath == "auto" {
		commandArgs = append(commandArgs, "-f", args.ConfigPath, "--output", args.ResultsPath, reportFormat, args.RepoPath)
	} else {
		commandArgs = append(commandArgs, "--metrics", "off", "-f", args.ConfigPath, "--output", args.ResultsPath, reportFormat, args.RepoPath)
	}

	cmd = exec.Command("semgrep", commandArgs...)
	g.logger.Info("cmd", cmd.Args)
	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}), &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	err := cmd.Run()
	if err != nil {
		err := fmt.Errorf(stdBuffer.String())

		g.logger.Error("Semgrep execution error", "err", err)
		return err
	}
	g.logger.Info("Scan finished", "project", args.RepoPath, "config", args.ConfigPath, "resultsFile", args.ResultsPath, "cmd", cmd.Args)
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
