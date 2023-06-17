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

type ScannerCodeQL struct {
	logger hclog.Logger
}

var CODEQL_SUPPORTED_LANGUAGES = []string{"cpp", "csharp", "go", "java", "javascript", "python", "ruby", "swift"}

func isLanguageSupported(language string) bool {
	for _, l := range CODEQL_SUPPORTED_LANGUAGES {
		if l == language {
			return true
		}
	}
	return false
}

func (g *ScannerCodeQL) createDatabase(databaseDir string, args shared.ScannerScanRequest) error {
	// codeql database create /tmp/scanio.codeqldb --language go

	var stdBuffer bytes.Buffer

	g.logger.Debug("Creating CodeQL database", "project", args.RepoPath)

	language := os.Getenv("SCANIO_CODEQL_LANGUAGE")
	if !isLanguageSupported(language) {
		return fmt.Errorf("unsupported language for CodeQL")
	}

	commandArgs := []string{"database", "create", databaseDir, "--language", language, "--source-root", args.RepoPath}

	cmd := exec.Command("codeql", commandArgs...)
	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}), &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	err := cmd.Run()
	if err != nil {
		err := fmt.Errorf(stdBuffer.String())
		g.logger.Error("codeql execution error", "error", err)
		return err
	}

	return nil
}

func (g *ScannerCodeQL) analyzeDatabase(databaseDir string, args shared.ScannerScanRequest) error {
	g.logger.Debug("Analyzing CodeQL database", "project", args.RepoPath)

	// codeql database analyze /tmp/scanio.codeqldb/ --format sarifv2.1.0 codeql/go-queries -o /tmp/scanio.sarif

	// query := os.Getenv("SCANIO_CODEQL_QUERY")

	var stdBuffer bytes.Buffer

	commandArgs := []string{"database", "analyze", databaseDir, "--format", args.ReportFormat, args.ConfigPath, "--output", args.ResultsPath}

	cmd := exec.Command("codeql", commandArgs...)
	mw := io.MultiWriter(g.logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	}), &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	err := cmd.Run()
	if err != nil {
		err := fmt.Errorf(stdBuffer.String())
		g.logger.Error("codeql execution error", "error", err)
		return err
	}

	return nil
}

func (g *ScannerCodeQL) Scan(args shared.ScannerScanRequest) error {

	g.logger.Info("CodeQL flow starting", "project", args.RepoPath)
	g.logger.Debug("Debug info", "args", args)

	databaseDir, err := os.MkdirTemp("", "codeqdb")
	if err != nil {
		return err
	}
	defer os.RemoveAll(databaseDir)

	if err = g.createDatabase(databaseDir, args); err != nil {
		return err
	}

	if err = g.analyzeDatabase(databaseDir, args); err != nil {
		return err
	}

	g.logger.Info("Scan finished for", "project", args.RepoPath)
	g.logger.Info("Result is saved to", "path to a result file", args.ResultsPath)
	g.logger.Debug("Debug info", "project", args.RepoPath, "config", args.ConfigPath, "resultsFile", args.ResultsPath)
	return nil
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	Scanner := &ScannerCodeQL{
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
