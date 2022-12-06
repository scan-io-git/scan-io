package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

var (
	allArgumentsAnalyse RunOptionsAnalyse
)

func scanRepos(analyseArgs []shared.ScannerScanRequest) {

	logger := shared.NewLogger("core")
	logger.Info("Scan starting", "total", len(analyseArgs), "goroutines", allArgumentsAnalyse.Threads)
	values := make([]interface{}, len(analyseArgs))
	for i := range analyseArgs {
		values[i] = analyseArgs[i]
	}

	shared.ForEveryStringWithBoundedGoroutines(allArgumentsAnalyse.Threads, values, func(i int, value interface{}) {
		args := value.(shared.ScannerScanRequest)
		logger.Info("Goroutine started", "#", i+1, "args", args)

		resultsFolder := args.ResultsPath[:strings.LastIndex(args.ResultsPath, "/")]
		err := os.MkdirAll(resultsFolder, os.ModePerm)
		if err != nil {
			logger.Error("create resultsFolder failed", "resultsFolder", resultsFolder, "error", err)
			return
		}

		shared.WithPlugin("plugin-scanner", shared.PluginTypeScanner, allArgumentsAnalyse.ScannerPluginName, func(raw interface{}) {

			var resultScan shared.ScannerScanResult
			scanName := raw.(shared.Scanner)

			err := scanName.Scan(args)
			if err != nil {
				resultScan = shared.ScannerScanResult{Args: args, Result: nil, Status: "FAILED", Message: err.Error()}
				//resultChannel <- resultFetch
				logger.Error("Failed", "error", resultScan.Message)
				logger.Debug("Failed", "debug_fetch_res", resultScan)
			} else {
				resultScan = shared.ScannerScanResult{Args: args, Result: nil, Status: "OK", Message: ""}
				//resultChannel <- resultFetch
				logger.Info("Analyze fuctions is finished with status", "#", i+1, "args", args, "status", resultScan.Status)
				logger.Debug("Success", "debug_fetch_res", resultScan)

			}
		})
	})
	logger.Info("All analyze operations are finished")
}

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
		analyseArgs := []shared.ScannerScanRequest{}

		reposInf, err := utils.ReadReposFile2(allArgumentsAnalyse.InputFile)
		if err != nil {
			return fmt.Errorf("something happend when tool was parsing the Input File - %v", err)
		}

		for _, repository := range reposInf {
			domain, err := utils.GetDomain(repository.SshLink)
			if err != nil {
				domain, err = utils.GetDomain(repository.HttpLink)
				if err != nil {
					return err
				}
				// return err
			}

			targetFolder := shared.GetRepoPath(domain, filepath.Join(repository.Namespace, repository.RepoName))
			resultsPath := filepath.Join(shared.GetResultsHome(), domain, filepath.Join(repository.Namespace, repository.RepoName), fmt.Sprintf("%s.raw", allArgumentsAnalyse.ScannerPluginName))
			analyseArgs = append(analyseArgs, shared.ScannerScanRequest{
				RepoPath:       targetFolder,
				ResultsPath:    resultsPath,
				ConfigPath:     allArgumentsAnalyse.Config,
				AdditionalArgs: allArgumentsAnalyse.AdditionalArgs,
				ReportFormat:   allArgumentsAnalyse.ReportFormat,
			})
			//shared.NewLogger("core").Info(fmt.Sprintf("%v/%v", repository.Namespace, repository.RepoName))
		}

		if len(analyseArgs) > 0 {
			scanRepos(analyseArgs)
		} else {
			return fmt.Errorf("hasn't found no one repo")
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
