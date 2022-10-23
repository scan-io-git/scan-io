/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/scan-io-git/scan-io/shared"
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/libs/common"
	"github.com/scan-io-git/scan-io/libs/vcs"
)

type RunOptionsFetch struct {
	VCSPlugName  string
	VCSURL       string
	Repositories []string
	AuthType     string
	SSHKey       string
	InputFile    string
	RmExts       string
	Threads      int
}

var (
	allArgumentsFetch RunOptionsFetch
	repositories      []string
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

func fetchRepos(repositories []string) {

	logger := shared.NewLogger("core")
	logger.Info("Fetching starting", "total", len(repositories), "goroutines", allArgumentsFetch.Threads)

	//resultChannel := make(chan vcs.FetchFuncResult)

	shared.ForEveryStringWithBoundedGoroutines(allArgumentsFetch.Threads, repositories, func(i int, repository string) {
		logger.Info("Goroutine started", "#", i+1, "project", repository)

		var resultFetch vcs.FetchFuncResult
		parsedUrl, err := url.Parse(repository)
		if err != nil {
			logger.Error("Failed", "error", resultFetch.Message)
		}
		domain := allArgumentsFetch.VCSURL
		if domain == "" {
			host, _, _ := net.SplitHostPort(parsedUrl.Host)
			domain = host
		}
		removeDotGit := regexp.MustCompile(`\.git$`)
		path := removeDotGit.ReplaceAllLiteralString(parsedUrl.Path, "")

		targetFolder := shared.GetRepoPath(domain, path)

		shared.WithPlugin("plugin-vcs", shared.PluginTypeVCS, allArgumentsFetch.VCSPlugName, func(raw interface{}) {

			vcsName := raw.(vcs.VCS)
			args := vcs.VCSFetchRequest{
				CloneURL:     repository,
				AuthType:     allArgumentsFetch.AuthType,
				SSHKey:       allArgumentsFetch.SSHKey,
				TargetFolder: targetFolder,
			}

			err := vcsName.Fetch(args)
			if err != nil {
				resultFetch = vcs.FetchFuncResult{Args: args, Result: nil, Status: "FAILED", Message: err.Error()}
				//resultChannel <- resultFetch
				logger.Error("Failed", "error", resultFetch.Message)
				logger.Debug("Failed", "debug_fetch_res", resultFetch)
			} else {
				resultFetch = vcs.FetchFuncResult{Args: args, Result: nil, Status: "OK", Message: ""}
				//resultChannel <- resultFetch
				logger.Info("Fetch fuctions is finished with status", "status", resultFetch.Status)
				logger.Debug("Success", "debug_fetch_res", resultFetch)

			}
			logger.Info("Removing files with some extentions", "extentions", allArgumentsFetch.RmExts)
			findByExtAndRemove(targetFolder, strings.Split(allArgumentsFetch.RmExts, ","))
		})
	})

	// allResults := []vcs.FetchFuncResult{}
	// close(resultChannel)
	// for status := range resultChannel {
	// 	allResults = append(allResults, status)
	// }

	logger.Info("All fetch operations are finished")
	//logger.Info("Result", "result", allResults)
	//vcs.WriteJsonFile(resultVCS, allArgumentsList.OutputFile, logger)
}

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "A brief description of your command",
	RunE: func(cmd *cobra.Command, args []string) error {
		checkArgs := func() error {
			if len(allArgumentsFetch.VCSPlugName) == 0 {
				return fmt.Errorf(("'vcs' flag must be specified"))
			}

			if len(allArgumentsFetch.VCSURL) == 0 && allArgumentsFetch.InputFile == "" {
				return fmt.Errorf(("'vcs-url' flag must be specified"))
			}

			if len(allArgumentsFetch.Repositories) != 0 && allArgumentsFetch.InputFile != "" {
				return fmt.Errorf(("you can't use both input types for repositories"))
			}

			if len(allArgumentsFetch.Repositories) == 0 && len(allArgumentsFetch.InputFile) == 0 {
				return fmt.Errorf(("'repos' or 'input-file' flag must be specified"))
			}

			if len(allArgumentsFetch.AuthType) == 0 {
				return fmt.Errorf(("'auth-type' flag must be specified"))
			}

			authType := allArgumentsFetch.AuthType
			if authType != "http" && authType != "ssh-key" && authType != "ssh-agent" {
				return fmt.Errorf("unknown auth-type - %v", authType)
			}

			if authType == "ssh-key" && len(allArgumentsFetch.SSHKey) == 0 {
				return fmt.Errorf("you must specify ssh-key with auth-type 'ssh-key'")
			}

			return nil
		}

		if err := checkArgs(); err != nil {
			return err
		}

		if allArgumentsFetch.InputFile != "" {
			repos_inf, err := common.ReadReposFile2(allArgumentsFetch.InputFile)
			if err != nil {
				return fmt.Errorf("Something happend when tool was parseing the Input File - %v", err)
			}

			if allArgumentsFetch.AuthType == "http" {
				for _, repository := range repos_inf {
					repositories = append(repositories, repository.HttpLink)
				}
			} else {
				for _, repository := range repos_inf {
					parsed_url, err := url.Parse(repository.SshLink)
					if err != nil {
						return err
					}

					if parsed_url.Scheme != "ssh" {
						return fmt.Errorf("URL for fetching has incorrect format")
					}
					repositories = append(repositories, repository.SshLink)
				}
			}
		} else {
			repositories = allArgumentsFetch.Repositories
		}
		//shared.NewLogger("core").Debug("list of repos", "repos", repos)

		if len(repositories) > 0 {
			fetchRepos(repositories)
		} else {
			return fmt.Errorf("Hasn't found no one repo")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

	fetchCmd.Flags().StringVar(&allArgumentsFetch.VCSPlugName, "vcs", "", "vcs plugin name")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.VCSURL, "vcs-url", "", "url to VCS - github.com")
	fetchCmd.Flags().StringSliceVar(&allArgumentsFetch.Repositories, "repos", []string{}, "list of repos to fetch - full path format. Bitbucket V1 API format - /project/reponame")
	fetchCmd.Flags().StringVarP(&allArgumentsFetch.InputFile, "input-file", "f", "", "file with list of repos to fetch")
	fetchCmd.Flags().IntVarP(&allArgumentsFetch.Threads, "threads", "j", 2, "number of concurrent goroutines")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.AuthType, "auth-type", "", "Type of authentication: 'http', 'ssh-agent' or 'ssh-key'")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.SSHKey, "ssh-key", "", "Path to ssh key")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.RmExts, "rm-ext", "csv,png,ipynb,txt,md,mp4,zip,gif,gz,jpg,jpeg,cache,tar,svg,bin,lock,exe", "Files with extention to remove automatically after checkout")
}
