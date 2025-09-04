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
	PrMode        string   `json:"pr_mode,omitempty"`
	SingleBranch  bool     `json:"single_branch,omitempty"`
	Tags          bool     `json:"tags,omitempty"`
	NoTags        bool     `json:"no_tags,omitempty"`
	Depth         int      `json:"depth,omitempty"`
	AutoRepair    bool     `json:"auto_repair,omitempty"`
	CleanWorkdir  bool     `json:"clean_workdir,omitempty"`
	RmListExts    []string `json:"rm_list_exts"`
	Threads       int      `json:"threads"`
}

// Global variables for configuration and command arguments
var (
	AppConfig         *config.Config
	fetchOptions      RunOptionsFetch
	exampleFetchUsage = `  # Fetching using SSH agent authentication, URL pointing to a specific repository
  scanio fetch --vcs github --auth-type ssh-agent https://github.com/scan-io-git/scan-io

  # Fetching using SSH agent authentication, specifying an output folder and URL pointing to a specific repository
  scanio fetch --vcs github --auth-type ssh-agent -o /path/to/repo_folder/ https://github.com/scan-io-git/scan-io

  # Fetching using SSH agent authentication, specifying a branch and URL pointing to a specific repository
  scanio fetch --vcs github --auth-type ssh-agent -b develop https://github.com/scan-io-git/scan-io

  # Fetching using SSH agent authentication, specifying a commit hash and URL pointing to a specific repository
  scanio fetch --vcs github --auth-type ssh-agent -b c0c9e9af80666d80e564881a5bdfa661c60e053e https://github.com/scan-io-git/scan-io

  # Fetching the tip of pull request using SSH key authentication, with a URL pointing to a specific pull request
  scanio fetch --vcs github --auth-type ssh-agent https://github.com/scan-io-git/scan-io/pull/1

  # Fetching the main branch using HTTP authentication, with a URL pointing to a specific repository
  scanio fetch --vcs github --auth-type http https://github.com/scan-io-git/scan-io

  # Fetching the main branch using SSH key authentication, with a URL pointing to a specific repository
  scanio fetch --vcs github --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 https://github.com/scan-io-git/scan-io

  # Fetching using SSH agent authentication, with a URL pointing to a specific repository, and removing specific file extensions after fetching
  scanio fetch --vcs github --auth-type ssh-agent --rm-ext zip,tar.gz,log https://github.com/scan-io-git/scan-io

  # Fetching from an input file from the list cmd using SSH agent authentication with multiple concurrent threads
  scanio fetch --vcs github --input-file /path/to/list_output.file --auth-type ssh-agent -j 5`
)

// FetchCmd represents the command for fetch command.
var FetchCmd = &cobra.Command{
	Use:                   "fetch --vcs/p PLUGIN_NAME --auth-type/-a AUTH_TYPE [--ssh-key/-k PATH] [--output/-o PATH] [--rm-ext LIST_OF_EXTENTIONS][-j THREADS_NUMBER, default=1][--pr-mode PR_MODE][--single-branch][--depth DEPTH, default=0][--tags][--no-tags] {--input-file/-i PATH | [-b/--branch BRANCH/HASH] URL}",
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

	cmdMode := determineCmdMode(args)
	tagMode, err := determineAddFlags(cmd, &fetchOptions)
	if err != nil {
		return errors.NewCommandError(fetchOptions, nil, err, 1)
	}

	reposParams, err := prepareFetchTargets(&fetchOptions, args, cmdMode, logger)
	if err != nil {
		logger.Error("failed to prepare fetch targets", "error", err)
		return errors.NewCommandError(fetchOptions, nil, fmt.Errorf("failed to prepare fetch targets: %w", err), 1)
	}

	f := fetcher.New(
		fetchOptions.VCSPluginName,
		fetchOptions.AuthType,
		fetchOptions.SSHKey,
		fetchOptions.OutputPath,
		fetchOptions.RmListExts,
		fetchOptions.AutoRepair,
		fetchOptions.CleanWorkdir,
		fetchOptions.Threads,
		logger,
	)

	fetchReqList, err := f.PrepFetchReqList(AppConfig, reposParams, fetchOptions.PrMode, fetchOptions.Depth, fetchOptions.SingleBranch, tagMode)
	if err != nil {
		logger.Error("failed to prepare fetch requests", "error", err)
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

List of available vcs plugins:
  %s`, strings.Join(plugins, "\n  "))
}

func init() {
	FetchCmd.Flags().StringVarP(&fetchOptions.VCSPluginName, "vcs", "p", "", "Name of the VCS plugin to use (e.g., bitbucket, gitlab, github).")
	FetchCmd.Flags().StringVarP(&fetchOptions.InputFile, "input-file", "i", "", "Path to a file in Scanio format containing a list of repositories to fetch. Use the list command to prepare this file.")
	FetchCmd.Flags().StringVarP(&fetchOptions.AuthType, "auth-type", "a", "", "Type of authentication (e.g., http, ssh-agent, ssh-key).")
	FetchCmd.Flags().StringVarP(&fetchOptions.SSHKey, "ssh-key", "k", "", "Path to the SSH key to use for authentication.")
	FetchCmd.Flags().StringVarP(&fetchOptions.Branch, "branch", "b", "", "Specific branch to fetch. Default: main or master. Implies --single-branch.")
	FetchCmd.Flags().StringVarP(&fetchOptions.OutputPath, "output", "o", "", "Directory where the fetched repository will be saved.")
	FetchCmd.Flags().StringVarP(&fetchOptions.PrMode, "pr-mode", "", "", "PR fetching mode: 'branch', 'ref', or 'commit'.")
	FetchCmd.Flags().BoolVar(&fetchOptions.SingleBranch, "single-branch", false, "Fetch only the specified branch without history from other branches.")
	FetchCmd.Flags().IntVar(&fetchOptions.Depth, "depth", -1, "Create a shallow clone with a history truncated to the specified number of commits. Default: 1 in CI mode, 0 in User mode.")
	FetchCmd.Flags().BoolVar(&fetchOptions.Tags, "tags", false, "Fetch all tags from the repository.")
	FetchCmd.Flags().BoolVar(&fetchOptions.NoTags, "no-tags", false, "Do not fetch any tags from the repository.")
	FetchCmd.Flags().BoolVar(&fetchOptions.AutoRepair, "auto-repair", false, "Automatically repair corrupted repositories by forcing a refetch and recloning if needed.")
	FetchCmd.Flags().BoolVar(&fetchOptions.CleanWorkdir, "clean-workdir", false, "Reset the working tree to HEAD and remove untracked files (like 'git reset --hard' + 'git clean -fdx').")
	FetchCmd.Flags().StringSliceVar(&fetchOptions.RmListExts, "rm-ext", []string{"csv", "png", "ipynb", "txt", "md", "mp4", "zip", "gif", "gz", "jpg", "jpeg", "cache", "tar", "svg", "bin", "lock", "exe"}, "Comma-separated list of file extensions to remove automatically after fetching.")
	FetchCmd.Flags().IntVarP(&fetchOptions.Threads, "threads", "j", 1, "Number of concurrent threads to use.")
	FetchCmd.Flags().BoolP("help", "h", false, "Show help for the fetch command.")
}
