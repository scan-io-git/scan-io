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

type ScannerTrufflehog struct {
	logger hclog.Logger
}

func (g *ScannerTrufflehog) Scan(args shared.ScannerScanRequest) error {
	g.logger.Info("Scan is starting", "project", args.RepoPath, "config", args.ConfigPath, "resultsFile", args.ResultsPath)
	var commandArgs []string
	var cmd *exec.Cmd
	var stdBuffer bytes.Buffer

	// Add additional arguments
	if len(args.AdditionalArgs) != 0 {
		commandArgs = append(commandArgs, args.AdditionalArgs...)
	}

	if args.ConfigPath != "" {
		commandArgs = append(commandArgs, "--config", args.ConfigPath)
	}

	if args.ReportFormat != "" && args.ReportFormat == "json" {
		reportFormat := fmt.Sprintf("--%v", args.ReportFormat)
		commandArgs = append(commandArgs, reportFormat)
	} else if args.ReportFormat != "" {
		g.logger.Warn("Trufflehog supports only a json non default format. Will be used default format instead of your reportFormat", "reportFormat", args.ReportFormat)
	}

	commandArgs = append(commandArgs, "--no-verification", "filesystem", args.RepoPath)

	cmd = exec.Command("trufflehog", commandArgs...)
	g.logger.Info("cmd", cmd.Args)

	// trufflehog doesn't support writing results in file, only to stdout
	// writing stdout to a file with results
	resultsFile, err := os.Create(args.ResultsPath)
	if err != nil {
		return err
	}
	defer resultsFile.Close()

	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}), &stdBuffer, resultsFile)

	cmd.Stdout = mw
	cmd.Stderr = mw

	err = cmd.Run()
	if err != nil {
		err := fmt.Errorf(stdBuffer.String())

		g.logger.Error("Trufflehog execution error", "err", err)
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

	Scanner := &ScannerTrufflehog{
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
