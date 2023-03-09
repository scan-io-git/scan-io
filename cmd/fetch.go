package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/fetcher"
	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
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

var allArgumentsFetch RunOptionsFetch

var fetchCmd = &cobra.Command{
	Use:          "fetch",
	SilenceUsage: true,
	Short:        "The main function is to fetch code of a specified repositories and do consistency support.",

	RunE: func(cmd *cobra.Command, args []string) error {
		checkArgs := func() error {
			if len(allArgumentsFetch.VCSPlugName) == 0 {
				return fmt.Errorf(("'vcs' flag must be specified!"))
			}

			if len(allArgumentsFetch.VCSURL) == 0 && allArgumentsFetch.InputFile == "" {
				return fmt.Errorf(("'vcs-url' flag must be specified!"))
			}

			if len(allArgumentsFetch.Repositories) != 0 && allArgumentsFetch.InputFile != "" {
				return fmt.Errorf(("You can't use both input types for repositories!"))
			}

			// if len(allArgumentsFetch.Repositories) == 0 && len(allArgumentsFetch.InputFile) == 0 {
			// 	return fmt.Errorf(("'repos' or 'input-file' flag must be specified!"))
			// }

			if len(allArgumentsFetch.InputFile) == 0 {
				return fmt.Errorf(("'input-file' flag must be specified!"))
			}

			if len(allArgumentsFetch.AuthType) == 0 {
				return fmt.Errorf(("'auth-type' flag must be specified!"))
			}

			authType := allArgumentsFetch.AuthType
			if authType != "http" && authType != "ssh-key" && authType != "ssh-agent" {
				return fmt.Errorf("unknown auth-type - %v!", authType)
			}

			if authType == "ssh-key" && len(allArgumentsFetch.SSHKey) == 0 {
				return fmt.Errorf("You must specify ssh-key with auth-type 'ssh-key'!")
			}

			return nil
		}

		if err := checkArgs(); err != nil {
			return err
		}

		reposParams, err := utils.ReadReposFile2(allArgumentsFetch.InputFile)
		if err != nil {
			return err
		}

		logger := shared.NewLogger("core-fetcher")
		fetcher := fetcher.New(allArgumentsFetch.AuthType, allArgumentsFetch.SSHKey, allArgumentsFetch.Threads, allArgumentsFetch.VCSPlugName, strings.Split(allArgumentsFetch.RmExts, ","), logger)

		fetchArgs, err := fetcher.PrepFetchArgs(reposParams)
		if err != nil {
			return err
		}

		err = fetcher.FetchRepos(fetchArgs)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

	fetchCmd.Flags().StringVar(&allArgumentsFetch.VCSPlugName, "vcs", "", "the plugin name of the VCS used. Eg. bitbucket, gitlab, github, etc.")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.VCSURL, "vcs-url", "", "URL to a root of the VCS API. Eg. github.com.")
	// fetchCmd.Flags().StringSliceVar(&allArgumentsFetch.Repositories, "repos", []string{}, "the list of repositories to fetching. Bitbucket V1 API format - /project/reponame.")
	fetchCmd.Flags().StringVarP(&allArgumentsFetch.InputFile, "input-file", "f", "", "a file in scanio format with list of repositories to fetching. The list command could prepare this file.")
	fetchCmd.Flags().IntVarP(&allArgumentsFetch.Threads, "threads", "j", 1, "number of concurrent goroutines.")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.AuthType, "auth-type", "", "type of authentication: 'http', 'ssh-agent' or 'ssh-key'.")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.SSHKey, "ssh-key", "", "the path to an ssh key.")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.RmExts, "rm-ext", "csv,png,ipynb,txt,md,mp4,zip,gif,gz,jpg,jpeg,cache,tar,svg,bin,lock,exe", "extensions of files to remove it automatically after fetching.")
}
