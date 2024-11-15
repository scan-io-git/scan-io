package main

import (
	"bytes"
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

	plugin_internal "github.com/scan-io-git/scan-io/plugins/trufflehog3/internal"
)

// TODO: Wrap it in a custom error handler to add to the stack trace.
// Metadata of the plugin
var (
	Version       = "unknown"
	GolangVersion = "unknown"
	BuildTime     = "unknown"
)

// ScannerTrufflehog3 represents the Trufflehog3 scanner with its configuration and logger.
type ScannerTrufflehog3 struct {
	logger       hclog.Logger
	globalConfig *config.Config
}

// newScannerScannerTrufflehog3 creates a new instance of ScannerTrufflehog3.
func newScannerTrufflehog3(logger hclog.Logger) *ScannerTrufflehog3 {
	return &ScannerTrufflehog3{
		logger: logger,
	}
}

// setGlobalConfig sets the global configuration for the ScannerTrufflehog3 instance.
func (g *ScannerTrufflehog3) setGlobalConfig(globalConfig *config.Config) {
	g.globalConfig = globalConfig
}

// handleGlobalConfig processes the global configuration for the Trufflehog3 scanner.
func (g *ScannerTrufflehog3) handleGlobalConfig(args shared.ScannerScanRequest) error {
	if err := plugin_internal.HandleScannerConfig(
		g.logger,
		g.globalConfig.Trufflehog3Plugin.ExcludePaths,
		args.TargetPath,
		g.globalConfig.Trufflehog3Plugin.WriteDefaultConfig,
		g.globalConfig.Trufflehog3Plugin.OverwriteConfig,
	); err != nil {
		return fmt.Errorf("failed to process configuration for Trufflehog3 plugin: %w", err)
	}
	return nil
}

// buildCommandArgs constructs the command-line arguments for the Trufflehog3 command.
func (g *ScannerTrufflehog3) buildCommandArgs(args shared.ScannerScanRequest, reportFormat string) []string {
	commandArgs := append([]string{}, args.AdditionalArgs...)

	// Trufflehog3 --rules is a rules file that contains regexes that might trigger
	// --config it's a flag for .trufflehog3.yml file with wider configuration rather than rules
	// .trufflehog3.yml will be found automatically in root of your folder
	if args.ConfigPath != "" {
		commandArgs = append(commandArgs, "--rules", args.ConfigPath)
	}

	if reportFormat != "" {
		commandArgs = append(commandArgs, "--format", reportFormat)
	}

	// Here we added -z flag because Trufflehog3 sends a not correct exit code even when it finished without errors
	// TODO: should we move -z to the scanio config file
	commandArgs = append(commandArgs, "-z", "--output", args.ResultsPath, args.TargetPath)

	return commandArgs
}

// convertReportFormat handles the conversion of the scan result to the desired format.
func (g *ScannerTrufflehog3) convertReportFormat(originalFormat, resultsPath string, updatedResultsPath *string) error {
	var err error

	switch originalFormat {
	case "sarif":
		*updatedResultsPath, err = plugin_internal.JsonToSarifReport(resultsPath)
	case "markdown":
		*updatedResultsPath, err = plugin_internal.JsonToPlainReport(resultsPath)
	default:
		err = fmt.Errorf("unsupported format: %s", originalFormat)
	}
	return err
}

// Scan executes the Trufflehog3 scan with the provided arguments and returns the scan response.
func (g *ScannerTrufflehog3) Scan(args shared.ScannerScanRequest) (shared.ScannerScanResponse, error) {
	var (
		cmd          *exec.Cmd
		stdBuffer    bytes.Buffer
		result       shared.ScannerScanResponse
		reportFormat string
	)

	g.logger.Info("Scan is starting", "project", args.TargetPath)
	g.logger.Debug("Scan arguments", "args", args)

	if err := g.handleGlobalConfig(args); err != nil {
		g.logger.Error("Failed to handle global configuration", "error", err)
		return result, fmt.Errorf("failed to handle global configuration: %w", err)
	}

	originalFormat, reportFormat, needsConversion := g.CheckReportFormat(&args)
	if reportFormat != "" {
		args.ResultsPath = strings.TrimSuffix(args.ResultsPath, filepath.Ext(args.ResultsPath)) + "." + reportFormat
	}
	commandArgs := g.buildCommandArgs(args, reportFormat)

	cmd = exec.Command("trufflehog3", commandArgs...)
	g.logger.Debug("Executing command", "cmd", cmd.Args)

	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}), &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	if err := cmd.Run(); err != nil {
		g.logger.Error("trufflehog3 execution error", "error", err, "output", stdBuffer.String())
		return result, fmt.Errorf("trufflehog3 execution failed: %w", err)
	}

	result.ResultsPath = args.ResultsPath
	g.logger.Info("Scan finished for", "project", args.TargetPath, "resultsPath", result.ResultsPath)

	if needsConversion {
		g.logger.Warn("Converting report", "originalFormat", originalFormat)
		if err := g.convertReportFormat(originalFormat, args.ResultsPath, &result.ResultsPath); err != nil {
			g.logger.Error("Error during report conversion", "error", err)
			return result, fmt.Errorf("report conversion failed: %w", err)
		}
		g.logger.Info("Report conversion finished", "newFormat", originalFormat, "convertedPath", result.ResultsPath)
	}

	g.logger.Info("Result is saved to", "path to a result file", result.ResultsPath)
	g.logger.Debug("Debug info", "project", args.TargetPath, "config", args.ConfigPath, "resultsFile", result.ResultsPath, "cmd", cmd.Args)
	return result, nil
}

// Setup initializes the global configuration for the ScannerTrufflehog3 instance.
func (g *ScannerTrufflehog3) Setup(configData config.Config) (bool, error) {
	g.setGlobalConfig(&configData)
	return true, nil
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	trufflehog3Instance := newScannerTrufflehog3(logger)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			shared.PluginTypeScanner: &shared.ScannerPlugin{Impl: trufflehog3Instance},
		},
		Logger: logger,
	})
}
