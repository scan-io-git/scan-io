package fetch

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/fetcher"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// RunOptionsFetch holds the arguments for the fetch command.
type RunOptionsFetch struct {
	VCSPluginName string
	InputFile     string
	AuthType      string
	SSHKey        string
	Branch        string
	RmListExts    []string
	Threads       int
}

// Global variables for configuration and command arguments
// TODO: add PR example for github
var (
	AppConfig         *config.Config
	fetchOptions      RunOptionsFetch
	exampleFetchUsage = `  # Fetching using SSH agent authentication, specifying a branch and URL pointing to a specific repository
  scanio fetch --vcs github --auth-type ssh-agent -b develop https://github.com/scan-io-git/scan-io

  # Fetching using SSH agent authentication, specifying a branch and SSH URL pointing to a specific repository
  scanio fetch --vcs github --auth-type ssh-agent -b develop ssh://git@github.com:scan-io-git/scan-io.git

  # Fetching using SSH agent authentication, specifying a commit hash and URL pointing to a specific repository
  scanio fetch --vcs github --auth-type ssh-agent -b c0c9e9af80666d80e564881a5bdfa661c60e053e https://github.com/scan-io-git/scan-io

  # Fetching the main branch using HTTP authentication, with a URL pointing to a specific repository
  scanio fetch --vcs github --auth-type http https://github.com/scan-io-git/scan-io

  # Fetching the main branch using SSH key authentication, with a URL pointing to a specific repository
  scanio fetch --vcs github --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 https://github.com/scan-io-git/scan-io

  # Fetching using SSH agent authentication, with a URL pointing to a specific repository, and removing specific file extensions after fetching
  scanio fetch --vcs github --auth-type ssh-agent --rm-ext zip,tar.gz,log https://github.com/scan-io-git/scan-io

  # Fetching using SSH agent authentication, specifying a branch and URL pointing to a specific project, with multiple concurrent jobs (not implemented)
  scanio fetch --vcs github --auth-type ssh-agent -b develop -j 5 https://github.com/scan-io-git/

  # Fetching using SSH agent authentication, specifying a branch and URL pointing to the whole VCS, with multiple concurrent jobs (not implemented)
  scanio fetch --vcs github --auth-type ssh-agent -b main -j 5 https://github.com/

  # Fetching from an input file using SSH agent authentication with multiple concurrent threads
  scanio fetch --vcs github --input-file /path/to/list_output.file --auth-type ssh-agent -j 5`
)

var FetchCmd = &cobra.Command{
	Use:                   "fetch --vcs/p PLUGIN_NAME --auth-type/-a AUTH_TYPE [--ssh-key/-k PATH] [--rm-ext LIST_OF_EXTENTIONS][-j THREADS_NUMBER, default=1] {--input-file/-i PATH | [-b BRANCH/HASH] URL}",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example:               exampleFetchUsage,
	Short:                 "Fetches repository code using the specified VCS plugin with consistency support",
	RunE:                  runFetchCommand,
}

// Init initializes the global configuration variable.
func Init(cfg *config.Config) {
	AppConfig = cfg
	FetchCmd.Long = generateLongDescription(AppConfig)
}

// runFetchCommand executes the analyse command.
func runFetchCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 && !shared.HasFlags(cmd.Flags()) {
		return cmd.Help()
	}

	logger := logger.NewLogger(AppConfig, "core-fetch")

	if err := validateFetchArgs(&fetchOptions, args); err != nil {
		logger.Error("invalid fetch arguments", "error", err)
		return err
	}

	mode := determineMode(args)

	f := fetcher.New(
		fetchOptions.VCSPluginName,
		fetchOptions.AuthType,
		fetchOptions.SSHKey,
		fetchOptions.Branch,
		fetchOptions.RmListExts,
		fetchOptions.Threads,
		logger,
	)

	reposParams, err := prepareFetchTargets(&fetchOptions, args, mode)
	if err != nil {
		logger.Error("failed to prepare fetch targets", "error", err)
		return err
	}

	fetchReqList, err := f.PrepFetchReqList(AppConfig, reposParams)
	if err != nil {
		logger.Error("failed to prepare fetch targets", "error", err)
		return err
	}

	retchResult, fetchErr := f.FetchRepos(AppConfig, fetchReqList)

	if err := shared.WriteGenericResult(AppConfig, logger, retchResult, "FETCH"); err != nil {
		logger.Error("failed to write result", "error", err)
		return err
	}

	if fetchErr != nil {
		logger.Error("fetch command failed", "error", fetchErr)
		return fetchErr
	}

	logger.Debug("fetch result", "result", retchResult)
	logger.Info("fetch command completed successfully")
	return nil
}

// generateLongDescription generates the long description dynamically with the list of available scanner plugins.
func generateLongDescription(AppConfig *config.Config) string {
	pluginsMeta := shared.GetPluginVersions(config.GetScanioPluginsHome(AppConfig), "vcs")
	var plugins []string
	for plugin := range pluginsMeta {
		plugins = append(plugins, plugin)
	}
	return fmt.Sprintf(`Fetches repository code using the specified VCS plugin with consistency support.

List of avaliable vcs plugins:
  %s`, strings.Join(plugins, "\n  "))
}

func init() {
	//TODO: add output
	FetchCmd.Flags().StringVarP(&fetchOptions.VCSPluginName, "vcs", "p", "", "Name of the VCS plugin to use (e.g., bitbucket, gitlab, github).")
	FetchCmd.Flags().StringVarP(&fetchOptions.InputFile, "input-file", "i", "", "Path to a file in Scanio format containing a list of repositories to fetch. Use the list command to prepare this file.")
	FetchCmd.Flags().StringVarP(&fetchOptions.AuthType, "auth-type", "a", "", "Type of authentication (e.g., http, ssh-agent, ssh-key).")
	FetchCmd.Flags().StringVarP(&fetchOptions.SSHKey, "ssh-key", "k", "", "Path to an SSH key.")
	FetchCmd.Flags().StringVarP(&fetchOptions.Branch, "branch", "b", "", "Specific branch to fetch (default: main or master).")
	FetchCmd.Flags().BoolP("help", "h", false, "Show help for the fetch command.")
	FetchCmd.Flags().StringSliceVar(&fetchOptions.RmListExts, "rm-ext", []string{"csv", "png", "ipynb", "txt", "md", "mp4", "zip", "gif", "gz", "jpg", "jpeg", "cache", "tar", "svg", "bin", "lock", "exe"}, "Comma-separated list of file extensions to remove automatically after fetching.")
	FetchCmd.Flags().IntVarP(&fetchOptions.Threads, "threads", "j", 1, "Number of concurrent threads to use.")
}
