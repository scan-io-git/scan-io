package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"

	plugin_internal "github.com/scan-io-git/scan-io/plugins/betterleaks/internal"
)

const (
	PluginName = "betterleaks"
	// findingsExitCode makes "findings present" distinguishable from errors,
	// which betterleaks reports with exit code 1.
	findingsExitCode = "2"
)

// Metadata of the plugin
var (
	Version       = "unknown"
	GolangVersion = "unknown"
	BuildTime     = "unknown"
)

// ScannerBetterleaks runs the betterleaks CLI and converts its report.
type ScannerBetterleaks struct {
	logger       hclog.Logger
	globalConfig *config.Config
	name         string
}

// newScannerBetterleaks creates a new instance of ScannerBetterleaks.
func newScannerBetterleaks(logger hclog.Logger) *ScannerBetterleaks {
	return &ScannerBetterleaks{
		logger: logger,
		name:   PluginName,
	}
}

// setGlobalConfig sets the global configuration for the ScannerBetterleaks instance.
func (g *ScannerBetterleaks) setGlobalConfig(globalConfig *config.Config) {
	g.globalConfig = globalConfig
}

// buildCommandArgs builds the betterleaks command line. AdditionalArgs pass
// through unchanged; policy-specific flags are the caller's choice.
func buildCommandArgs(args shared.ScannerScanRequest, jsonPath string) []string {
	commandArgs := []string{resolveCommand(args.Command)}
	commandArgs = append(commandArgs, args.AdditionalArgs...)
	if args.ConfigPath != "" {
		commandArgs = append(commandArgs, "-c", args.ConfigPath)
	}
	return append(commandArgs, "--exit-code", findingsExitCode, "-o", jsonPath, args.TargetPath)
}

// exitCodeOf extracts the process exit code. A failure to start the process
// is returned as an error.
func exitCodeOf(runErr error) (int, error) {
	if runErr == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return 0, runErr
}

// Scan executes the betterleaks scan with the provided arguments and returns the scan response.
func (g *ScannerBetterleaks) Scan(args shared.ScannerScanRequest) (shared.ScannerScanResponse, error) {
	var result shared.ScannerScanResponse

	g.logger.Info("Scan is starting", "project", args.TargetPath)

	if err := g.validateScan(&args); err != nil {
		g.logger.Error("validation failed for scan operation", "error", err)
		return result, err
	}

	originalFormat, _, needsConversion := g.CheckReportFormat(&args)
	jsonPath := strings.TrimSuffix(args.ResultsPath, filepath.Ext(args.ResultsPath)) + ".json"
	commandArgs := buildCommandArgs(args, jsonPath)

	// A leftover report from an earlier run must not be mistaken for this run's output.
	if err := os.Remove(jsonPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, fmt.Errorf("failed to remove stale report %q: %w", jsonPath, err)
	}

	cmd := exec.Command("betterleaks", commandArgs...)
	g.logger.Debug("Executing command", "cmd", cmd.Args)

	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}), &stdBuffer)
	cmd.Stdout = mw
	cmd.Stderr = mw

	exitCode, err := exitCodeOf(cmd.Run())
	if err != nil {
		g.logger.Error("betterleaks execution error", "error", err)
		return result, fmt.Errorf("betterleaks execution failed: %w. Output: %s", err, stdBuffer.String())
	}

	report, err := plugin_internal.LoadCompleteReport(exitCode, jsonPath)
	if err != nil {
		g.logger.Error("betterleaks scan failed", "error", err, "output", stdBuffer.String())
		return result, fmt.Errorf("betterleaks scan failed: %w. Output: %s", err, stdBuffer.String())
	}

	result.ResultsPath = jsonPath
	if needsConversion {
		g.logger.Warn("Converting report", "originalFormat", originalFormat)
		sarifPath, err := plugin_internal.WriteSarif(report, jsonPath, args.TargetPath)
		if err != nil {
			g.logger.Error("Error during report conversion", "error", err)
			return result, fmt.Errorf("report conversion failed: %w", err)
		}
		result.ResultsPath = sarifPath
	}

	g.logger.Info("Result is saved to", "path to a result file", result.ResultsPath, "findings", len(report.Findings))
	return result, nil
}

// Setup initializes the global configuration for the ScannerBetterleaks instance.
func (g *ScannerBetterleaks) Setup(configData config.Config) (bool, error) {
	g.setGlobalConfig(&configData)
	return true, nil
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	instance := newScannerBetterleaks(logger)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			shared.PluginTypeScanner: &shared.ScannerPlugin{Impl: instance},
		},
		Logger: logger,
	})
}
