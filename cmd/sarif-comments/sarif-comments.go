package sarifcomments

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/ci"
	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/internal/vcsintegrator"
	"github.com/scan-io-git/scan-io/pkg/shared"
	// "github.com/scan-io-git/scan-io/pkg/shared/artifacts"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"

	cmdutil "github.com/scan-io-git/scan-io/internal/cmd"
)

// Global variables for configuration and command arguments
var (
	AppConfig            *config.Config
	logger               hclog.Logger
	sarifCommentsOptions vcsintegrator.RunOptionsIntegrationVCS

	exampleSarifCommentsUsage = `  # Post inline comments from SARIF findings to a pull request
  scanio sarif-comments --vcs bitbucket --domain bitbucket.org --namespace team --repository project --pull-request-id 42 --sarif /path/to/report.sarif

  # Limit number of comments and provide an overall summary
  scanio sarif-comments --vcs bitbucket --domain bitbucket.org --namespace team --repository project --pull-request-id 42 --sarif report.sarif --limit 10 --summary "See findings below"`

	SarifCommentsCmd = &cobra.Command{
		Use:                   "sarif-comments --vcs NAME --domain DOMAIN --namespace NAMESPACE --repository REPO --pull-request-id ID --sarif PATH --source-folder PATH [--limit N] [--summary TEXT]",
		Short:                 "Post SARIF findings as PR comments using a VCS plugin",
		Example:               exampleSarifCommentsUsage,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		RunE:                  runSarifComments,
	}
)

// Init wires config and logger into the command package.
func Init(cfg *config.Config, l hclog.Logger) {
	AppConfig = cfg
	logger = l
	SarifCommentsCmd.Long = generateLongDescription(AppConfig)
}

func runSarifComments(cmd *cobra.Command, args []string) error {
	if len(args) == 0 && !shared.HasFlags(cmd.Flags()) {
		return cmd.Help()
	}

	mode := cmdutil.DetermineMode(args)

	resolutionEnv, err := ci.ResolveFromEnvironment(logger, sarifCommentsOptions.VCSPluginName)
	if err != nil {
		return errors.NewCommandError(sarifCommentsOptions, nil, err, 1)
	}

	resolutionGitMeta, err := git.ApplyGitMetadataOptionsFallbacks(logger, sarifCommentsOptions.SourceFolder,
		sarifCommentsOptions.Namespace, sarifCommentsOptions.Repository, sarifCommentsOptions.VCSPluginName, "")
	if err != nil {
		logger.Debug("git metadata fallback failed", "error", err)
	}

	if resolutionEnv.PluginName != "" {
		sarifCommentsOptions.VCSPluginName = resolutionEnv.PluginName
	}
	if sarifCommentsOptions.Domain == "" && resolutionEnv.Domain != "" {
		sarifCommentsOptions.Domain = resolutionEnv.Domain
	}
	if sarifCommentsOptions.Namespace == "" && resolutionEnv.Namespace != "" {
		sarifCommentsOptions.Namespace = resolutionEnv.Namespace
	} else if sarifCommentsOptions.Namespace == "" && resolutionGitMeta.Namespace != "" {
		sarifCommentsOptions.Namespace = resolutionGitMeta.Namespace
	}
	if sarifCommentsOptions.Repository == "" && resolutionEnv.Repository != "" {
		sarifCommentsOptions.Repository = resolutionEnv.Repository
	} else if sarifCommentsOptions.Repository == "" && resolutionGitMeta.Repository != "" {
		sarifCommentsOptions.Repository = resolutionGitMeta.Repository
	}
	if sarifCommentsOptions.PullRequestID == "" && resolutionEnv.PullRequest != "" {
		sarifCommentsOptions.PullRequestID = resolutionEnv.PullRequest
	}

	if err := validateSarifCommentsArgs(&sarifCommentsOptions, args, mode); err != nil {
		logger.Error("invalid command arguments", "error", err)
		return errors.NewCommandError(sarifCommentsOptions, nil, fmt.Errorf("invalid arguments: %w", err), 1)
	}

	sarifCommentsOptions.Action = vcsintegrator.VCSAddInLineCommentsSarif
	i := vcsintegrator.New(
		sarifCommentsOptions.VCSPluginName,
		sarifCommentsOptions.Action,
		logger,
	)

	repoParams, err := prepareSarifCommentsTarget(&sarifCommentsOptions, args, mode)
	if err != nil {
		logger.Error("failed to prepare sarif comments targets", "error", err)
		return errors.NewCommandError(sarifCommentsOptions, nil, fmt.Errorf("failed to prepare sarif comments targets: %w", err), 1)
	}

	sarifCommentsRequest, err := i.PrepIntegrationRequest(AppConfig, &sarifCommentsOptions, repoParams)
	if err != nil {
		logger.Error("failed to prepare integration VCS request", "error", err)
		return errors.NewCommandError(sarifCommentsOptions, nil, fmt.Errorf("failed to prepare integration VCS request: %w", err), 1)
	}

	resultSarifCommentsVCS, _ := i.IntegrationAction(AppConfig, sarifCommentsRequest)
	fmt.Print(resultSarifCommentsVCS)

	return nil
}

// generateLongDescription generates the long description dynamically with the list of available scanner plugins.
func generateLongDescription(cfg *config.Config) string {
	pluginsMeta := shared.GetPluginVersions(config.GetScanioPluginsHome(cfg), "vcs")
	var plugins []string
	for plugin := range pluginsMeta {
		plugins = append(plugins, plugin)
	}
	return fmt.Sprintf(`Post SARIF findings as PR comments using a VCS plugin.

List of available scanner plugins:
  %s`, strings.Join(plugins, "\n  "))
}

func init() {
	SarifCommentsCmd.Flags().StringVarP(&sarifCommentsOptions.VCSPluginName, "vcs", "p", "", "Name of the VCS plugin (e.g., bitbucket, gitlab, github)")
	SarifCommentsCmd.Flags().StringVar(&sarifCommentsOptions.Domain, "domain", "", "Domain of the VCS instance (e.g., bitbucket.org)")
	SarifCommentsCmd.Flags().StringVar(&sarifCommentsOptions.Namespace, "namespace", "", "Namespace/organization that owns the repository")
	SarifCommentsCmd.Flags().StringVar(&sarifCommentsOptions.Repository, "repository", "", "Repository name")
	SarifCommentsCmd.Flags().StringVar(&sarifCommentsOptions.PullRequestID, "pull-request-id", "", "Pull request identifier")
	SarifCommentsCmd.Flags().StringVar(&sarifCommentsOptions.SarifInput, "input", "i", "Path to the SARIF report")
	SarifCommentsCmd.Flags().StringVar(&sarifCommentsOptions.SourceFolder, "source", "s", "Source folder used to resolve enrich SARIF report with mandatory data like a snippet hash")
	SarifCommentsCmd.Flags().IntVar(&sarifCommentsOptions.SarifIssuesLimit, "limit", 0, "Maximum number of SARIF findings to convert into comments (0 = no limit)")
	SarifCommentsCmd.Flags().StringVar(&sarifCommentsOptions.Comment, "comment", "", "Optional summary comment appended after posting inline comments")
	SarifCommentsCmd.Flags().StringVar(&sarifCommentsOptions.CommentFile, "comment-file", "", "File containing the summary comment appended after posting inline comments")
	SarifCommentsCmd.Flags().StringSliceVar(&sarifCommentsOptions.SarifLevels, "levels", []string{"error"}, "SARIF severity levels to process: SARIF levels (error, warning, note, none) or display levels (High, Medium, Low, Info). Cannot mix formats. (repeat flag or use comma-separated values)")
	// SarifCommentsCmd.Flags().StringVar(&sarifCommentsOptions.Ref, "ref", "", "Git ref (branch or commit SHA) to build a permalink to the vulnerable code")
	SarifCommentsCmd.Flags().BoolP("help", "h", false, "Show help for sarif-comments command.")
}
