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
var (
	Version       = "unknown"
	GolangVersion = "unknown"
	BuildTime     = "unknown"
)

type ScannerTrufflehog3 struct {
	logger       hclog.Logger
	globalConfig *config.Config
}

func (g *ScannerTrufflehog3) Scan(args shared.ScannerScanRequest) (shared.ScannerScanResponse, error) {
	var (
		commandArgs []string
		cmd         *exec.Cmd
		stdBuffer   bytes.Buffer
		result      shared.ScannerScanResponse
	)

	g.logger.Info("Scan is starting", "project", args.TargetPath)
	g.logger.Debug("Debug info", "args", args)

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

func (g *ScannerTrufflehog3) Setup(configData config.Config) (bool, error) {
	g.globalConfig = &configData
	return true, nil
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
