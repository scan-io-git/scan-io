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
	VCSPlugName string
	AuthType    string
	SSHKey      string
	InputFile   string
	RmExts      string
	Threads     int
}

var allArgumentsFetch RunOptionsFetch

var fetchCmd = &cobra.Command{
	Use:          "fetch [flags] \n  scanio fetch [flags] [url]",
	SilenceUsage: true,
	Short:        "The main function is to fetch code of a specified repositories and do consistency support.",

	RunE: func(cmd *cobra.Command, args []string) error {
		reposParams := []shared.RepositoryParams{}

		checkArgs := func() error {
			if len(args) >= 2 {
				return fmt.Errorf("Invalid argument(s) received!")
			}
			if len(allArgumentsFetch.VCSPlugName) == 0 {
				return fmt.Errorf(("'vcs' flag must be specified!"))
			}

			if len(allArgumentsFetch.InputFile) == 0 && len(args) == 0 {
				return fmt.Errorf(("'vcs-url' flag or 'input-file' flag or URL must be specified!"))
			}

			if len(allArgumentsFetch.InputFile) != 0 && len(args) != 0 {
				return fmt.Errorf(("You can't use a few input types for repositories!"))
			}

			if len(args) == 1 {
				if len(allArgumentsFetch.InputFile) != 0 {
					return fmt.Errorf(("You can't use a specific url with an input-file argument!"))
				}

				URL := args[0]
				_, namespace, repository, httpLink, sshLink, err := shared.ExtractRepositoryInfoFromURL(URL, allArgumentsFetch.VCSPlugName)
				if len(repository) == 0 {
					return fmt.Errorf(("A fetch function for a whole project is not supported."))
				}

				if err != nil {
					return err
				}
				reposParams = append(reposParams, shared.RepositoryParams{
					Namespace: namespace,
					RepoName:  repository,
					HttpLink:  httpLink,
					SshLink:   sshLink,
				})

			} else {
				if len(allArgumentsFetch.InputFile) == 0 {
					return fmt.Errorf(("'input-file' flag must be specified!"))
				}

				reposData, err := utils.ReadReposFile2(allArgumentsFetch.InputFile)
				if err != nil {
					return err
				}
				reposParams = reposData
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
	fetchCmd.Flags().StringVarP(&allArgumentsFetch.InputFile, "input-file", "f", "", "a file in scanio format with list of repositories to fetching. The list command could prepare this file.")
	fetchCmd.Flags().IntVarP(&allArgumentsFetch.Threads, "threads", "j", 1, "number of concurrent goroutines.")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.AuthType, "auth-type", "", "type of authentication: 'http', 'ssh-agent' or 'ssh-key'.")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.SSHKey, "ssh-key", "", "the path to an ssh key.")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.RmExts, "rm-ext", "csv,png,ipynb,txt,md,mp4,zip,gif,gz,jpg,jpeg,cache,tar,svg,bin,lock,exe", "extensions of files to remove it automatically after fetching.")
}
