/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/libs/vcs"
	"github.com/scan-io-git/scan-io/shared"
)

type RunOptionsFetch struct {
	VCSPlugName  string
	VCSURL       string
	Repository   string
	AuthType     string
	SSHKey       string
	TargetFolder string
	RmExts       string
	Threads      int
}

var (
	allArgumentsFetch RunOptionsFetch
	resultFetch       vcs.FetchFuncResult
)

func findByExtAndRemove(root string, exts []string) {
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		ext := filepath.Ext(d.Name())
		match := false
		for _, rmExt := range exts {
			if fmt.Sprintf(".%s", rmExt) == ext {
				match = true
				break
			}
		}
		if !match {
			return nil
		}
		e = os.Remove(s)
		if e != nil {
			return e
		}
		return nil
	})
}

func fetchRepos(repos []string) {

	logger := shared.NewLogger("core")
	logger.Info("Fetching starting", "total", len(repos), "goroutines", allArgumentsFetch.Threads)

	shared.ForEveryStringWithBoundedGoroutines(allArgumentsFetch.Threads, repos, func(i int, repository string) {
		logger.Info("Goroutine started", "#", i+1, "project", repository)

		targetFolder := shared.GetRepoPath(allArgumentsFetch.VCSURL, repository)

		shared.WithPlugin("plugin-vcs", shared.PluginTypeVCS, allArgumentsFetch.VCSPlugName, func(raw interface{}) {

			vcsName := raw.(vcs.VCS)
			args := vcs.VCSFetchRequest{
				Repository:   repository,
				AuthType:     allArgumentsFetch.AuthType,
				SSHKey:       allArgumentsFetch.SSHKey,
				VCSURL:       allArgumentsFetch.VCSURL,
				TargetFolder: targetFolder,
			}

			err := vcsName.Fetch(args)
			if err != nil {
				resultFetch = vcs.FetchFuncResult{Args: args, Result: nil, Status: "FAILED", Message: err.Error()}
				logger.Error("Failed", "error", resultFetch.Message)
				logger.Debug("Failed", "debug_fetch_res", resultFetch)
			} else {
				resultFetch = vcs.FetchFuncResult{Args: args, Result: nil, Status: "OK", Message: ""}
				logger.Info("Fetch fuctions is finished with status", "status", resultVCS.Status)
				logger.Debug("Success", "debug_fetch_res", resultFetch)

				logger.Info("Removing files with some extentions", "extentions", allArgumentsFetch.RmExts)
				findByExtAndRemove(targetFolder, strings.Split(allArgumentsFetch.RmExts, ","))
			}
		})
	})

	logger.Info("All fetch operations are finished")
}

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "A brief description of your command",
	// 	Long: `A longer description that spans multiple lines and likely contains examples
	// and usage of using your command. For example:

	// Cobra is a CLI library for Go that empowers applications.
	// This application is a tool to generate the needed files
	// to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("fetch called")
		err := cmd.Flags().Parse(args)
		if err != nil {
			panic("parse args error")
		}

		repos, err := cmd.Flags().GetStringSlice("repos")
		if err != nil {
			panic("get 'repos' arg error")
		}
		inputFile, err := cmd.Flags().GetString("input-file")
		if err != nil {
			panic("get 'input-file' arg error")
		}

		authType := allArgumentsFetch.AuthType
		if authType != "http" && authType != "ssh-key" && authType != "ssh-agent" {
			panic("unknown auth-type")
		}
		if authType == "ssh-key" && len(allArgumentsFetch.SSHKey) == 0 {
			panic("specify ssh-key with auth-type 'ssh-key'")
		}

		inputCount := 0
		// if len(org) > 0 {
		// 	inputCount += 1
		// }
		if len(repos) > 0 {
			inputCount += 1
		}
		if len(inputFile) > 0 {
			inputCount += 1
		}
		if inputCount != 1 {
			panic("you must specify one of 'repos' or 'input-file")
		}
		// if len(org) > 0 {
		// 	repos = ListRepos(vcsPluginName, vcsUrl, org, authType, sshKey)
		// }

		if len(inputFile) > 0 {
			reposFromFile, err := shared.ReadFileLines(inputFile)
			if err != nil {
				log.Fatal(err)
			}
			repos = reposFromFile
		}

		shared.NewLogger("core").Debug("list of repos", "repos", repos)

		if len(repos) > 0 {
			fetchRepos(repos)
		}
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

	fetchCmd.Flags().StringVar(&allArgumentsFetch.VCSPlugName, "vcs", "", "vcs plugin name")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.VCSURL, "vcs-url", "", "url to VCS - github.com")
	fetchCmd.Flags().StringSlice("repos", []string{}, "list of repos to fetch - full path format. Bitbucket V1 API format - /project/reponame")
	fetchCmd.Flags().StringP("input-file", "f", "", "file with list of repos to fetch")
	//fetchCmd.Flags().Bool("cache-checking", false, "Cheking existing repos varsion on a disk ")
	// fetchCmd.Flags().String("org", "", "fetch repos from this organization")
	fetchCmd.Flags().IntVarP(&allArgumentsFetch.Threads, "threads", "j", 1, "number of concurrent goroutines")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.AuthType, "auth-type", "http", "Type of authentication: 'http', 'ssh-agent' or 'ssh-key'")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.SSHKey, "ssh-key", "", "Path to ssh key")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.RmExts, "rm-ext", "csv,png,ipynb,txt,md,mp4,zip,gif,gz,jpg,jpeg,cache,tar,svg,bin,lock,exe", "Files with extention to remove automatically after checkout")
}
