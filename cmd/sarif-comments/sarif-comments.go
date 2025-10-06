package sarifcomments

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/ci"
	"github.com/scan-io-git/scan-io/internal/vcsintegrator"
	"github.com/scan-io-git/scan-io/pkg/shared"
	// "github.com/scan-io-git/scan-io/pkg/shared/artifacts"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"

	cmdutil "github.com/scan-io-git/scan-io/internal/cmd"
)

// RunOptionsSarifComments holds flags for sarif-comments command.
type RunOptionsSarifComments struct {
	VCS           string `json:"vcs,omitempty"`
	Domain        string `json:"domain,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	Repository    string `json:"repository,omitempty"`
	PullRequestID string `json:"pull_request_id,omitempty"`
	SarifPath     string `json:"sarif_path,omitempty"`
	SourceFolder  string `json:"source_folder,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

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

	resolution, err := ci.ResolveFromEnvironment(logger, sarifCommentsOptions.VCSPluginName)
	if err != nil {
		return errors.NewCommandError(sarifCommentsOptions, nil, err, 1)
	}

	if resolution.PluginName != "" {
		sarifCommentsOptions.VCSPluginName = resolution.PluginName
	}
	if sarifCommentsOptions.Domain == "" && resolution.Domain != "" {
		sarifCommentsOptions.Domain = resolution.Domain
	}
	if sarifCommentsOptions.Namespace == "" && resolution.Namespace != "" {
		sarifCommentsOptions.Namespace = resolution.Namespace
	}
	if sarifCommentsOptions.Repository == "" && resolution.Repository != "" {
		sarifCommentsOptions.Repository = resolution.Repository
	}
	if sarifCommentsOptions.PullRequestID == "" && resolution.PullRequest != "" {
		sarifCommentsOptions.PullRequestID = resolution.PullRequest
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

	// repo.HTTPLink = buildRepoHTTPLink(sarifCommentsOptions)

	sarifCommentsRequest, err := i.PrepIntegrationRequest(AppConfig, &sarifCommentsOptions, repoParams)
	if err != nil {
		logger.Error("failed to prepare integration VCS request", "error", err)
		return errors.NewCommandError(sarifCommentsOptions, nil, fmt.Errorf("failed to prepare integration VCS request: %w", err), 1)
	}

	resultSarifCommentsVCS, _ := i.IntegrationAction(AppConfig, sarifCommentsRequest)
	fmt.Print(resultSarifCommentsVCS)

	return nil
}

func buildRepoHTTPLink(opts RunOptionsSarifComments) string {
	if opts.Domain == "" || opts.Namespace == "" || opts.Repository == "" {
		return ""
	}
	return fmt.Sprintf("https://%s/%s/%s", opts.Domain, opts.Namespace, opts.Repository)
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
	// SarifCommentsCmd.Flags().StringVar(&sarifCommentsOptions.Ref, "ref", "", "Git ref (branch or commit SHA) to build a permalink to the vulnerable code")
	SarifCommentsCmd.Flags().BoolP("help", "h", false, "Show help for sarif-comments command.")
}
