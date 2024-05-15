package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/fetcher"
	"github.com/scan-io-git/scan-io/internal/logger"
	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
)

type RunOptionsFetch struct {
	VCSPlugName string
	AuthType    string
	SSHKey      string
	InputFile   string
	RmExts      string
	Branch      string
	Threads     int
}

var (
	allArgumentsFetch RunOptionsFetch
	resultFetch       shared.GenericLaunchesResult
	execExampleFetch  = `  # Fetching from an input file using an ssh-key authentification
  scanio fetch --vcs bitbucket --input-file /Users/root/.scanio/output.file --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 -j 1

  # Fetching using an ssh-key authentification, branch and URL that points a specific repository.
  scanio fetch --vcs bitbucket --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 -j 1 -b develop https://example.com/projects/scanio_project/repos/scanio/browse

  # Fetching from an input file using an ssh-agent authentification
  scanio fetch --vcs bitbucket --input-file /Users/root/.scanio/output.file --auth-type ssh-agent -j 1

  # Fetching from an input file with an HTTP.
  scanio fetch --vcs bitbucket --input-file /Users/root/.scanio/output.file --auth-type http -j 1`
)

var fetchCmd = &cobra.Command{
	Use:                   "fetch --vcs PLUGIN_NAME --output /local_path/output.file --auth-type AUTH_TYPE [--ssh-key /local_path/.ssh/id_ed25519] [--rm-ext LIST_OF_EXTENTIONS] [-j THREADS_NUMBER] (--input-file/-i /local_path/repositories.file | [-b BRANCH] <url>)",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example:               execExampleFetch,
	Short:                 "The main function is to fetch code of a specified repositories and do consistency support",
	Long: `The main function is to fetch code of a specified repositories and do consistency support

List of plugins:
  - bitbucket
  - gitlab
  - github`,

	RunE: func(cmd *cobra.Command, args []string) error {
		// Decision maker MVP needs
		var outputBuffer bytes.Buffer
		reposParams := []shared.RepositoryParams{}

		checkArgs := func() error {
			if len(args) >= 2 {
				return fmt.Errorf("Invalid argument(s) received!")
			}
			if len(allArgumentsFetch.VCSPlugName) == 0 {
				return fmt.Errorf(("'vcs' flag must be specified!"))
			}

			if len(allArgumentsFetch.InputFile) == 0 && len(args) == 0 {
				return fmt.Errorf(("'input-file' flag or URL must be specified!"))
			}

			if len(allArgumentsFetch.InputFile) != 0 && len(args) != 0 {
				return fmt.Errorf(("You can't use a few input types for repositories!"))
			}

			if len(args) == 1 {
				if len(allArgumentsFetch.InputFile) != 0 {
					return fmt.Errorf(("You can't use a specific url with an input-file argument!"))
				}

				URL := args[0]
				_, namespace, repository, pullRequestId, httpLink, sshLink, err := shared.ExtractRepositoryInfoFromURL(URL, allArgumentsFetch.VCSPlugName)
				if err != nil {
					return err
				}

				if len(namespace) == 0 {
					return fmt.Errorf(("A fetch function for fetching all VCS is not supported."))
				} else if len(repository) == 0 {
					return fmt.Errorf(("A fetch function for fetching a whole project is not supported."))
				}
				reposParams = append(reposParams, shared.RepositoryParams{
					Namespace: namespace,
					RepoName:  repository,
					PRID:      pullRequestId,
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

		logger := logger.NewLogger(AppConfig, "core-fetcher")
		fetcher := fetcher.New(allArgumentsFetch.AuthType, allArgumentsFetch.SSHKey, allArgumentsFetch.Threads, allArgumentsFetch.Branch, allArgumentsFetch.VCSPlugName, strings.Split(allArgumentsFetch.RmExts, ","), logger)

		fetchArgs, err := fetcher.PrepFetchArgs(logger, reposParams)
		if err != nil {
			return err
		}

		resultFetch = fetcher.FetchRepos(AppConfig, fetchArgs)
		resultJSON, err := json.Marshal(resultFetch)
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

		logger.Debug("Integration result", "result", resultFetch)

		shared.WriteJsonFile(fmt.Sprintf("%v/FETCH.scanio-result", shared.GetScanioHome()), logger, resultFetch)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

	fetchCmd.Flags().StringVar(&allArgumentsFetch.VCSPlugName, "vcs", "", "the plugin name of the VCS used. Eg. bitbucket, gitlab, github, etc.")
	fetchCmd.Flags().StringVarP(&allArgumentsFetch.InputFile, "input-file", "i", "", "a file in Scanio format with list of repositories to fetching. The list command could prepare this file.")
	fetchCmd.Flags().IntVarP(&allArgumentsFetch.Threads, "threads", "j", 1, "number of concurrent goroutines.")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.AuthType, "auth-type", "", "type of authentication: 'http', 'ssh-agent' or 'ssh-key'.")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.SSHKey, "ssh-key", "", "the path to an ssh key.")
	fetchCmd.Flags().StringVarP(&allArgumentsFetch.Branch, "branch", "b", "", "a specific branch for fetching. You can use it manual URL mode. A default value is main or master.")
	fetchCmd.Flags().StringVar(&allArgumentsFetch.RmExts, "rm-ext", "csv,png,ipynb,txt,md,mp4,zip,gif,gz,jpg,jpeg,cache,tar,svg,bin,lock,exe", "extensions of files to remove it automatically after fetching.")
}
