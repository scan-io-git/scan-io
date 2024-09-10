package fetch

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/fetcher"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// RunOptionsFetch holds the arguments for the fetch command.
type RunOptionsFetch struct {
	VCSPluginName string   `json:"vcs_plugin_name,omitempty"`
	InputFile     string   `json:"input_file,omitempty"`
	AuthType      string   `json:"auth_type,omitempty"`
	SSHKey        string   `json:"ssh_key,omitempty"`
	Branch        string   `json:"branch,omitempty"`
	OutputPath    string   `json:"output_path,omitempty"`
	RmListExts    []string `json:"rm_list_exts"`
	Threads       int      `json:"threads"`
}

// Global variables for configuration and command arguments
// TODO: add PR example for github
// output example
var (
	AppConfig         *config.Config
	fetchOptions      RunOptionsFetch
	exampleFetchUsage = `  # Fetching using SSH agent authentication, specifying a branch and URL pointing to a specific repository
  scanio fetch --vcs github --auth-type ssh-agent -b develop https://github.com/scan-io-git/scan-io

  # Fetching using SSH agent authentication, specifying an output folder and URL pointing to a specific repository
  scanio fetch --vcs github --auth-type ssh-agent -o /path/to/repo_folder/ https://github.com/scan-io-git/scan-io

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
  scanio fetch --vcs github --input-file /path/to/list_output.file --auth-type ssh-agent -j 5
  
  # Fetching using SSH agent authentication, specifying a branch and URL pointing to the whole VCS, with multiple concurrent jobs (not implemented)
  scanio fetch --vcs github --auth-type ssh-agent -b main -j 5 https://github.com/`
)

// FetchCmd represents the command for fetch command.
var FetchCmd = &cobra.Command{
	Use:                   "fetch --vcs/p PLUGIN_NAME --auth-type/-a AUTH_TYPE [--ssh-key/-k PATH] [--output/-o PATH] [--rm-ext LIST_OF_EXTENTIONS][-j THREADS_NUMBER, default=1] {--input-file/-i PATH | [-b BRANCH/HASH] URL}",
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
		return errors.NewCommandError(fetchOptions, nil, fmt.Errorf("invalid fetch arguments: %w", err), 1)
	}

	mode := determineMode(args)

	f := fetcher.New(
		fetchOptions.VCSPluginName,
		fetchOptions.AuthType,
		fetchOptions.SSHKey,
		fetchOptions.Branch,
		fetchOptions.OutputPath,
		fetchOptions.RmListExts,
		fetchOptions.Threads,
		logger,
	)

	reposParams, err := prepareFetchTargets(&fetchOptions, args, mode)
	if err != nil {
		logger.Error("failed to prepare fetch targets", "error", err)
		return errors.NewCommandError(fetchOptions, nil, fmt.Errorf("failed to prepare fetch targets: %w", err), 1)
	}

	fetchReqList, err := f.PrepFetchReqList(AppConfig, reposParams)
	if err != nil {
		logger.Error("failed to prepare fetch arguments", "error", err)
		return errors.NewCommandError(fetchOptions, nil, fmt.Errorf("failed to prepare fetch arguments: %w", err), 1)
	}

	fetchResult, fetchErr := f.FetchRepos(AppConfig, fetchReqList)

	metaDataFileName := fmt.Sprintf("FETCH_%s", strings.ToUpper(f.PluginName))
	if config.IsCI(AppConfig) {
		startTime := time.Now().UTC().Format(time.RFC3339)
		metaDataFileName = fmt.Sprintf("FETCH_%s_%v", strings.ToUpper(f.PluginName), startTime)
	}
	if err := shared.WriteGenericResult(AppConfig, logger, fetchResult, metaDataFileName); err != nil {
		logger.Error("failed to write result", "error", err)
	}

	if fetchErr != nil {
		logger.Error("fetch command failed", "error", fetchErr)
		return errors.NewCommandErrorWithResult(fetchResult, fmt.Errorf("fetch command failed: %w", fetchErr), 2)
	}

	logger.Info("fetch command completed successfully")
	logger.Debug("fetch result", "result", fetchResult)
	if config.IsCI(AppConfig) {
		shared.PrintResultAsJSON(logger, fetchResult)
	}
	return nil
}

// generateLongDescription generates the long description dynamically with the list of available scanner plugins.
func generateLongDescription(cfg *config.Config) string {
	pluginsMeta := shared.GetPluginVersions(config.GetScanioPluginsHome(cfg), "vcs")
	var plugins []string
	for plugin := range pluginsMeta {
		plugins = append(plugins, plugin)
	}
	return fmt.Sprintf(`Fetches repository code using the specified VCS plugin with consistency support.

List of avaliable vcs plugins:
  %s`, strings.Join(plugins, "\n  "))
}

func init() {
	FetchCmd.Flags().StringVarP(&fetchOptions.VCSPluginName, "vcs", "p", "", "Name of the VCS plugin to use (e.g., bitbucket, gitlab, github).")
	FetchCmd.Flags().StringVarP(&fetchOptions.InputFile, "input-file", "i", "", "Path to a file in Scanio format containing a list of repositories to fetch. Use the list command to prepare this file.")
	FetchCmd.Flags().StringVarP(&fetchOptions.AuthType, "auth-type", "a", "", "Type of authentication (e.g., http, ssh-agent, ssh-key).")
	FetchCmd.Flags().StringVarP(&fetchOptions.SSHKey, "ssh-key", "k", "", "Path to an SSH key.")
	FetchCmd.Flags().StringVarP(&fetchOptions.Branch, "branch", "b", "", "Specific branch to fetch (default: main or master).")
	FetchCmd.Flags().StringVarP(&fetchOptions.OutputPath, "output", "o", "", "Path to the output directory where the cmd results will be saved.")
	FetchCmd.Flags().StringSliceVar(&fetchOptions.RmListExts, "rm-ext", []string{"csv", "png", "ipynb", "txt", "md", "mp4", "zip", "gif", "gz", "jpg", "jpeg", "cache", "tar", "svg", "bin", "lock", "exe"}, "Comma-separated list of file extensions to remove automatically after fetching.")
	FetchCmd.Flags().IntVarP(&fetchOptions.Threads, "threads", "j", 1, "Number of concurrent threads to use.")
	FetchCmd.Flags().BoolP("help", "h", false, "Show help for the fetch command.")
}
