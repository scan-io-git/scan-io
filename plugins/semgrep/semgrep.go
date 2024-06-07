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

const (
	defaultCommand  = "scan"
	autoConfig      = "auto"
	metricsFlag     = "--metrics"
	metricsOffValue = "off"
)

// TODO: Wrap it in a custom error handler to add to the stack trace.
// Metadata of the plugin
var (
	Version       = "unknown"
	GolangVersion = "unknown"
	BuildTime     = "unknown"
)

// ScannerSemgrep represents the Semgrep scanner with its configuration and logger.
type ScannerSemgrep struct {
	logger       hclog.Logger
	globalConfig *config.Config
}

// newScannerSemgrep creates a new instance of ScannerSemgrep.
func newScannerSemgrep(logger hclog.Logger) *ScannerSemgrep {
	return &ScannerSemgrep{
		logger: logger,
	}
}

// setGlobalConfig sets the global configuration for the ScannerSemgrep instance.
func (g *ScannerSemgrep) setGlobalConfig(globalConfig *config.Config) {
	g.globalConfig = globalConfig
}

// buildCommandArgs constructs the command-line arguments for the Semgrep command.
func (g *ScannerSemgrep) buildCommandArgs(args shared.ScannerScanRequest) []string {
	var commandArgs []string

	appendArg := func(arg ...string) {
		commandArgs = append(commandArgs, arg...)
	}

	appendArg(defaultCommand)

	if len(args.AdditionalArgs) != 0 {
		appendArg(args.AdditionalArgs...)
	}

	if args.ReportFormat != "" {
		g.validateFormatSoft(args.ReportFormat)
		appendArg(fmt.Sprintf("--%v", args.ReportFormat))
	}

	configPath := args.ConfigPath
	if configPath == "" {
		configPath = getDefaultRuleSet(g.globalConfig)
	}
	appendArg("-f", configPath)

	if configPath != autoConfig {
		appendArg(metricsFlag, metricsOffValue)
	}

	appendArg("--output", args.ResultsPath)
	appendArg(args.TargetPath)

	return commandArgs
}

// Scan executes the Semgrep scan with the provided arguments and returns the scan response.
func (g *ScannerSemgrep) Scan(args shared.ScannerScanRequest) (shared.ScannerScanResponse, error) {
	var result shared.ScannerScanResponse
	g.logger.Info("scan is starting", "project", args.TargetPath)
	g.logger.Debug("debug info", "args", args)

	if err := g.validateScan(&args); err != nil {
		g.logger.Error("validation failed for scan operation", "error", err)
		return result, err
	}

	commandArgs := g.buildCommandArgs(args)

	cmd := exec.Command("semgrep", commandArgs...)
	g.logger.Debug("debug info", "cmd", cmd.Args)

	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}), &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	if err := cmd.Run(); err != nil {
		g.logger.Error("semgrep execution error", "error", err)
		return result, fmt.Errorf("semgrep execution error: %w. Output: %s", err, stdBuffer.String())
	}
	result.ResultsPath = args.ResultsPath
	g.logger.Info("scan finished", "project", args.TargetPath)
	g.logger.Info("result saved", "path", args.ResultsPath)
	g.logger.Debug("debug info", "project", args.TargetPath, "config", args.ConfigPath, "resultsFile", args.ResultsPath, "cmd", cmd.Args)
	return result, nil
}

// Setup initializes the global configuration for the ScannerSemgrep instance.
func (g *ScannerSemgrep) Setup(configData config.Config) (bool, error) {
	g.setGlobalConfig(&configData)
	return true, nil
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	semgrepInstance := newScannerSemgrep(logger)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			shared.PluginTypeScanner: &shared.ScannerPlugin{Impl: semgrepInstance},
		},
		Logger: logger,
	})
}
