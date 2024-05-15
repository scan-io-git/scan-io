package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	utils "github.com/scan-io-git/scan-io/internal/utils"

	"github.com/scan-io-git/scan-io/internal/logger"
	"github.com/scan-io-git/scan-io/internal/scanner"
	"github.com/scan-io-git/scan-io/pkg/shared"
)

type RunOptionsAnalyse struct {
	ScannerPluginName string
	InputFile         string
	ReportFormat      string
	Config            string
	AdditionalArgs    []string
	OutputPrefix      string
	Threads           int
}

var (
	allArgumentsAnalyse RunOptionsAnalyse
	resultAnalyse       shared.GenericLaunchesResult
	execExampleAnalyse  = `  # Analysing using semgrep with an input file argument
  scanio analyse --scanner semgrep --input-file /Users/root/.scanio/output.file --format sarif -j 1
  
  # Analysing using semgrep with a specific path
  scanio analyse --scanner semgrep --format sarif -j 1 /tmp/my_project

  # Analysing using semgrep with an input file and custom rules
  scanio analyse --scanner semgrep --config /Users/root/scan-io-semgrep-rules --input-file /Users/root/.scanio/output.file --format sarif -j 1

  # Analysing using semgrep with an input file and additional arguments
    # If you want to execute scanner with custom arguments,
    # you could use two dashes (--) to separate additional flags/arguments
  scanio analyse --scanner semgrep --input-file /Users/root/.scanio/output.file --format sarif -j 1 -- --verbose --severity INFO`
)

var analyseCmd = &cobra.Command{
	Use:                   "analyse --scanner PLUGIN_NAME [--config /local_path] [--format/-f FILE_FORMAT] [-j THREADS_NUMBER] (--input-file/-i /local_path/repositories.file | /local_path) -- [args...]",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example:               execExampleAnalyse,
	Short:                 "The main function is to present a top-level interface for a specified scanner",
	Long: `The main function is to present a top-level interface for a specified scanner

List of plugins:
  - semgrep
  - bandit
  - trufflehog
  - trufflehog3`,

	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			reposInf     []shared.RepositoryParams
			path         string
			outputBuffer bytes.Buffer // Decision maker MVP needs
		)

		logger := logger.NewLogger(AppConfig, "core-analyze")
		argsLenAtDash := cmd.ArgsLenAtDash()
		checkArgs := func() error {
			if len(allArgumentsAnalyse.ScannerPluginName) == 0 {
				return fmt.Errorf("A 'scanner' flag must be specified!")
			}

			if argsLenAtDash > -1 {
				allArgumentsAnalyse.AdditionalArgs = args[argsLenAtDash:]
			}
			if ((len(args) == 0) || (len(args) > 0 && argsLenAtDash == 0)) && len(allArgumentsAnalyse.InputFile) == 0 {
				return fmt.Errorf(("An 'input-file' flag or a path must be specified!"))
			}

			if len(args) > 0 && (argsLenAtDash == -1 || argsLenAtDash == 1) {
				if len(allArgumentsAnalyse.InputFile) != 0 {
					return fmt.Errorf(("You can't use an 'input-file' flag and a path at the same time!"))
				}

				path = args[0]
				if _, err := os.Stat(path); os.IsNotExist(err) {
					return fmt.Errorf("The path does not exists: %v", path)
				}

			} else {
				if len(allArgumentsAnalyse.InputFile) == 0 {
					return fmt.Errorf(("An 'input-file' flag must be specified!"))
				}

				reposData, err := utils.ReadReposFile2(allArgumentsAnalyse.InputFile)
				if err != nil {
					return fmt.Errorf("Something happend when tool was parsing the Input File - %v", err)
				}
				reposInf = reposData
			}

			return nil
		}

		if err := checkArgs(); err != nil {
			return err
		}

		s := scanner.New(allArgumentsAnalyse.ScannerPluginName, allArgumentsAnalyse.Threads, allArgumentsAnalyse.Config, allArgumentsAnalyse.ReportFormat, allArgumentsAnalyse.AdditionalArgs, logger)

		analyseArgs, err := s.PrepScanArgs(reposInf, path, allArgumentsAnalyse.OutputPrefix)
		if err != nil {
			return err
		}

		resultAnalyse = s.ScanRepos(AppConfig, analyseArgs)
		resultJSON, err := json.Marshal(resultAnalyse)
		outputBuffer.Write(resultJSON)
		if err != nil {
			logger.Error("Error", "message", err)
			return err
		}

		// Decision maker MVP needs
		shared.ResultBufferMutex.Lock()
		shared.ResultBuffer = outputBuffer
		shared.ResultBufferMutex.Unlock()
		outputBuffer.Write(resultJSON)

		shared.WriteJsonFile(fmt.Sprintf("%v/ANALYSE.scanio-result", shared.GetScanioHome()), logger, resultAnalyse)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(analyseCmd)

	analyseCmd.Flags().StringVar(&allArgumentsAnalyse.ScannerPluginName, "scanner", "", "the plugin name of the scanner used.. Eg. semgrep, bandit etc.")
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.InputFile, "input-file", "i", "", "a file in Scanio format with a list of repositories to analyse. The list command could prepare this file.")
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.Config, "config", "c", "", "a path or type of config for a scanner. The value depends on a particular scanner's used formats.")
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.ReportFormat, "format", "f", "", "a format for a report with results.") //doesn't have default for "Uses ASCII output if no format specified"
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.OutputPrefix, "output", "o", "", "a path for scanner's output. The path will be used like a prefix for a scanner's report.")
	analyseCmd.Flags().IntVarP(&allArgumentsAnalyse.Threads, "threads", "j", 1, "number of concurrent goroutines.")
}
