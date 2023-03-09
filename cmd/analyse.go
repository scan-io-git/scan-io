package cmd

import (
	"fmt"

	"github.com/scan-io-git/scan-io/internal/scanner"
	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/spf13/cobra"
)

type RunOptionsAnalyse struct {
	ScannerPluginName string
	// Repositories      []string
	InputFile      string
	ReportFormat   string
	Config         string
	AdditionalArgs []string
	Threads        int
}

var allArgumentsAnalyse RunOptionsAnalyse

var analyseCmd = &cobra.Command{
	Use:          "analyse",
	SilenceUsage: true,
	Short:        "The main function is to present a top-level interface for a specified scanner.",

	RunE: func(cmd *cobra.Command, args []string) error {
		checkArgs := func() error {
			if len(allArgumentsAnalyse.ScannerPluginName) == 0 {
				return fmt.Errorf("'scanner' flag must be specified!")
			}

			// if len(allArgumentsAnalyse.Repositories) != 0 && allArgumentsAnalyse.InputFile != "" {
			// 	return fmt.Errorf("you can't use both input types for repositories")
			// }

			// if len(allArgumentsAnalyse.Repositories) == 0 && len(allArgumentsAnalyse.InputFile) == 0 {
			// 	return fmt.Errorf("'repos' or 'input-file' flag must be specified")
			// }
			// fmt.Println(allArgumentsAnalyse.InputFile)
			if allArgumentsAnalyse.InputFile == "" {
				return fmt.Errorf("'input-file' flag must be specified!")
			}

			return nil
		}

		if err := checkArgs(); err != nil {
			return err
		}

		reposInf, err := utils.ReadReposFile2(allArgumentsAnalyse.InputFile)
		if err != nil {
			return fmt.Errorf("something happend when tool was parsing the Input File - %v", err)
		}

		if len(allArgumentsAnalyse.AdditionalArgs) != 0 && allArgumentsAnalyse.ScannerPluginName != "semgrep" {
			return fmt.Errorf(("'args' is supported only for a semgrep plugin."))
		}

		logger := shared.NewLogger("core-analyze-scanner")
		s := scanner.New(allArgumentsAnalyse.ScannerPluginName, allArgumentsAnalyse.Threads, allArgumentsAnalyse.Config, allArgumentsAnalyse.ReportFormat, allArgumentsAnalyse.AdditionalArgs, logger)

		analyseArgs, err := s.PrepScanArgs(reposInf)
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
	// analyseCmd.Flags().StringSliceVar(&allArgumentsAnalyse.Repositories, "repos", []string{}, "list of repos to analyse - full path format. Bitbucket V1 API format - /project/reponame")
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.InputFile, "input-file", "f", "", "a file in scanio format with a list of repositories to analyse. The list command could prepare this file..")
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.Config, "config", "c", "auto", "a path or type of config for a scanner. The value depends on a particular scanner's used formats.")
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.ReportFormat, "format", "o", "", "a format for a report with results.") //doesn't have default for "Uses ASCII output if no format specified"
	analyseCmd.Flags().StringSliceVar(&allArgumentsAnalyse.AdditionalArgs, "args", []string{}, "additional commands for semgrep which will be added to a semgrep call. The format in quotes with commas without spaces.")
	analyseCmd.Flags().IntVarP(&allArgumentsAnalyse.Threads, "threads", "j", 1, "number of concurrent goroutines.")
}
