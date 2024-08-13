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

	plugin_internal "github.com/scan-io-git/scan-io/plugins/trufflehog3/internal"
)

// TODO: Wrap it in a custom error handler to add to the stack trace.
// Metadata of the plugin
var (
	Version       = "unknown"
	GolangVersion = "unknown"
	BuildTime     = "unknown"
)

// setGlobalConfig sets the global configuration for the ScannerTrufflehog3 instance.
func (g *ScannerTrufflehog3) setGlobalConfig(globalConfig *config.Config) {
	g.globalConfig = globalConfig
}

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

// handleGlobalConfig processes the global configuration for the Trufflehog3 scanner.
func (g *ScannerTrufflehog3) handleGlobalConfig(args shared.ScannerScanRequest) error {
	err := plugin_internal.HandleScannerConfig(
		g.logger,
		g.globalConfig.Trufflehog3Plugin.ExcludePaths,
		args.TargetPath,
		g.globalConfig.Trufflehog3Plugin.WriteDefaultConfig,
		g.globalConfig.Trufflehog3Plugin.OverwriteConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to process configuration for Trufflehog3 plugin: %w", err)
	}
	return nil
}

// Scan executes the Trufflehog3 scan with the provided arguments and returns the scan response.
func (g *ScannerTrufflehog3) Scan(args shared.ScannerScanRequest) (shared.ScannerScanResponse, error) {
	var (
		commandArgs []string
		cmd         *exec.Cmd
		stdBuffer   bytes.Buffer
		result      shared.ScannerScanResponse
	)

	g.logger.Info("Scan is starting", "project", args.TargetPath)
	g.logger.Debug("Debug info", "args", args)

	g.handleGlobalConfig(args)

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

	// Here we added -z flag because Trufflehog3 sends a not correct exit code even when it finished without errors
	// TODO: should we move -z to the scanio config file
	commandArgs = append(commandArgs, "-z", "--output", args.ResultsPath, args.TargetPath)

	cmd = exec.Command("trufflehog3", commandArgs...)
	g.logger.Debug("Debug info", "cmd", cmd.Args)

	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}), &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	err := cmd.Run()
	if err != nil {
		err := fmt.Errorf(stdBuffer.String())
		g.logger.Error("Trufflehog3 execution error", "error", err)
		return result, err
	}

	result.ResultsPath = args.ResultsPath
	g.logger.Info("Scan finished for", "project", args.TargetPath)
	g.logger.Info("Result is saved to", "path to a result file", args.ResultsPath)
	g.logger.Debug("Debug info", "project", args.TargetPath, "config", args.ConfigPath, "resultsFile", args.ResultsPath, "cmd", cmd.Args)
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
