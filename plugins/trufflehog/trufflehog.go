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

// ScannerTrufflehog represents the Trufflehog scanner with its configuration and logger.
type ScannerTrufflehog struct {
	logger       hclog.Logger
	globalConfig *config.Config
}

// newScannerTrufflehog creates a new instance of ScannerTrufflehog.
func newScannerTrufflehog(logger hclog.Logger) *ScannerTrufflehog {
	return &ScannerTrufflehog{
		logger: logger,
	}
}

// setGlobalConfig sets the global configuration for the ScannerTrufflehog instance.
func (g *ScannerTrufflehog) setGlobalConfig(globalConfig *config.Config) {
	g.globalConfig = globalConfig
}

// buildCommandArgs constructs the command-line arguments for the Trufflehog command.
func (g *ScannerTrufflehog) buildCommandArgs(args shared.ScannerScanRequest) []string {
	var commandArgs []string

	appendArg := func(arg ...string) {
		commandArgs = append(commandArgs, arg...)
	}

	if args.ConfigPath != "" {
		appendArg("--config", args.ConfigPath)
	}

	if args.ReportFormat != "" {
		g.validateFormatSoft(args.ReportFormat)
		appendArg(fmt.Sprintf("--%v", args.ReportFormat))
	}

	appendArg("--no-verification", "filesystem")

	// additional arguments should be added after command name
	// ref: https://github.com/scan-io-git/scan-io/issues/86
	if len(args.AdditionalArgs) != 0 {
		appendArg(args.AdditionalArgs...)
	}

	appendArg(args.TargetPath)
	return commandArgs
}

// Scan executes the Trufflehog scan with the provided arguments and returns the scan response.
func (g *ScannerTrufflehog) Scan(args shared.ScannerScanRequest) (shared.ScannerScanResponse, error) {
	var result shared.ScannerScanResponse
	g.logger.Info("scan is starting", "project", args.TargetPath)
	g.logger.Debug("debug info", "args", args)

	if err := g.validateScan(&args); err != nil {
		g.logger.Error("validation failed for scan operation", "error", err)
		return result, err
	}

	commandArgs := g.buildCommandArgs(args)

	cmd := exec.Command("trufflehog", commandArgs...)
	g.logger.Debug("debug info", "cmd", cmd.Args)

	// Trufflehog doesn't support writing results to a file, only to stdout
	// writing stdout to a file with results
	resultsFile, err := os.Create(args.ResultsPath)
	if err != nil {
		return result, fmt.Errorf("failed to create results file: %w", err)
	}
	defer resultsFile.Close()

	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}), &stdBuffer, resultsFile)

	cmd.Stdout = mw
	cmd.Stderr = mw

	if err := cmd.Run(); err != nil {
		g.logger.Error("trufflehog execution error", "error", err)
		return result, fmt.Errorf("trufflehog execution error: %w. Output: %s", err, stdBuffer.String())
	}
	result.ResultsPath = args.ResultsPath
	g.logger.Info("scan finished", "project", args.TargetPath)
	g.logger.Info("result saved", "path", args.ResultsPath)
	g.logger.Debug("debug info", "project", args.TargetPath, "config", args.ConfigPath, "resultsFile", args.ResultsPath, "cmd", cmd.Args)
	return result, nil
}

// Setup initializes the global configuration for the ScannerTrufflehog instance.
func (g *ScannerTrufflehog) Setup(configData config.Config) (bool, error) {
	g.setGlobalConfig(&configData)
	return true, nil
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	trufflehogInstance := newScannerTrufflehog(logger)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			shared.PluginTypeScanner: &shared.ScannerPlugin{Impl: trufflehogInstance},
		},
		Logger: logger,
	})
}
