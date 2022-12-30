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
	Use:   "analyse",
	Short: "A brief description of your command",
	RunE: func(cmd *cobra.Command, args []string) error {
		checkArgs := func() error {
			if len(allArgumentsAnalyse.ScannerPluginName) == 0 {
				return fmt.Errorf("'scanner' flag must be specified")
			}

			// if len(allArgumentsAnalyse.Repositories) != 0 && allArgumentsAnalyse.InputFile != "" {
			// 	return fmt.Errorf("you can't use both input types for repositories")
			// }

			// if len(allArgumentsAnalyse.Repositories) == 0 && len(allArgumentsAnalyse.InputFile) == 0 {
			// 	return fmt.Errorf("'repos' or 'input-file' flag must be specified")
			// }
			// fmt.Println(allArgumentsAnalyse.InputFile)
			if allArgumentsAnalyse.InputFile == "" {
				return fmt.Errorf("'input-file' flag must be specified")
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

	analyseCmd.Flags().StringVar(&allArgumentsAnalyse.ScannerPluginName, "scanner", "semgrep", "scanner plugin name")
	// analyseCmd.Flags().StringSliceVar(&allArgumentsAnalyse.Repositories, "repos", []string{}, "list of repos to analyse - full path format. Bitbucket V1 API format - /project/reponame")
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.InputFile, "input-file", "f", "", "file with list of repos to analyse")
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.Config, "config", "c", "auto", "file with list of repos to analyse")
	analyseCmd.Flags().StringVarP(&allArgumentsAnalyse.ReportFormat, "format", "o", "", "file with list of repos to analyse") //doesn't have default for "Uses ASCII output if no format specified"
	analyseCmd.Flags().StringSliceVar(&allArgumentsAnalyse.AdditionalArgs, "args", []string{}, "additional commands for semgrep which are will be added to a semgrep call. Format in quots with commas withous spaces.")
	analyseCmd.Flags().IntVarP(&allArgumentsAnalyse.Threads, "threads", "j", 2, "number of concurrent goroutines")
}
