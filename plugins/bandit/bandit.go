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
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// TODO: Wrap it in a custom error handler to add to the stack trace.
// Metadata of the plugin
var (
	Version       = "unknown"
	GolangVersion = "unknown"
	BuildTime     = "unknown"
)

// ScannerBandit represents the Bandit scanner with its configuration and logger.
type ScannerBandit struct {
	logger       hclog.Logger
	globalConfig *config.Config
}

// newScannerBandit creates a new instance of ScannerBandit.
func newScannerBandit(logger hclog.Logger) *ScannerBandit {
	return &ScannerBandit{
		logger: logger,
	}
}

// setGlobalConfig sets the global configuration for the ScannerBandit instance.
func (g *ScannerBandit) setGlobalConfig(globalConfig *config.Config) {
	g.globalConfig = globalConfig
}

// buildCommandArgs constructs the command-line arguments for the Bandit command.
func (g *ScannerBandit) buildCommandArgs(args shared.ScannerScanRequest) []string {
	var commandArgs []string

	appendArg := func(arg ...string) {
		commandArgs = append(commandArgs, arg...)
	}

	if len(args.AdditionalArgs) != 0 {
		appendArg(args.AdditionalArgs...)
	}

	if args.ReportFormat != "" {
		appendArg("-f", args.ReportFormat)
	}

	if args.ConfigPath != "" {
		appendArg("-c", args.ConfigPath)
	}

	appendArg("-r", "-o", args.ResultsPath)
	appendArg(args.TargetPath)

	return commandArgs
}

// Scan executes the Bandit scan with the provided arguments and returns the scan response.
func (g *ScannerBandit) Scan(args shared.ScannerScanRequest) (shared.ScannerScanResponse, error) {
	var result shared.ScannerScanResponse
	g.logger.Info("scan is starting", "project", args.TargetPath)
	g.logger.Debug("debug info", "args", args)

	if err := g.validateScan(&args); err != nil {
		g.logger.Error("validation failed for scan operation", "error", err)
		return result, err
	}

	commandArgs := g.buildCommandArgs(args)

	cmd := exec.Command("bandit", commandArgs...)
	g.logger.Debug("debug info", "cmd", cmd.Args)

	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}), &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	if err := cmd.Run(); err != nil {
		g.logger.Error("bandit execution error", "error", err)
		return result, fmt.Errorf("bandit execution error: %w. Output: %s", err, stdBuffer.String())
	}
	result.ResultsPath = args.ResultsPath
	g.logger.Info("scan finished", "project", args.TargetPath)
	g.logger.Info("result saved", "path", args.ResultsPath)
	g.logger.Debug("debug info", "project", args.TargetPath, "config", args.ConfigPath, "resultsFile", args.ResultsPath, "cmd", cmd.Args)
	return result, nil
}

// Setup initializes the global configuration for the ScannerBandit instance.
func (g *ScannerBandit) Setup(configData config.Config) (bool, error) {
	g.setGlobalConfig(&configData)
	return true, nil
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	banditInstance := newScannerBandit(logger)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			shared.PluginTypeScanner: &shared.ScannerPlugin{Impl: banditInstance},
		},
		Logger: logger,
	})
}
