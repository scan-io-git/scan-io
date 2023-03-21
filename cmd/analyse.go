package cmd

import (
	"fmt"
	"os"

	"github.com/scan-io-git/scan-io/internal/scanner"
	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/spf13/cobra"
)

type RunOptionsAnalyse struct {
	ScannerPluginName string
	InputFile         string
	ReportFormat      string
	Config            string
	AdditionalArgs    []string
	Threads           int
}

var allArgumentsAnalyse RunOptionsAnalyse

var analyseCmd = &cobra.Command{
	Use:          "analyse [flags] \n  scanio analyse [flags] [url]",
	SilenceUsage: true,
	Short:        "The main function is to present a top-level interface for a specified scanner.",

	RunE: func(cmd *cobra.Command, args []string) error {
		var reposInf []shared.RepositoryParams
		var path string

		checkArgs := func() error {
			if len(allArgumentsAnalyse.ScannerPluginName) == 0 {
				return fmt.Errorf("'scanner' flag must be specified!")
			}

			if len(args) == 1 {
				if len(allArgumentsAnalyse.InputFile) != 0 {
					return fmt.Errorf(("You can't use a specific url with an input-file argument!"))
				}
				path = args[0]
				if _, err := os.Stat(path); os.IsNotExist(err) {
					return fmt.Errorf("Path does not exists: %v", path)
				}

			} else {
				if len(allArgumentsAnalyse.InputFile) == 0 {
					return fmt.Errorf(("'input-file' flag must be specified!"))
				}

				reposData, err := utils.ReadReposFile2(allArgumentsAnalyse.InputFile)
				if err != nil {
					return fmt.Errorf("something happend when tool was parsing the Input File - %v", err)
				}
				reposInf = reposData
			}

			if len(allArgumentsAnalyse.AdditionalArgs) != 0 && allArgumentsAnalyse.ScannerPluginName != "semgrep" {
				return fmt.Errorf(("'args' is supported only for a semgrep plugin."))
			}

			return nil
		}

		if err := checkArgs(); err != nil {
			return err
		}

		logger := shared.NewLogger("core-analyze-scanner")
		s := scanner.New(allArgumentsAnalyse.ScannerPluginName, allArgumentsAnalyse.Threads, allArgumentsAnalyse.Config, allArgumentsAnalyse.ReportFormat, allArgumentsAnalyse.AdditionalArgs, logger)

		analyseArgs, err := s.PrepScanArgs(reposInf, path)
		if err != nil {
			return err
		}

		err = s.ScanRepos(analyseArgs)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(analyseCmd)

	analyseCmd.Flags().StringVar(&allArgumentsAnalyse.ScannerPluginName, "scanner", "", "the plugin name of the scanner used.. Eg. semgrep, bandit etc.")
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.InputFile, "input-file", "f", "", "a file in scanio format with a list of repositories to analyse. The list command could prepare this file..")
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.Config, "config", "c", "auto", "a path or type of config for a scanner. The value depends on a particular scanner's used formats.")
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.ReportFormat, "format", "o", "", "a format for a report with results.") //doesn't have default for "Uses ASCII output if no format specified"
	analyseCmd.Flags().StringSliceVar(&allArgumentsAnalyse.AdditionalArgs, "args", []string{}, "additional commands for semgrep which will be added to a semgrep call. The format in quotes with commas without spaces.")
	analyseCmd.Flags().IntVarP(&allArgumentsAnalyse.Threads, "threads", "j", 1, "number of concurrent goroutines.")
}
