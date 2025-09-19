package list

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/vcsintegrator"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/artifacts"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
)

// Global variables for configuration and command arguments
var (
	AppConfig   *config.Config
	logger      hclog.Logger
	listOptions vcsintegrator.RunOptionsIntegrationVCS

	exampleListUsage = `  # List all repositories in a VCS
  scanio list --vcs github --domain github.com -o /path/to/list_output.file

  # List all repositories in a specific namespace in a VCS
  scanio list --vcs github --domain github.com --namespace scan-io-git -o /path/to/list_output.file

  # List all repositories in a VCS using a direct URL
  scanio list --vcs github -o /path/tolist_output.file https://github.com/

  # List all repositories in a specific namespace using a direct URL
  scanio list --vcs github -o /path/to/list_output.file https://github.com/scan-io-git/`
)

// ListCmd represents the command for list command.
var ListCmd = &cobra.Command{
	Use:                   "list --vcs/-p PLUGIN_NAME --output/-o PATH [--language LANGUAGE] {--domain VCS_DOMAIN_NAME --namespace NAMESPACE | URL}",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example:               exampleListUsage,
	Short:                 "List repositories from a version control system",
	RunE:                  runListCommand,
}

// Init initializes the global configuration variable and sets the long description for the ListCmd command.
func Init(cfg *config.Config, l hclog.Logger) {
	AppConfig = cfg
	logger = l
	ListCmd.Long = generateLongDescription(AppConfig)
}

func runListCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 && !shared.HasFlags(cmd.Flags()) {
		return cmd.Help()
	}

	if err := validateListArgs(&listOptions, args); err != nil {
		logger.Error("invalid list arguments", "error", err)
		return errors.NewCommandError(listOptions, nil, fmt.Errorf("invalid list arguments: %w", err), 1)
	}

	listOptions.Action = vcsintegrator.VCSListing
	mode := determineMode(args)

	i := vcsintegrator.New(
		listOptions.VCSPluginName,
		listOptions.Action,
		logger,
	)

	repoParams, err := prepareListTarget(&listOptions, args, mode)
	if err != nil {
		logger.Error("failed to prepare fetch targets", "error", err)
		return errors.NewCommandError(listOptions, nil, fmt.Errorf("failed to prepare fetch targets: %w", err), 1)
	}

	if repoParams.Repository != "" {
		logger.Warn("Listing a particular repository is not supported. The namespace will be listed instead", "namespace", repoParams.Namespace)
	}

	listRequest, err := i.PrepIntegrationRequest(AppConfig, &listOptions, repoParams)
	if err != nil {
		logger.Error("failed to prepare integration list request", "error", err)
		return errors.NewCommandError(listOptions, nil, fmt.Errorf("failed to prepare integration list request: %w", err), 1)
	}

	resultList, listErr := i.IntegrationAction(AppConfig, listRequest)

	if config.IsCI(AppConfig) {
		if _, err := artifacts.SaveArtifactJSON(AppConfig, logger, "list", i.PluginName, resultList); err != nil {
			logger.Error("failed to write artifact", "error", err)
		}
	}

	if listErr != nil {
		logger.Error("list command failed", "error", listErr)
		return errors.NewCommandErrorWithResult(resultList, fmt.Errorf("list command failed: %w", listErr), 2)
	}

	vcsData, ok := resultList.Launches[0].Result.([]shared.VCSParams)
	if !ok {
		return fmt.Errorf("failed to parse results")
	}

	// TODO: fix temporary code
	resultData, err := json.MarshalIndent(vcsData[0].Namespaces, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshaling the result data: %w", err)
	}
	if err := files.WriteJsonFile(listOptions.OutputPath, resultData); err != nil {
		logger.Error("failed to write result", "error", err)
		return err
	}

	logger.Info("list command completed successfully")
	logger.Info("results saved to file", "path", listOptions.OutputPath)
	logger.Info("statistic", "number_namespaces", vcsData[0].NamespaceCount, "number_repositories", vcsData[0].RepositoryCount)
	if config.IsCI(AppConfig) {
		resultList.Launches[0].Result = vcsData[0].Namespaces
		if err := shared.PrintResultAsJSON(resultList); err != nil {
			logger.Error("error serializing JSON result", "error", err)
		}
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
	return fmt.Sprintf(`List repositories from a version control system.

List of avaliable vcs plugins:
  %s`, strings.Join(plugins, "\n  "))
}

func init() {
	ListCmd.Flags().StringVarP(&listOptions.VCSPluginName, "vcs", "p", "", "Name of the VCS plugin to use (e.g., bitbucket, gitlab, github).")
	ListCmd.Flags().StringVar(&listOptions.Domain, "domain", "", "Domain name of the VCS (e.g., github.com).")
	ListCmd.Flags().StringVar(&listOptions.Namespace, "namespace", "", "Name of the specific namespace, project, or organization.")
	ListCmd.Flags().StringVarP(&listOptions.OutputPath, "output", "o", "", "Path to the output file or directory where the list result will be saved.")
	ListCmd.Flags().StringVarP(&listOptions.Language, "language", "l", "", "Collect only projects that use the specified language (supported only for GitLab).")
	ListCmd.Flags().BoolP("help", "h", false, "Show help for the list command.")
}
