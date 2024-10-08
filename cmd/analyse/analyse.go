package analyse

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/scanner"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// RunOptionsAnalyse holds the arguments for the analyse command.
type RunOptionsAnalyse struct {
	Scanner        string   `json:"scanner,omitempty"`
	InputFile      string   `json:"input_file,omitempty"`
	ReportFormat   string   `json:"report_format,omitempty"`
	ScannerConfig  string   `json:"scanner_config,omitempty"`
	AdditionalArgs []string `json:"additional_args,omitempty"`
	OutputPath     string   `json:"output_path,omitempty"`
	Threads        int      `json:"threads"`
}

// Global variables for configuration and command arguments
var (
	AppConfig           *config.Config
	analyseOptions      RunOptionsAnalyse
	exampleAnalyseUsage = `  # Running semgrep scanner with an input file
  scanio analyse --scanner semgrep --input-file /path/to/list_output.file
	
  # Running semgrep scanner on a specific path
  scanio analyse --scanner semgrep /path/to/my_project
	
  # Running semgrep scanner on a specific path with a specified report format
  scanio analyse --scanner semgrep --format sarif /path/to/my_project
	
  # Running semgrep scanner with a configuration file and an input file with multiple concurrent threads 
  scanio analyse --scanner semgrep --config /path/to/scanner-config --input-file /path/to/list_output.file -j 2
	
  # Running semgrep scanner with additional arguments
  scanio analyse --scanner semgrep --input-file /path/to/list_output.file --format sarif -- --verbose --severity INFO

  # Running semgrep scanner with an input file and specifying the output directory
  scanio analyse --scanner semgrep --input-file /path/to/list_output.file --output /path/to/scanner_results

  # Running semgrep scanner on a specific path and specifying the output file
  scanio analyse --scanner semgrep /path/to/my_project --format json --output /path/to/scanner_results/result.json`
)

// AnalyseCmd represents the analyse command.
var AnalyseCmd = &cobra.Command{
	Use:                   "analyse --scanner/-p PLUGIN_NAME [--config/-c PATH] [--format/-f OUTPUT_FORMAT] [-j THREADS_NUMBER, default=1] {--input-file/-i PATH | PATH} -- [args...]",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example:               exampleAnalyseUsage,
	Short:                 "Provides a top-level interface with orchestration for running a specified scanner",
	RunE:                  runAnalyseCommand,
}

// Init initializes the global configuration variable.
func Init(cfg *config.Config) {
	AppConfig = cfg
	AnalyseCmd.Long = generateLongDescription(AppConfig)
}

// runAnalyseCommand executes the analyse command.
func runAnalyseCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 && !shared.HasFlags(cmd.Flags()) {
		return cmd.Help()
	}

	logger := logger.NewLogger(AppConfig, "core-analyze")
	argsLenAtDash := cmd.ArgsLenAtDash()

	if err := validateAnalyseArgs(&analyseOptions, args, argsLenAtDash); err != nil {
		logger.Error("invalid analyse arguments", "error", err)
		return errors.NewCommandError(analyseOptions, nil, fmt.Errorf("invalid analyse arguments: %w", err), 1)
	}

	mode := determineMode(args, argsLenAtDash)

	s := scanner.New(
		analyseOptions.Scanner,
		analyseOptions.ScannerConfig,
		analyseOptions.ReportFormat,
		analyseOptions.AdditionalArgs,
		analyseOptions.Threads,
		logger,
	)

	reposInf, targetPath, err := prepareScanTargets(&analyseOptions, args, mode)
	if err != nil {
		logger.Error("failed to prepare scan targets", "error", err)
		return errors.NewCommandError(analyseOptions, nil, fmt.Errorf("failed to prepare scan targets: %w", err), 1)
	}

	analyseArgs, err := s.PrepareScanArgs(AppConfig, reposInf, targetPath, analyseOptions.OutputPath)
	if err != nil {
		logger.Error("failed to prepare scan requests", "error", err)
		return errors.NewCommandError(analyseOptions, nil, fmt.Errorf("failed to prepare scan arguments: %w", err), 1)
	}

	analyseResult, scanErr := s.ScanRepos(AppConfig, analyseArgs)

	// TODO: use a logger system to write it in a persistent log
	metaDataFileName := fmt.Sprintf("ANALYSE_%s", strings.ToUpper(s.PluginName))
	if config.IsCI(AppConfig) {
		startTime := time.Now().UTC().Format(time.RFC3339)
		metaDataFileName = fmt.Sprintf("ANALYSE_%s_%v", strings.ToUpper(s.PluginName), startTime)
	}

	if err := shared.WriteGenericResult(AppConfig, logger, analyseResult, metaDataFileName); err != nil {
		logger.Error("failed to write result", "error", err)
	}

	if scanErr != nil {
		return errors.NewCommandErrorWithResult(analyseResult, fmt.Errorf("analyse command failed: %w", scanErr), 2)
	}

	logger.Info("analyse command completed successfully")
	logger.Debug("analyse result", "result", analyseResult)
	if config.IsCI(AppConfig) {
		shared.PrintResultAsJSON(logger, analyseResult)
	}
	return nil
}

// generateLongDescription generates the long description dynamically with the list of available scanner plugins.
func generateLongDescription(cfg *config.Config) string {
	pluginsMeta := shared.GetPluginVersions(config.GetScanioPluginsHome(cfg), "scanner")
	var plugins []string
	for plugin := range pluginsMeta {
		plugins = append(plugins, plugin)
	}
	return fmt.Sprintf(`Provides a top-level interface with orchestration for running a specified scanner.

List of available scanner plugins:
  %s`, strings.Join(plugins, "\n  "))
}

// Initialize flags for the analyse command.
func init() {
	AnalyseCmd.Flags().StringVarP(&analyseOptions.ScannerConfig, "config", "c", "", "Path or type of configuration for the scanner. The format depends on the specific scanner being used.")
	AnalyseCmd.Flags().StringVarP(&analyseOptions.ReportFormat, "format", "f", "", "Format for the report with results.")
	AnalyseCmd.Flags().StringVarP(&analyseOptions.InputFile, "input-file", "i", "", "Path to a file in Scanio format containing a list of repositories to analyse. Use the list command to prepare this file.")
	AnalyseCmd.Flags().StringVarP(&analyseOptions.OutputPath, "output", "o", "", "Path to the output file or directory where the scanner's results will be saved.")
	AnalyseCmd.Flags().StringVarP(&analyseOptions.Scanner, "scanner", "p", "", "Name of the scanner plugin to use (e.g., semgrep, bandit).")
	AnalyseCmd.Flags().IntVarP(&analyseOptions.Threads, "threads", "j", 1, "Number of concurrent threads to use.")
	AnalyseCmd.Flags().BoolP("help", "h", false, "Show help for the analyse command.")
}
