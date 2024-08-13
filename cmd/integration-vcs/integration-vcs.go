package integrationvcs

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/vcsintegrator"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

type Arguments interface{}

// Global variables for configuration and command arguments
var (
	AppConfig                  *config.Config
	integrationVCSOptions      vcsintegrator.RunOptionsIntegrationVCS
	exampleIntegrationVCSUsage = `# Check the existence of a PR
  scanio integration-vcs --vcs github --action checkPR --domain github.com --namespace scan-io-git --repository scan-io --pull-request-id 1

  # Check the existence of a PR using URL
  scanio integration-vcs --vcs github --action checkPR https://github.com/scan-io-git/scan-io/pull/1

  # Add a user to the PR
  scanio integration-vcs --vcs github --action addRoleToPR --domain github.com --namespace scan-io-git --repository scan-io --pull-request-id 1 --login scanio-bot --role REVIEWER

  # Add a user to the PR using URL
  scanio integration-vcs --vcs github --action addRoleToPR --login scanio-bot --role REVIEWER https://github.com/scan-io-git/scan-io/pull/1

  # Change the status of the PR
  scanio integration-vcs --vcs github --action setStatusOfPR --domain github.com --namespace scan-io-git --repository scan-io --pull-request-id 1 --login scanio-bot --status UNAPPROVED

  # Change the status of the PR using URL
  scanio integration-vcs --vcs github --action setStatusOfPR --login scanio-bot --status UNAPPROVED https://github.com/scan-io-git/scan-io/pull/1

  # Leave a comment on the PR with text directly
  scanio integration-vcs --vcs github --action addComment --domain github.com --namespace scan-io-git --repository scan-io --pull-request-id 1 --comment "Hello username"

  # Leave a comment on the PR using URL with text directly
  scanio integration-vcs --vcs github --action addComment --comment "Hello username" https://github.com/scan-io-git/scan-io/pull/1

  # Leave a comment on the PR with text from a file
  scanio integration-vcs --vcs github --action addComment --domain github.com --namespace scan-io-git --repository scan-io --pull-request-id 1 --comment-file /path/to/comment.txt

  # Leave a comment on the PR using URL with text from a file
  scanio integration-vcs --vcs github --action addComment --comment-file /path/to/comment.txt https://github.com/scan-io-git/scan-io/pull/1

  # Leave a comment on the PR and attach files
  scanio integration-vcs --vcs github --action addComment --domain github.com --namespace scan-io-git --repository scan-io --pull-request-id 1 --comment "See attached files" --files /path/to/file1.txt,/path/to/file2.txt
  
  # Leave a comment on the PR using URL and attach files
  scanio integration-vcs --vcs github --action addComment --comment "See attached files" --files /path/to/file1.txt,/path/to/file2.txt https://github.com/scan-io-git/scan-io/pull/1`
)

// IntegrationVCSCmd represents the command for VCS integrations.
var IntegrationVCSCmd = &cobra.Command{
	Use:                   "integration-vcs --vcs/-p PLUGIN_NAME [--login LOGIN --role ROLE --status STATUS --comment COMMENT --comment-file COMMENT_FILE --files FILES] {--domain VCS_DOMAIN_NAME --namespace NAMESPACE --repository REPOSITORY --pull-request-id ID | URL}",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example:               exampleIntegrationVCSUsage,
	Short:                 "Execute VCS integrations for different actions",
	RunE:                  runIntegrationVCSCommand,
}

// Init initializes the global configuration variable and sets the long description for the IntegrationVCSCmd command.
func Init(cfg *config.Config) {
	AppConfig = cfg
	IntegrationVCSCmd.Long = generateLongDescription(AppConfig)
}

// runIntegrationVCSCommand runs the integration VCS command with the provided arguments and options.
func runIntegrationVCSCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 && !shared.HasFlags(cmd.Flags()) {
		return cmd.Help()
	}

	logger := logger.NewLogger(AppConfig, "core-integration-vcs")

	if err := validateIntegrationVCSArgs(&integrationVCSOptions, args); err != nil {
		logger.Error("invalid integration VCS arguments", "error", err)
		return err
	}

	mode := determineMode(args)
	i := vcsintegrator.New(
		integrationVCSOptions.VCSPluginName,
		integrationVCSOptions.Action,
		logger,
	)
	repoParams, err := prepareIntegrationVCSTarget(&integrationVCSOptions, args, mode)
	if err != nil {
		logger.Error("failed to prepare integration VCS targets", "error", err)
		return err
	}

	integrationVCSRequest, err := i.PrepIntegrationRequest(AppConfig, &integrationVCSOptions, repoParams)
	if err != nil {
		logger.Error("failed to prepare integration VCS request", "error", err)
		return err
	}

	resultIntegrationVCS, integrationVCSErr := i.IntegrationAction(AppConfig, integrationVCSRequest)
	metaDataFileName := fmt.Sprintf("VCS-INTEGRATION_%s_%s",
		strings.ToUpper(i.PluginName),
		strings.ToUpper(integrationVCSOptions.Action))
	if config.IsCI(AppConfig) {
		startTime := time.Now().UTC().Format(time.RFC3339)
		metaDataFileName = fmt.Sprintf("VCS-INTEGRATION_%s_%s_%v",
			strings.ToUpper(i.PluginName),
			strings.ToUpper(integrationVCSOptions.Action),
			startTime)
	}

	if err := shared.WriteGenericResult(AppConfig, logger, resultIntegrationVCS, metaDataFileName); err != nil {
		logger.Error("failed to write result", "error", err)
		return err
	}

	if integrationVCSErr != nil {
		logger.Error("integration-vcs command failed", "error", integrationVCSErr)
		return integrationVCSErr
	}

	logger.Debug("integration-vcs result", "result", resultIntegrationVCS)
	logger.Info("integration-vcs command completed successfully")
	return nil
}

// generateLongDescription generates the long description dynamically with the list of available VCS plugins and actions.
func generateLongDescription(cfg *config.Config) string {
	pluginsMeta := shared.GetPluginVersions(config.GetScanioPluginsHome(cfg), "vcs")
	var plugins []string
	for plugin := range pluginsMeta {
		plugins = append(plugins, plugin)
	}

	actions := []string{
		vcsintegrator.VCSCheckPR,
		vcsintegrator.VCSCommentPR,
		vcsintegrator.VCSAddRoleToPR,
		vcsintegrator.VCSSetStatusOfPR,
	}
	return fmt.Sprintf(`Execute VCS integrations for different actions.

List of actions:
  %s
  
List of available VCS plugins:
  %s`, strings.Join(actions, "\n  "), strings.Join(plugins, "\n  "))
}

func init() {
	IntegrationVCSCmd.Flags().StringVarP(&integrationVCSOptions.VCSPluginName, "vcs", "p", "", "Name of the VCS plugin to use (e.g., bitbucket, gitlab, github).")
	IntegrationVCSCmd.Flags().StringVar(&integrationVCSOptions.Domain, "domain", "", "Domain name of the VCS (e.g., github.com).")
	IntegrationVCSCmd.Flags().StringVar(&integrationVCSOptions.Namespace, "namespace", "", "Name of the specific namespace, project, or organization.")
	IntegrationVCSCmd.Flags().StringVar(&integrationVCSOptions.Repository, "repository", "", "Name of a specific repository.")
	IntegrationVCSCmd.Flags().StringVar(&integrationVCSOptions.PullRequestID, "pull-request-id", "", "ID of a specific pull request from the repository.")
	IntegrationVCSCmd.Flags().StringVar(&integrationVCSOptions.Action, "action", "", "Action to execute (e.g., checkPR, addComment, addRoleToPR, setStatusOfPR).")
	IntegrationVCSCmd.Flags().StringVar(&integrationVCSOptions.Login, "login", "", "Login for integrations, e.g., add a reviewer with this login to a PR.")
	IntegrationVCSCmd.Flags().StringVar(&integrationVCSOptions.Role, "role", "", "Role for integrations, e.g., add a person with a specific role to a PR.")
	IntegrationVCSCmd.Flags().StringVar(&integrationVCSOptions.Status, "status", "", "Status for integrations, e.g., set the status of a PR.")
	IntegrationVCSCmd.Flags().StringVar(&integrationVCSOptions.Comment, "comment", "", "Comment text to be added to the pull request.")
	IntegrationVCSCmd.Flags().StringVar(&integrationVCSOptions.CommentFile, "comment-file", "", "File containing the comment text to be added to the pull request.")
	IntegrationVCSCmd.Flags().StringSliceVar(&integrationVCSOptions.AttachFiles, "files", nil, "Comma-separated list of paths to files to be uploaded and attached to the comment.")
	IntegrationVCSCmd.Flags().BoolP("help", "h", false, "Show help for the integration command.")
}
