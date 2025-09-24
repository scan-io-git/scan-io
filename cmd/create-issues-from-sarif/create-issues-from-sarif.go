package createissuesfromsarif

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/spf13/cobra"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/internal/git"
	internalsarif "github.com/scan-io-git/scan-io/internal/sarif"
	issuecorrelation "github.com/scan-io-git/scan-io/pkg/issuecorrelation"
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

// RunOptions holds flags for the create-issues-from-sarif command.
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

	// CreateIssuesFromSarifCmd represents the command to create GitHub issues from a SARIF file.
	CreateIssuesFromSarifCmd = &cobra.Command{
		Use:                   "create-issues-from-sarif --sarif PATH [--namespace NAMESPACE] [--repository REPO] [--source-folder PATH] [--ref REF] [--labels label[,label...]] [--assignees user[,user...]]",
		Short:                 "Create GitHub issues for high severity SARIF findings",
		Example:               "scanio create-issues-from-sarif --namespace scan-io-git --repository scan-io --sarif /path/to/report.sarif --labels bug,security --assignees alice,bob",
		SilenceUsage:          true,
		Hidden:                true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !shared.HasFlags(cmd.Flags()) {
				return cmd.Help()
			}

			// Fallback: if --namespace not provided, try $GITHUB_REPOSITORY_OWNER
			if strings.TrimSpace(opts.Namespace) == "" {
				if ns := strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY_OWNER")); ns != "" {
					opts.Namespace = ns
				}
			}

			// Fallback: if --repository not provided, try ${GITHUB_REPOSITORY#*/}
			if strings.TrimSpace(opts.Repository) == "" {
				if gr := strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY")); gr != "" {
					if idx := strings.Index(gr, "/"); idx >= 0 && idx < len(gr)-1 {
						opts.Repository = gr[idx+1:]
					} else {
						// No slash present; fall back to the whole value
						opts.Repository = gr
					}
				}
			}

			// Fallback: if --ref not provided, try $GITHUB_SHA
			if strings.TrimSpace(opts.Ref) == "" {
				if sha := strings.TrimSpace(os.Getenv("GITHUB_SHA")); sha != "" {
					opts.Ref = sha
				}
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

			// get all open github issues
			openIssues, err := listOpenIssues(opts)
			if err != nil {
				return err
			}
			lg.Info("fetched open issues from repository", "count", len(openIssues))

			created, err := processSARIFReport(report, opts, lg)
			if err != nil {
				return err
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
	CreateIssuesFromSarifCmd.Flags().StringVar(&opts.Namespace, "namespace", "", "GitHub org/user (defaults to $GITHUB_REPOSITORY_OWNER when unset)")
	CreateIssuesFromSarifCmd.Flags().StringVar(&opts.Repository, "repository", "", "Repository name (defaults to ${GITHUB_REPOSITORY#*/} when unset)")
	CreateIssuesFromSarifCmd.Flags().StringVar(&opts.SarifPath, "sarif", "", "Path to SARIF file")
	CreateIssuesFromSarifCmd.Flags().StringVar(&opts.SourceFolder, "source-folder", "", "Optional: source folder to improve file path resolution in SARIF (used for absolute paths)")
	CreateIssuesFromSarifCmd.Flags().StringVar(&opts.Ref, "ref", "", "Git ref (branch or commit SHA) to build a permalink to the vulnerable code (defaults to $GITHUB_SHA when unset)")
	// --labels supports multiple usages (e.g., --labels bug --labels security) or comma-separated values
	CreateIssuesFromSarifCmd.Flags().StringSliceVar(&opts.Labels, "labels", nil, "Optional: labels to assign to created GitHub issues (repeat flag or use comma-separated values)")
	// --assignees supports multiple usages or comma-separated values
	CreateIssuesFromSarifCmd.Flags().StringSliceVar(&opts.Assignees, "assignees", nil, "Optional: assignees (GitHub logins) to assign to created issues (repeat flag or use comma-separated values)")
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

// displaySeverity normalizes SARIF severity levels to more descriptive labels.
func displaySeverity(level string) string {
	normalized := strings.ToLower(strings.TrimSpace(level))
	switch normalized {
	case "error":
		return "High"
	case "warning":
		return "Medium"
	case "note":
		return "Low"
	case "none":
		return "Info"
	default:
		if normalized == "" {
			return ""
		}
		return cases.Title(language.Und).String(normalized)
	}
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

// buildIssueTitle creates a concise issue title using scanner name (fallback to SARIF),
// severity, ruleID and location info. It formats as "[<scanner>][<severity>][<ruleID>] at"
// and includes a range when endLine > line.
func buildIssueTitle(scannerName, severity, ruleID, fileURI string, line, endLine int) string {
	label := strings.TrimSpace(scannerName)
	if label == "" {
		label = "SARIF"
	}
	sev := strings.TrimSpace(severity)
	parts := []string{label}
	if sev != "" {
		parts = append(parts, sev)
	}
	parts = append(parts, ruleID)
	title := fmt.Sprintf("[%s]", strings.Join(parts, "]["))
	if line > 0 {
		if endLine > line {
			return fmt.Sprintf("%s at %s:%d-%d", title, fileURI, line, endLine)
		}
		return fmt.Sprintf("%s at %s:%d", title, fileURI, line)
	}
	return fmt.Sprintf("%s at %s", title, fileURI)
}

// computeSnippetHash reads the snippet (single line or range) from sourceFolder + fileURI
// and returns its SHA256 hex string. Returns empty string on any error or if inputs are invalid.
func computeSnippetHash(fileURI string, line, endLine int, sourceFolder string) string {
	if fileURI == "" || fileURI == "<unknown>" || line <= 0 || sourceFolder == "" {
		return ""
	}
	absPath := filepath.Join(sourceFolder, filepath.FromSlash(fileURI))
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	start := line
	end := line
	if endLine > line {
		end = endLine
	}
	// Validate bounds (1-based line numbers)
	if start < 1 || start > len(lines) {
		return ""
	}
	if end > len(lines) {
		end = len(lines)
	}
	if end < start {
		return ""
	}
	snippet := strings.Join(lines[start-1:end], "\n")
	sum := sha256.Sum256([]byte(snippet))
	return fmt.Sprintf("%x", sum[:])
}

// parseRuleHelpMarkdown removes promotional content from help markdown and splits
// it into the descriptive details and a list of reference bullet points.
func parseRuleHelpMarkdown(markdown string) (string, []string) {
	cleaned := strings.ReplaceAll(markdown, semgrepPromoFooter, "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "", nil
	}

	lines := strings.Split(cleaned, "\n")
	referencesStart := -1
	for idx, raw := range lines {
		if strings.TrimSpace(raw) == "<b>References:</b>" {
			referencesStart = idx
			break
		}
	}

	if referencesStart == -1 {
		return cleaned, nil
	}

	detail := strings.TrimSpace(strings.Join(lines[:referencesStart], "\n"))
	var references []string
	for _, raw := range lines[referencesStart+1:] {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		// Normalise to Markdown bullet points regardless of the original marker.
		trimmed = strings.TrimLeft(trimmed, "-* \t")
		if trimmed == "" {
			continue
		}
		references = append(references, "- "+trimmed)
	}

	return detail, references
}

// getScannerName returns the tool/driver name for a SARIF run when available.
func getScannerName(run *sarif.Run) string {
	if run == nil {
		return ""
	}
	if run.Tool.Driver == nil {
		return ""
	}
	if run.Tool.Driver.Name != "" {
		return run.Tool.Driver.Name
	}
	return ""
}

// buildGitHubPermalink builds a permalink to a file and region in GitHub.
// It prefers the CLI --ref when provided; otherwise attempts to read the
// current commit hash from --source-folder using git metadata. When neither
// is available, returns an empty string.
func buildGitHubPermalink(options RunOptions, fileURI string, start, end int) string {
	base := fmt.Sprintf("https://github.com/%s/%s", options.Namespace, options.Repository)
	ref := strings.TrimSpace(options.Ref)

	if ref == "" && options.SourceFolder != "" {
		if md, err := git.CollectRepositoryMetadata(options.SourceFolder); err == nil && md.CommitHash != nil && *md.CommitHash != "" {
			ref = *md.CommitHash
		}
	}

	if ref == "" || fileURI == "" || fileURI == "<unknown>" {
		return ""
	}

	path := filepath.ToSlash(fileURI)
	anchor := ""
	if start > 0 {
		anchor = fmt.Sprintf("#L%d", start)
		if end > start {
			anchor = fmt.Sprintf("%s-L%d", anchor, end)
		}
	}

	return fmt.Sprintf("%s/blob/%s/%s%s", base, ref, path, anchor)
}

// OpenIssueReport represents parsed metadata from an open issue body.
type OpenIssueReport struct {
	Severity    string
	Scanner     string
	RuleID      string
	FilePath    string
	StartLine   int
	EndLine     int
	Hash        string
	Permalink   string
	Description string
}

// OpenIssueEntry combines parsed metadata from an open issue body with the
// original IssueParams returned by the VCS plugin. The map returned by
// listOpenIssues uses the issue number as key and this struct as value.
type OpenIssueEntry struct {
	OpenIssueReport
	Params shared.IssueParams
}

// parseIssueBody attempts to read the body produced by this command and extract
// known metadata lines (Severity, Scanner, File, Line(s), Snippet SHA256, Description).
// Returns an OpenIssueReport with zero values when fields are missing.
func parseIssueBody(body string) OpenIssueReport {
	rep := OpenIssueReport{}
	// Prefer new-style rule ID header first; fallback to legacy "Rule:" line if absent.
	if rid := extractRuleIDFromBody(body); rid != "" {
		rep.RuleID = rid
	}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		// Support new blockquote compact metadata lines
		// "> **Severity**: Error,  **Scanner**: Semgrep OSS"
		// "> **File**: app.py, **Lines**: 11-29"
		if strings.HasPrefix(line, "> ") {
			// Remove "> " prefix for easier parsing
			l := strings.TrimSpace(strings.TrimPrefix(line, "> "))
			// Normalize bold markers to plain keys
			l = strings.ReplaceAll(l, "**", "")
			// Split into comma-separated parts first
			parts := strings.Split(l, ",")
			for _, p := range parts {
				seg := strings.TrimSpace(p)
				if strings.HasPrefix(seg, "Severity:") {
					rep.Severity = strings.TrimSpace(strings.TrimPrefix(seg, "Severity:"))
					continue
				}
				if strings.HasPrefix(seg, "Scanner:") {
					rep.Scanner = strings.TrimSpace(strings.TrimPrefix(seg, "Scanner:"))
					continue
				}
				if strings.HasPrefix(seg, "File:") {
					// If File appears on the first line with comma, capture
					v := strings.TrimSpace(strings.TrimPrefix(seg, "File:"))
					if v != "" {
						rep.FilePath = v
					}
					continue
				}
				if strings.HasPrefix(seg, "Lines:") {
					v := strings.TrimSpace(strings.TrimPrefix(seg, "Lines:"))
					if strings.Contains(v, "-") {
						lr := strings.SplitN(v, "-", 2)
						if len(lr) == 2 {
							if s, err := strconv.Atoi(strings.TrimSpace(lr[0])); err == nil {
								rep.StartLine = s
							}
							if e, err := strconv.Atoi(strings.TrimSpace(lr[1])); err == nil {
								rep.EndLine = e
							}
						}
					} else {
						if n, err := strconv.Atoi(v); err == nil {
							rep.StartLine = n
							rep.EndLine = n
						}
					}
					continue
				}
				// Support snippet hash in blockquoted metadata line at end of issue
				if strings.HasPrefix(seg, "Snippet SHA256:") {
					rep.Hash = strings.TrimSpace(strings.TrimPrefix(seg, "Snippet SHA256:"))
					continue
				}
			}
			continue
		}
		if strings.HasPrefix(line, "Severity:") {
			rep.Severity = strings.TrimSpace(strings.TrimPrefix(line, "Severity:"))
			continue
		}
		if strings.HasPrefix(line, "Scanner:") {
			rep.Scanner = strings.TrimSpace(strings.TrimPrefix(line, "Scanner:"))
			continue
		}
		if strings.HasPrefix(line, "Rule:") {
			// Legacy fallback only if not already populated by new header format
			if rep.RuleID == "" {
				rep.RuleID = strings.TrimSpace(strings.TrimPrefix(line, "Rule:"))
			}
			continue
		}
		if strings.HasPrefix(line, "File:") {
			rep.FilePath = strings.TrimSpace(strings.TrimPrefix(line, "File:"))
			continue
		}
		if strings.HasPrefix(line, "Line:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "Line:"))
			if n, err := strconv.Atoi(v); err == nil {
				rep.StartLine = n
				rep.EndLine = n
			}
			continue
		}
		if strings.HasPrefix(line, "Lines:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "Lines:"))
			parts := strings.Split(v, "-")
			if len(parts) == 2 {
				if s, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
					rep.StartLine = s
				}
				if e, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					rep.EndLine = e
				}
			}
			continue
		}
		if strings.HasPrefix(line, "Snippet SHA256:") {
			rep.Hash = strings.TrimSpace(strings.TrimPrefix(line, "Snippet SHA256:"))
			continue
		}
		if strings.HasPrefix(line, "Permalink:") {
			rep.Permalink = strings.TrimSpace(strings.TrimPrefix(line, "Permalink:"))
			continue
		}
		// Check if line is a URL (for new format without "Permalink:" prefix)
		if strings.HasPrefix(line, "https://github.com/") && strings.Contains(line, "/blob/") {
			rep.Permalink = strings.TrimSpace(line)
			continue
		}
		// When we hit a non-metadata line and description is empty, assume rest is description
		if rep.Description == "" && line != "" {
			rep.Description = line
		}
	}
	return rep
}

// extractRuleIDFromBody attempts to parse a rule ID from the new body format header line:
// "## <emoji> <ruleID>" where <emoji> can be any single or combined emoji/symbol token.
// Returns empty string if not found.
func extractRuleIDFromBody(body string) string {
	// Compile regex once per call; trivial cost compared to network IO. If needed, lift to package scope.
	re := regexp.MustCompile(`^##\s+[^\w\s]+\s+(.+)$`)
	for _, line := range strings.Split(body, "\n") {
		l := strings.TrimSpace(line)
		if !strings.HasPrefix(l, "##") {
			continue
		}
		if m := re.FindStringSubmatch(l); len(m) == 2 {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

// listOpenIssues calls the VCS plugin to list open issues for the configured repo
// and parses their bodies into OpenIssueReport structures.
func listOpenIssues(options RunOptions) (map[int]OpenIssueEntry, error) {
	req := shared.VCSListIssuesRequest{
		VCSRequestBase: shared.VCSRequestBase{
			RepoParam: shared.RepositoryParams{
				Namespace:  options.Namespace,
				Repository: options.Repository,
			},
			Action: "listIssues",
		},
		State: "open",
	}

	var issues []shared.IssueParams
	err := shared.WithPlugin(AppConfig, "plugin-vcs", shared.PluginTypeVCS, "github", func(raw interface{}) error {
		vcs, ok := raw.(shared.VCS)
		if !ok {
			return fmt.Errorf("invalid VCS plugin type")
		}
		list, err := vcs.ListIssues(req)
		if err != nil {
			return err
		}
		issues = list
		return nil
	})
	if err != nil {
		return nil, err
	}

	reports := make(map[int]OpenIssueEntry, len(issues))
	for _, it := range issues {
		rep := parseIssueBody(it.Body)
		reports[it.Number] = OpenIssueEntry{
			OpenIssueReport: rep,
			Params:          it,
		}
	}
	return reports, nil
}

// processSARIFReport iterates runs/results in the SARIF report and creates VCS issues for
// high severity findings. Returns number of created issues or an error.
func processSARIFReport(report *internalsarif.Report, options RunOptions, lg hclog.Logger) (int, error) {
	// Build list of new issues from SARIF (only high severity -> "error").
	newIssues := make([]issuecorrelation.IssueMetadata, 0)
	// Also keep parallel arrays of the text bodies and titles so we can create issues later.
	newBodies := make([]string, 0)
	newTitles := make([]string, 0)

	for _, run := range report.Runs {

		// Build a map of rules keyed by rule ID for quick lookups
		rulesByID := map[string]*sarif.ReportingDescriptor{}
		if run.Tool.Driver != nil {
			for _, r := range run.Tool.Driver.Rules {
				if r == nil {
					continue
				}
				id := strings.TrimSpace(r.ID)
				if id == "" {
					continue
				}
				rulesByID[id] = r
			}
		}

		for _, res := range run.Results {
			level, _ := res.Properties["Level"].(string)
			if strings.ToLower(level) != "error" {
				continue
			}

			ruleID := ""
			if res.RuleID != nil {
				ruleID = *res.RuleID
			}

			fileURI := filepath.ToSlash(extractFileURIFromResult(res, options.SourceFolder))
			if fileURI == "" {
				fileURI = "<unknown>"
			}
			line, endLine := extractRegionFromResult(res)

			// desc := getStringProp(res.Properties, "Description")
			// if desc == "" && res.Message.Text != nil {
			// 	desc = *res.Message.Text
			// }

			snippetHash := computeSnippetHash(fileURI, line, endLine, options.SourceFolder)
			scannerName := getScannerName(run)
			sev := displaySeverity(level)

			// build body and title with scanner name label
			titleText := buildIssueTitle(scannerName, sev, ruleID, fileURI, line, endLine)

			// New body header and compact metadata blockquote
			header := ""
			if strings.TrimSpace(ruleID) != "" {
				header = fmt.Sprintf("## 🐞 %s\n\n", ruleID)
			}
			scannerDisp := scannerName
			if scannerDisp == "" {
				scannerDisp = "SARIF"
			}
			fileDisp := fileURI
			linesDisp := fmt.Sprintf("%d", line)
			if endLine > line {
				linesDisp = fmt.Sprintf("%d-%d", line, endLine)
			}
			meta := fmt.Sprintf(
				"> **Severity**: %s,  **Scanner**: %s\n> **File**: %s, **Lines**: %s\n",
				sev, scannerDisp, fileDisp, linesDisp,
			)
			// Only use the new header and blockquote metadata
			body := header + meta + "\n"
			var references []string

			// Append rule help markdown if available
			if r, ok := rulesByID[ruleID]; ok && r != nil && r.Help != nil && r.Help.Markdown != nil {
				if hm := strings.TrimSpace(*r.Help.Markdown); hm != "" {
					detail, helpRefs := parseRuleHelpMarkdown(hm)
					if detail != "" {
						body += "\n\n" + detail
					}
					if len(helpRefs) > 0 {
						references = append(references, helpRefs...)
					}
				}
			}

			// Append permalink if available
			if link := buildGitHubPermalink(options, fileURI, line, endLine); link != "" {
				body += fmt.Sprintf("\n%s\n", link)
			}

			// Append security identifier tags (CWE, OWASP) with links if available in rule properties
			if r, ok := rulesByID[ruleID]; ok && r != nil && r.Properties != nil {
				var tags []string
				if v, ok := r.Properties["tags"]; ok && v != nil {
					switch tv := v.(type) {
					case []string:
						tags = tv
					case []interface{}:
						for _, it := range tv {
							if s, ok := it.(string); ok {
								tags = append(tags, s)
							}
						}
					}
				}

				if len(tags) > 0 {
					cweRe := regexp.MustCompile(`^CWE-(\d+)\b`)
					owaspRe := regexp.MustCompile(`^OWASP[- ]?A(\d{2}):(\d{4})\s*-\s*(.+)$`)
					var tagRefs []string
					for _, tag := range tags {
						t := strings.TrimSpace(tag)
						if t == "" {
							continue
						}
						if m := cweRe.FindStringSubmatch(t); len(m) == 2 {
							num := m[1]
							url := fmt.Sprintf("https://cwe.mitre.org/data/definitions/%s.html", num)
							tagRefs = append(tagRefs, fmt.Sprintf("- [%s](%s)", t, url))
							continue
						}
						if m := owaspRe.FindStringSubmatch(t); len(m) == 4 {
							rank := m[1]
							year := m[2]
							title := m[3]
							slug := strings.ReplaceAll(strings.TrimSpace(title), " ", "_")
							// Remove characters that are not letters, numbers, underscore, or hyphen
							clean := make([]rune, 0, len(slug))
							for _, r := range slug {
								if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
									clean = append(clean, r)
								}
							}
							slug = string(clean)
							url := fmt.Sprintf("https://owasp.org/Top10/A%s_%s-%s/", rank, year, slug)
							tagRefs = append(tagRefs, fmt.Sprintf("- [%s](%s)", t, url))
							continue
						}
					}
					if len(tagRefs) > 0 {
						references = append(references, tagRefs...)
					}
				}
			}

			if len(references) > 0 {
				body += "\n\n<b>References:</b>\n" + strings.Join(references, "\n")
			}

			// Add a second snippet hash right before the scanio-managed note, as a blockquote
			if snippetHash != "" {
				body += fmt.Sprintf("\n\n> **Snippet SHA256**: %s\n", snippetHash)
			}
			body += "\n" + scanioManagedAnnotation

			newIssues = append(newIssues, issuecorrelation.IssueMetadata{
				IssueID:     "",
				Scanner:     scannerName,
				RuleID:      ruleID,
				Severity:    level,
				Filename:    fileURI,
				StartLine:   line,
				EndLine:     endLine,
				SnippetHash: snippetHash,
			})
			newBodies = append(newBodies, body)
			newTitles = append(newTitles, titleText)
		}
	}

	// Build list of known issues (open issues fetched previously by caller via listOpenIssues)
	openIssues, err := listOpenIssues(options)
	if err != nil {
		return 0, err
	}

	knownIssues := make([]issuecorrelation.IssueMetadata, 0, len(openIssues))
	for num, entry := range openIssues {
		rep := entry.OpenIssueReport
		// Only include well-structured issues for automatic closure.
		// If an open issue doesn't include basic metadata we skip it so
		// we don't accidentally close unrelated or free-form issues.
		if rep.Scanner == "" || rep.RuleID == "" || rep.FilePath == "" {
			lg.Debug("skipping malformed open issue (won't be auto-closed)", "number", num)
			continue
		}

		// Only consider issues that contain the scanio-managed annotation.
		// If the annotation is absent, treat the issue as manually managed and
		// exclude it from correlation/auto-closure logic.
		if !strings.Contains(entry.Params.Body, scanioManagedAnnotation) {
			lg.Debug("skipping non-scanio-managed issue (won't be auto-closed)", "number", num)
			continue
		}
		knownIssues = append(knownIssues, issuecorrelation.IssueMetadata{
			IssueID:     fmt.Sprintf("%d", num),
			Scanner:     rep.Scanner,
			RuleID:      rep.RuleID,
			Severity:    rep.Severity,
			Filename:    rep.FilePath,
			StartLine:   rep.StartLine,
			EndLine:     rep.EndLine,
			SnippetHash: rep.Hash,
		})
	}

	// correlate
	corr := issuecorrelation.NewCorrelator(newIssues, knownIssues)
	corr.Process()

	// Create only unmatched new issues
	unmatchedNew := corr.UnmatchedNew()
	created := 0
	for _, u := range unmatchedNew {
		// find corresponding index in newIssues to retrieve body/title
		var idx int = -1
		for ni, n := range newIssues {
			if n == u {
				idx = ni
				break
			}
		}
		if idx == -1 {
			// shouldn't happen
			continue
		}

		req := shared.VCSIssueCreationRequest{
			VCSRequestBase: shared.VCSRequestBase{
				RepoParam: shared.RepositoryParams{
					Namespace:  options.Namespace,
					Repository: options.Repository,
				},
				Action: "createIssue",
			},
			Title:     newTitles[idx],
			Body:      newBodies[idx],
			Labels:    opts.Labels,
			Assignees: opts.Assignees,
		}

		err := shared.WithPlugin(AppConfig, "plugin-vcs", shared.PluginTypeVCS, "github", func(raw interface{}) error {
			vcs, ok := raw.(shared.VCS)
			if !ok {
				return fmt.Errorf("invalid VCS plugin type")
			}
			_, err := vcs.CreateIssue(req)
			return err
		})
		if err != nil {
			lg.Error("failed to create issue via plugin", "error", err, "file", u.Filename, "line", u.StartLine)
			return created, errors.NewCommandError(options, nil, fmt.Errorf("create issue failed: %w", err), 2)
		}
		created++
	}

	// Close unmatched known issues (open issues that did not correlate)
	unmatchedKnown := corr.UnmatchedKnown()
	for _, k := range unmatchedKnown {
		// known IssueID contains the number as string
		num, err := strconv.Atoi(k.IssueID)
		if err != nil {
			// skip if we can't parse number
			continue
		}
		// Leave a comment before closing the issue to explain why it is being closed
		commentReq := shared.VCSCreateIssueCommentRequest{
			VCSRequestBase: shared.VCSRequestBase{
				RepoParam: shared.RepositoryParams{
					Namespace:  options.Namespace,
					Repository: options.Repository,
				},
				Action: "createIssueComment",
			},
			Number: num,
			Body:   "Recent scan didn't see the issue; closing this as resolved.",
		}

		err = shared.WithPlugin(AppConfig, "plugin-vcs", shared.PluginTypeVCS, "github", func(raw interface{}) error {
			vcs, ok := raw.(shared.VCS)
			if !ok {
				return fmt.Errorf("invalid VCS plugin type")
			}
			_, err := vcs.CreateIssueComment(commentReq)
			return err
		})
		if err != nil {
			lg.Error("failed to add comment before closing issue", "error", err, "number", num)
			// continue to attempt closing even if commenting failed
		}

		upd := shared.VCSIssueUpdateRequest{
			VCSRequestBase: shared.VCSRequestBase{
				RepoParam: shared.RepositoryParams{
					Namespace:  options.Namespace,
					Repository: options.Repository,
				},
				Action: "updateIssue",
			},
			Number: num,
			State:  "closed",
		}

		err = shared.WithPlugin(AppConfig, "plugin-vcs", shared.PluginTypeVCS, "github", func(raw interface{}) error {
			vcs, ok := raw.(shared.VCS)
			if !ok {
				return fmt.Errorf("invalid VCS plugin type")
			}
			_, err := vcs.UpdateIssue(upd)
			return err
		})
		if err != nil {
			lg.Error("failed to close issue via plugin", "error", err, "number", num)
			// continue closing others but report an error at end
			return created, errors.NewCommandError(options, nil, fmt.Errorf("close issue failed: %w", err), 2)
		}
	}

	return created, nil
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
