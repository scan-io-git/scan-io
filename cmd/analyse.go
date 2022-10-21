/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"path/filepath"

	// "github.com/gitsight/go-vcsurl"
	"github.com/scan-io-git/scan-io/libs/common"
	"github.com/scan-io-git/scan-io/shared"
	"github.com/spf13/cobra"
)

// func getVCSURLInfo(VCSURL string, project string) (*vcsurl.VCS, error) {
// 	if strings.Contains(project, ":") {
// 		return vcsurl.Parse(project)
// 	}

// 	return vcsurl.Parse(fmt.Sprintf("https://%s/%s", VCSURL, project))
// }

func scanRepos(scannerPluginName string, repos []string, threads int) {

	logger := shared.NewLogger("core")
	logger.Info("Fetching starting", "total", len(repos), "goroutines", threads)

	shared.ForEveryStringWithBoundedGoroutines(threads, repos, func(i int, repo string) {
		logger.Info("Goroutine started", "#", i+1, "project", repo)

		repoPath := filepath.Join(shared.GetProjectsHome(), repo)
		resultsPath := filepath.Join(shared.GetResultsHome(), repo, fmt.Sprintf("%s.raw", scannerPluginName))

		shared.WithPlugin("plugin-scanner", shared.PluginTypeScanner, scannerPluginName, func(raw interface{}) {
			ok := raw.(shared.Scanner).Scan(shared.ScannerScanRequest{
				RepoPath:    repoPath,
				ResultsPath: resultsPath,
			})
			logger.Info("Scan finished", "#", i+1, "repo", repo, "results", resultsPath, "statusOK", ok)
		})
	})

	logger.Debug("All goroutines are finished.")
}

// analyseCmd represents the analyse command
var analyseCmd = &cobra.Command{
	Use:   "analyse",
	Short: "A brief description of your command",
	// 	Long: `A longer description that spans multiple lines and likely contains examples
	// and usage of using your command. For example:

	// Cobra is a CLI library for Go that empowers applications.
	// This application is a tool to generate the needed files
	// to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		err := cmd.Flags().Parse(args)
		if err != nil {
			panic("parse args error")
		}

		scannerPluginName, err := cmd.Flags().GetString("scanner")
		if err != nil {
			panic("get 'scanner' arg error")
		}
		threads, err := cmd.Flags().GetInt("threads")
		if err != nil {
			panic("get 'threads' arg error")
		}

		repos, err := cmd.Flags().GetStringSlice("repos")
		if err != nil {
			panic(err)
		}
		inputFile, err := cmd.Flags().GetString("input-file")
		if err != nil {
			panic("get 'input-file' arg error")
		}

		inputCount := 0
		if len(repos) > 0 {
			inputCount += 1
		}
		if len(inputFile) > 0 {
			inputCount += 1
		}
		if inputCount != 1 {
			panic("you must specify one of 'repos' or 'input-file")
		}

		if len(inputFile) > 0 {
			reposFromFile, err := common.ReadReposFile(inputFile)
			if err != nil {
				log.Fatal(err)
			}
			repos = reposFromFile
		}
		if len(repos) == 0 {
			panic("specify at least one 'repos' to scan")
		}

		scanRepos(scannerPluginName, repos, threads)
	},
}

func init() {
	rootCmd.AddCommand(analyseCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// analyseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// analyseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	analyseCmd.Flags().String("scanner", "semgrep", "scanner plugin name")
	// analyseCmd.Flags().String("vcs-url", "gitlab.com", "vcs url")
	analyseCmd.Flags().StringSlice("repos", []string{}, "Repos to scan")
	analyseCmd.Flags().StringP("input-file", "f", "", "file with list of repos to fetch")
	analyseCmd.Flags().IntP("threads", "j", 1, "number of concurrent goroutines")
}
