package createissuesfromsarif

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/owenrumney/go-sarif/v2/sarif"
	internalsarif "github.com/scan-io-git/scan-io/internal/sarif"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// RunOptions holds flags for the create-issues-from-sarif command.
type RunOptions struct {
	Namespace    string `json:"namespace,omitempty"`
	Repository   string `json:"repository,omitempty"`
	SarifPath    string `json:"sarif_path,omitempty"`
	SourceFolder string `json:"source_folder,omitempty"`
	Ref          string `json:"ref,omitempty"`
}

var (
	AppConfig *config.Config
	opts      RunOptions

	// CreateIssuesFromSarifCmd represents the command to create GitHub issues from a SARIF file.
	CreateIssuesFromSarifCmd = &cobra.Command{
		Use:                   "create-issues-from-sarif --namespace NAMESPACE --repository REPO --sarif PATH [--source-folder PATH] [--ref REF]",
		Short:                 "Create GitHub issues for high severity SARIF findings",
		Example:               "scanio create-issues-from-sarif --namespace org --repository repo --sarif /path/to/report.sarif",
		SilenceUsage:          true,
		Hidden:                true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !shared.HasFlags(cmd.Flags()) {
				return cmd.Help()
			}

			if err := validate(&opts); err != nil {
				return errors.NewCommandError(opts, nil, err, 1)
			}

			lg := logger.NewLogger(AppConfig, "create-issues-from-sarif")

			report, err := internalsarif.ReadReport(opts.SarifPath, lg, opts.SourceFolder, true)
			if err != nil {
				lg.Error("failed to read SARIF report", "error", err)
				return errors.NewCommandError(opts, nil, fmt.Errorf("failed to read SARIF report: %w", err), 2)
			}

			// Enrich to ensure Levels and Titles are present
			report.EnrichResultsLevelProperty()
			report.EnrichResultsTitleProperty()
			// No need to enrich locations here; we'll compute file path from URI directly

			created := 0
			// Iterate runs and results
			for _, run := range report.Runs {
				for _, res := range run.Results {
					// Only high severity: map to Level == "error"
					level, _ := res.Properties["Level"].(string)
					if strings.ToLower(level) != "error" {
						continue
					}

					// Basic fields
					ruleID := ""
					if res.RuleID != nil {
						ruleID = *res.RuleID
					}

					titleText := fmt.Sprintf("[SARIF][%s]", ruleID)

					// derive file path and region info from SARIF result
					fileURI := extractFileURIFromResult(res, opts.SourceFolder)
					line, endLine := extractRegionFromResult(res)
					// Normalize file path for title readability
					shortPath := filepath.ToSlash(fileURI)
					if shortPath == "" {
						shortPath = "<unknown>"
					}
					if line > 0 {
						if endLine > line {
							titleText = fmt.Sprintf("%s at %s:%d-%d", titleText, shortPath, line, endLine)
						} else {
							titleText = fmt.Sprintf("%s at %s:%d", titleText, shortPath, line)
						}
					} else {
						titleText = fmt.Sprintf("%s at %s", titleText, shortPath)
					}

					desc := getStringProp(res.Properties, "Description")
					if desc == "" && res.Message.Text != nil {
						desc = *res.Message.Text
					}

					// Optionally include a GitHub permalink if ref is provided
					// If EndLine is present, use a range anchor: #Lstart-Lend
					permalink := ""
					if opts.Ref != "" && shortPath != "<unknown>" && line > 0 {
						encodedPath := encodePathSegments(shortPath)
						if endLine > line {
							permalink = fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s#L%d-L%d", opts.Namespace, opts.Repository, opts.Ref, encodedPath, line, endLine)
						} else {
							permalink = fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s#L%d", opts.Namespace, opts.Repository, opts.Ref, encodedPath, line)
						}
					}

					// Include line or line range in the body
					lineInfo := fmt.Sprintf("Line: %d", line)
					if endLine > line {
						lineInfo = fmt.Sprintf("Lines: %d-%d", line, endLine)
					}

					// Compute SHA256 over the referenced snippet (single line or range)
					snippetHash := ""
					if shortPath != "<unknown>" && line > 0 && opts.SourceFolder != "" {
						absPath := filepath.Join(opts.SourceFolder, filepath.FromSlash(shortPath))
						if data, err := os.ReadFile(absPath); err == nil {
							lines := strings.Split(string(data), "\n")
							start := line
							end := line
							if endLine > line {
								end = endLine
							}
							// Validate bounds (1-based line numbers)
							if start >= 1 && start <= len(lines) {
								if end > len(lines) {
									end = len(lines)
								}
								if end >= start {
									snippet := strings.Join(lines[start-1:end], "\n")
									sum := sha256.Sum256([]byte(snippet))
									snippetHash = fmt.Sprintf("%x", sum[:])
								}
							}
						}
					}

					body := fmt.Sprintf("Severity: %s\nRule: %s\nFile: %s\n%s\n", strings.ToUpper(level), ruleID, shortPath, lineInfo)
					if permalink != "" {
						body += fmt.Sprintf("Permalink: %s\n", permalink)
					}
					if snippetHash != "" {
						body += fmt.Sprintf("Snippet SHA256: %s\n", snippetHash)
					}
					body += "\n" + desc

					// Build request for VCS plugin
					req := shared.VCSIssueCreationRequest{
						VCSRequestBase: shared.VCSRequestBase{
							RepoParam: shared.RepositoryParams{
								Namespace:  opts.Namespace,
								Repository: opts.Repository,
							},
							Action: "createIssue",
						},
						Title: titleText,
						Body:  body,
					}

					// Call plugin
					err := shared.WithPlugin(AppConfig, "plugin-vcs", shared.PluginTypeVCS, "github", func(raw interface{}) error {
						vcs, ok := raw.(shared.VCS)
						if !ok {
							return fmt.Errorf("invalid VCS plugin type")
						}
						_, err := vcs.CreateIssue(req)
						return err
					})
					if err != nil {
						lg.Error("failed to create issue via plugin", "error", err, "rule", ruleID, "file", shortPath, "line", line)
						return errors.NewCommandError(opts, nil, fmt.Errorf("create issue failed: %w", err), 2)
					}
					created++
				}
			}

			lg.Info("issues created from SARIF high severity findings", "count", created)
			fmt.Printf("Created %d issue(s) from SARIF high severity findings\n", created)
			return nil
		},
	}
)

// Init wires config into this command.
func Init(cfg *config.Config) { AppConfig = cfg }

func init() {
	CreateIssuesFromSarifCmd.Flags().StringVar(&opts.Namespace, "namespace", "", "GitHub org/user")
	CreateIssuesFromSarifCmd.Flags().StringVar(&opts.Repository, "repository", "", "Repository name")
	CreateIssuesFromSarifCmd.Flags().StringVar(&opts.SarifPath, "sarif", "", "Path to SARIF file")
	CreateIssuesFromSarifCmd.Flags().StringVar(&opts.SourceFolder, "source-folder", "", "Optional: source folder to improve file path resolution in SARIF (used for absolute paths)")
	CreateIssuesFromSarifCmd.Flags().StringVar(&opts.Ref, "ref", "", "Git ref (branch or commit SHA) to build a permalink to the vulnerable code")
	CreateIssuesFromSarifCmd.Flags().BoolP("help", "h", false, "Show help for create-issues-from-sarif command.")
}

func validate(o *RunOptions) error {
	if o.Namespace == "" {
		return fmt.Errorf("--namespace is required")
	}
	if o.Repository == "" {
		return fmt.Errorf("--repository is required")
	}
	if strings.TrimSpace(o.SarifPath) == "" {
		return fmt.Errorf("--sarif is required")
	}
	return nil
}

// helper to fetch a string property safely
func getStringProp(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// encodePathSegments safely encodes each path segment without encoding slashes
func encodePathSegments(p string) string {
	if p == "" {
		return ""
	}
	parts := strings.Split(p, "/")
	for i, seg := range parts {
		parts[i] = url.PathEscape(seg)
	}
	return strings.Join(parts, "/")
}

// extractLocationInfo derives a file path (relative when appropriate), start line and end line
// from a SARIF result's first location. It mirrors the previous inline logic used in the
// command handler. Returns (fileURI, startLine, endLine).
// extractFileURIFromResult returns a file path derived from the SARIF result's first location.
// If the URI is absolute and a non-empty sourceFolder is provided, the returned path will be
// made relative to sourceFolder (matching previous behaviour).
func extractFileURIFromResult(res *sarif.Result, sourceFolder string) string {
	if res == nil || len(res.Locations) == 0 {
		return ""
	}
	loc := res.Locations[0]
	if loc.PhysicalLocation == nil {
		return ""
	}
	art := loc.PhysicalLocation.ArtifactLocation
	if art == nil || art.URI == nil {
		return ""
	}
	uri := *art.URI
	// If URI is not absolute or there's no sourceFolder provided, return it unchanged.
	if !filepath.IsAbs(uri) || sourceFolder == "" {
		return uri
	}

	// Normalize sourceFolder to an absolute, cleaned path so relative inputs like
	// "../scanio-test" match absolute URIs such as "/home/jekos/.../scanio-test/...".
	if absSource, err := filepath.Abs(sourceFolder); err == nil {
		absSource = filepath.Clean(absSource)

		// Prefer filepath.Rel which will produce a relative path when uri is under absSource.
		if rel, err := filepath.Rel(absSource, uri); err == nil {
			// If rel does not escape to parent directories, it's a proper subpath.
			if rel != "" && !strings.HasPrefix(rel, "..") {
				return rel
			}
		}

		// Fallback: trim the absolute source prefix explicitly when possible.
		prefix := absSource + string(filepath.Separator)
		if strings.HasPrefix(uri, prefix) {
			return strings.TrimPrefix(uri, prefix)
		}
		if strings.HasPrefix(uri, absSource) {
			return strings.TrimPrefix(uri, absSource)
		}
	}

	// Last-resort: try trimming the raw sourceFolder string provided by the user.
	rel := strings.TrimPrefix(uri, sourceFolder)
	if strings.HasPrefix(rel, string(filepath.Separator)) {
		return rel[1:]
	}
	return rel
}

// extractRegionFromResult returns start and end line numbers (0 when not present)
// taken from the SARIF result's first location region.
func extractRegionFromResult(res *sarif.Result) (int, int) {
	if res == nil || len(res.Locations) == 0 {
		return 0, 0
	}
	loc := res.Locations[0]
	if loc.PhysicalLocation == nil || loc.PhysicalLocation.Region == nil {
		return 0, 0
	}
	start := 0
	end := 0
	if loc.PhysicalLocation.Region.StartLine != nil {
		start = *loc.PhysicalLocation.Region.StartLine
	}
	if loc.PhysicalLocation.Region.EndLine != nil {
		end = *loc.PhysicalLocation.Region.EndLine
	}
	return start, end
}

// getRuleFullDescription returns the human-readable description for a rule from the run's rules table.
// It prefers rule.FullDescription.Text, falls back to rule.ShortDescription.Text, otherwise empty string.
func getRuleFullDescription(run *sarif.Run, ruleID string) string {
	if run == nil || run.Tool.Driver == nil {
		return ""
	}
	for _, rule := range run.Tool.Driver.Rules {
		if rule == nil {
			continue
		}
		if rule.ID == ruleID {
			if rule.FullDescription != nil && rule.FullDescription.Text != nil && *rule.FullDescription.Text != "" {
				return *rule.FullDescription.Text
			}
			return ""
		}
	}
	return ""
}
