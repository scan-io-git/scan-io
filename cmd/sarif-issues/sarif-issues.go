package sarifissues

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/git"
	internalsarif "github.com/scan-io-git/scan-io/internal/sarif"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// scanioManagedAnnotation is appended to issue bodies created by this command
// and is required for correlation/auto-closure to consider an issue
// managed by automation.
const (
	scanioManagedAnnotation = "> [!NOTE]\n> This issue was created and will be managed by scanio automation. Don't change body manually for proper processing, unless you know what you do"
	semgrepPromoFooter      = "#### 💎 Enable cross-file analysis and Pro rules for free at <a href='https://sg.run/pro'>sg.run/pro</a>\n\n"
)

// RunOptions holds flags for the sarif-issues command.
type RunOptions struct {
	Namespace    string   `json:"namespace,omitempty"`
	Repository   string   `json:"repository,omitempty"`
	SarifPath    string   `json:"sarif_path,omitempty"`
	SourceFolder string   `json:"source_folder,omitempty"`
	Ref          string   `json:"ref,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	Assignees    []string `json:"assignees,omitempty"`
}

var (
	AppConfig *config.Config
	opts      RunOptions

	// Example usage for the sarif-issues command
	exampleSarifIssuesUsage = `  # Recommended: run from repository root and use relative paths
  scanio sarif-issues --namespace scan-io-git --repository scan-io --sarif /path/to/report.sarif --source-folder apps/demo

  # Run inside git repository (auto-detects namespace, repository, ref)
  scanio sarif-issues --sarif semgrep-demo.sarif --source-folder apps/demo

  # Create issues from SARIF report with basic configuration
  scanio sarif-issues --namespace scan-io-git --repository scan-io --sarif /path/to/report.sarif

  # Create issues with labels and assignees
  scanio sarif-issues --namespace scan-io-git --repository scan-io --sarif /path/to/report.sarif --labels bug,security --assignees alice,bob

  # Create issues with source folder for better file path resolution
  scanio sarif-issues --namespace scan-io-git --repository scan-io --sarif /path/to/report.sarif --source-folder /path/to/source

  # Create issues with specific git reference for permalinks
  scanio sarif-issues --namespace scan-io-git --repository scan-io --sarif /path/to/report.sarif --ref feature-branch

  # Using environment variables (GitHub Actions)
  GITHUB_REPOSITORY_OWNER=scan-io-git GITHUB_REPOSITORY=scan-io-git/scan-io GITHUB_SHA=abc123 scanio sarif-issues --sarif /path/to/report.sarif`

	// SarifIssuesCmd represents the command to create GitHub issues from a SARIF file.
	SarifIssuesCmd = &cobra.Command{
		Use:                   "sarif-issues --sarif PATH [--namespace NAMESPACE] [--repository REPO] [--source-folder PATH] [--ref REF] [--labels label[,label...]] [--assignees user[,user...]]",
		Short:                 "Create GitHub issues for high severity SARIF findings",
		Example:               exampleSarifIssuesUsage,
		SilenceUsage:          false,
		Hidden:                false,
		DisableFlagsInUseLine: true,
		RunE:                  runSarifIssues,
	}
)

// Init wires config into this command.
func Init(cfg *config.Config) {
	AppConfig = cfg
}

// runSarifIssues is the main execution function for the sarif-issues command.
func runSarifIssues(cmd *cobra.Command, args []string) error {
	// 1. Check for help request
	if len(args) == 0 && !shared.HasFlags(cmd.Flags()) {
		return cmd.Help()
	}

	// 2. Initialize logger
	lg := logger.NewLogger(AppConfig, "sarif-issues")

	// 3. Handle environment variable fallbacks
	ApplyEnvironmentFallbacks(&opts)

	// 4. Handle git metadata fallbacks
	ApplyGitMetadataFallbacks(&opts, lg)

	// 5. Validate arguments
	if err := validate(&opts); err != nil {
		lg.Error("invalid arguments", "error", err)
		return errors.NewCommandError(opts, nil, fmt.Errorf("invalid arguments: %w", err), 1)
	}

	// 6. Read and process SARIF report
	report, err := internalsarif.ReadReport(opts.SarifPath, lg, opts.SourceFolder, true)
	if err != nil {
		lg.Error("failed to read SARIF report", "error", err)
		return errors.NewCommandError(opts, nil, fmt.Errorf("failed to read SARIF report: %w", err), 2)
	}

	// Resolve source folder to absolute form for path calculations
	sourceFolderAbs := ResolveSourceFolder(opts.SourceFolder, lg)

	// Collect repository metadata to understand repo root vs. subfolder layout
	repoMetadata := resolveRepositoryMetadata(sourceFolderAbs, lg)

	// Enrich to ensure Levels and Titles are present
	report.EnrichResultsLevelProperty()
	report.EnrichResultsTitleProperty()

	// 7. Get all open GitHub issues
	openIssues, err := listOpenIssues(opts)
	if err != nil {
		lg.Error("failed to list open issues", "error", err)
		return errors.NewCommandError(opts, nil, fmt.Errorf("failed to list open issues: %w", err), 2)
	}
	lg.Info("fetched open issues from repository", "count", len(openIssues))

	// 8. Process SARIF report and create/close issues
	created, err := processSARIFReport(report, opts, sourceFolderAbs, repoMetadata, lg, openIssues)
	if err != nil {
		lg.Error("failed to process SARIF report", "error", err)
		return err
	}

	// 9. Log success and handle output
	lg.Info("issues created from SARIF high severity findings", "count", created)
	fmt.Printf("Created %d issue(s) from SARIF high severity findings\n", created)

	return nil
}

func init() {
	SarifIssuesCmd.Flags().StringVar(&opts.Namespace, "namespace", "", "GitHub org/user (defaults to $GITHUB_REPOSITORY_OWNER when unset)")
	SarifIssuesCmd.Flags().StringVar(&opts.Repository, "repository", "", "Repository name (defaults to ${GITHUB_REPOSITORY#*/} when unset)")
	SarifIssuesCmd.Flags().StringVar(&opts.SarifPath, "sarif", "", "Path to SARIF file")
	SarifIssuesCmd.Flags().StringVar(&opts.SourceFolder, "source-folder", "", "Optional: source folder to improve file path resolution in SARIF (used for absolute paths)")
	SarifIssuesCmd.Flags().StringVar(&opts.Ref, "ref", "", "Git ref (branch or commit SHA) to build a permalink to the vulnerable code (defaults to $GITHUB_SHA when unset)")
	// --labels supports multiple usages (e.g., --labels bug --labels security) or comma-separated values
	SarifIssuesCmd.Flags().StringSliceVar(&opts.Labels, "labels", nil, "Optional: labels to assign to created GitHub issues (repeat flag or use comma-separated values)")
	// --assignees supports multiple usages or comma-separated values
	SarifIssuesCmd.Flags().StringSliceVar(&opts.Assignees, "assignees", nil, "Optional: assignees (GitHub logins) to assign to created issues (repeat flag or use comma-separated values)")
	SarifIssuesCmd.Flags().BoolP("help", "h", false, "Show help for sarif-issues command.")
}

func resolveRepositoryMetadata(sourceFolderAbs string, lg hclog.Logger) *git.RepositoryMetadata {
	if strings.TrimSpace(sourceFolderAbs) == "" {
		return nil
	}

	md, err := git.CollectRepositoryMetadata(sourceFolderAbs)
	if err != nil {
		lg.Debug("unable to collect repository metadata", "error", err)
	}
	return md
}
