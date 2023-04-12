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

type ScannerTrufflehog3 struct {
	logger hclog.Logger
}

func (g *ScannerTrufflehog3) Scan(args shared.ScannerScanRequest) error {
	g.logger.Info("Scan is starting", "project", args.RepoPath)
	g.logger.Debug("Debug info", "args", args)
	var commandArgs []string
	var cmd *exec.Cmd
	var stdBuffer bytes.Buffer

	// Add additional arguments
	if len(args.AdditionalArgs) != 0 {
		commandArgs = append(commandArgs, args.AdditionalArgs...)
	}

	// Trufflehog3 --rules is a rules file that contains regexes that might trigger
	// --config it's a flag for .trufflehog3.yml file with wider configuration rather than rules
	// .trufflehog3.yml will be found automatically in root of your folder
	if args.ConfigPath != "" {
		commandArgs = append(commandArgs, "--rules", args.ConfigPath)
	}

	if args.ReportFormat != "" {
		commandArgs = append(commandArgs, "--format", args.ReportFormat)
	}

	// Here we added -z flag because Trufflehog3 send a not correct exit code even when it finished without errors
	commandArgs = append(commandArgs, "-z", "--output", args.ResultsPath, args.RepoPath)

	cmd = exec.Command("trufflehog3", commandArgs...)
	g.logger.Debug("Debug info", "cmd", cmd.Args)

	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: false,
	}), &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	err := cmd.Run()
	if err != nil {
		err := fmt.Errorf(stdBuffer.String())

		g.logger.Error("Trufflehog3 execution error", "err", err)
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

	Scanner := &ScannerTrufflehog3{
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
