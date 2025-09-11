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

const PluginName = "codeql"

// TODO: Wrap it in a custom error handler to add to the stack trace.
// Metadata of the plugin
var (
	Version       = "unknown"
	GolangVersion = "unknown"
	BuildTime     = "unknown"
)

// ScannerCodeQL represents the CodeQL scanner with its configuration and logger.
type ScannerCodeQL struct {
	logger       hclog.Logger
	globalConfig *config.Config
	name         string
}

// newScannerCodeQL creates a new instance of ScannerCodeQL.
func newScannerCodeQL(logger hclog.Logger) *ScannerCodeQL {
	return &ScannerCodeQL{
		logger: logger,
		name:   PluginName,
	}
}

// setGlobalConfig sets the global configuration for the ScannerCodeQL instance.
func (g *ScannerCodeQL) setGlobalConfig(globalConfig *config.Config) {
	g.globalConfig = globalConfig
}

// executeCommand runs the specified command and captures its output.
func (g *ScannerCodeQL) executeCommand(cmd *exec.Cmd) error {
	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true}), &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	if err := cmd.Run(); err != nil {
		g.logger.Error(fmt.Sprintf("%q execution error", cmd.Path), "error", err)
		return fmt.Errorf("%q execution error: %w. Output: %s", cmd.Path, err, stdBuffer.String())
	}
	return nil
}

// createDatabase creates a CodeQL database for the given project.
func (g *ScannerCodeQL) createDatabase(databaseDir string, args shared.ScannerScanRequest) error {
	g.logger.Debug("Creating CodeQL database", "project", args.TargetPath)
	language := g.globalConfig.CodeQLPlugin.DBLanguage
	if err := validateLanguageHard(language); err != nil {
		return err
	}

	commandArgs := []string{"database", "create", databaseDir, "--language", language, "--source-root", args.TargetPath}
	cmd := exec.Command("codeql", commandArgs...)
	return g.executeCommand(cmd)
}

// analyzeDatabase analyzes the CodeQL database and generates a report.
func (g *ScannerCodeQL) analyzeDatabase(databaseDir string, args shared.ScannerScanRequest) error {
	g.logger.Debug("Analyzing CodeQL database", "project", args.TargetPath)

	commandArgs := []string{"database", "analyze", databaseDir}
	if args.ReportFormat != "" {
		g.validateFormatSoft(args.ReportFormat)
		commandArgs = append(commandArgs, "--format", args.ReportFormat)
	}
	commandArgs = append(commandArgs, args.ConfigPath, "--output", args.ResultsPath)

	if len(args.AdditionalArgs) != 0 {
		commandArgs = append(commandArgs, args.AdditionalArgs...)
	}

	cmd := exec.Command("codeql", commandArgs...)
	return g.executeCommand(cmd)
}

// Scan executes the CodeQL scan with the provided arguments and returns the scan response.
func (g *ScannerCodeQL) Scan(args shared.ScannerScanRequest) (shared.ScannerScanResponse, error) {
	var result shared.ScannerScanResponse
	g.logger.Info("codeQL scan starting", "project", args.TargetPath)
	g.logger.Debug("debug info", "args", args)

	if err := g.validateScan(&args); err != nil {
		g.logger.Error("validation failed for scan operation", "error", err)
		return result, err
	}

	scanioTmp := config.GetScanioTempHome(g.globalConfig)
	databaseDir, err := os.MkdirTemp(scanioTmp, "codeql_db")
	if err != nil {
		return result, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(databaseDir)

	if err = g.createDatabase(databaseDir, args); err != nil {
		return result, err
	}

	if err = g.analyzeDatabase(databaseDir, args); err != nil {
		return result, err
	}

	result.ResultsPath = args.ResultsPath
	g.logger.Info("scan finished", "project", args.TargetPath)
	g.logger.Info("result saved", "path", args.ResultsPath)
	g.logger.Debug("debug info", "project", args.TargetPath, "config", args.ConfigPath, "resultsFile", args.ResultsPath)
	return result, nil
}

// Setup initializes the global configuration for the ScannerCodeQL instance.
func (g *ScannerCodeQL) Setup(configData config.Config) (bool, error) {
	g.setGlobalConfig(&configData)
	return true, nil
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	codeQLInstance := newScannerCodeQL(logger)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			shared.PluginTypeScanner: &shared.ScannerPlugin{Impl: codeQLInstance},
		},
		Logger: logger,
	})
}
