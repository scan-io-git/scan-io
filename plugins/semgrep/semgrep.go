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

type ScannerSemgrep struct {
	logger       hclog.Logger
	globalConfig *config.Config
}

const SEMGREP_RULES_FOLDER = "/scanio-rules/semgrep"

func (g *ScannerSemgrep) getDefaultConfig() string {
	if config.IsCI(g.globalConfig) {
		if _, err := os.Stat(SEMGREP_RULES_FOLDER); !os.IsNotExist(err) {
			return SEMGREP_RULES_FOLDER
		}
		return "p/ci"
	}

	return "p/default"
}

func (g *ScannerSemgrep) Scan(args shared.ScannerScanRequest) (shared.ScannerScanResponse, error) {
	var result shared.ScannerScanResponse
	g.logger.Info("Scan is starting", "project", args.RepoPath)
	g.logger.Debug("Debug info", "args", args)

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
		commandArgs = append(commandArgs, reportFormat)
	}

	// use "p/deafult" by default to not send metrics
	configPath := args.ConfigPath
	if args.ConfigPath == "" {
		configPath = g.getDefaultConfig()
	}

	// auto config requires sendings metrics
	commandArgs = append(commandArgs, "-f", configPath)
	if configPath != "auto" {
		commandArgs = append(commandArgs, "--metrics", "off")
	}

	// output file
	commandArgs = append(commandArgs, "--output", args.ResultsPath)

	// repo path
	commandArgs = append(commandArgs, args.RepoPath)

	// prep cmd
	cmd = exec.Command("semgrep", commandArgs...)
	g.logger.Debug("Debug info", "cmd", cmd.Args)

	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}), &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	err := cmd.Run()
	if err != nil {
		err := fmt.Errorf(stdBuffer.String())
		g.logger.Error("Semgrep execution error", "error", err)
		return result, err
	}

	result.ResultsPath = args.ResultsPath
	g.logger.Info("Scan finished for", "project", args.RepoPath)
	g.logger.Info("Result is saved to", "path to a result file", args.ResultsPath)
	g.logger.Debug("Debug info", "project", args.RepoPath, "config", args.ConfigPath, "resultsFile", args.ResultsPath, "cmd", cmd.Args)
	return result, nil
}

func (g *ScannerSemgrep) Setup(configData config.Config) (bool, error) {
	g.globalConfig = &configData
	return true, nil
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
