package sarifissues

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/internal/git"
	internalsarif "github.com/scan-io-git/scan-io/internal/sarif"
	issuecorrelation "github.com/scan-io-git/scan-io/pkg/issuecorrelation"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
)

// Compiled regex patterns for security tag parsing
var (
	cweRegex   = regexp.MustCompile(`^CWE-(\d+)\b`)
	owaspRegex = regexp.MustCompile(`^OWASP[- ]?A(\d{2}):(\d{4})\s*-\s*(.+)$`)
)

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

// NewIssueData holds the data needed to create a new issue from SARIF results.
type NewIssueData struct {
	Metadata issuecorrelation.IssueMetadata
	Body     string
	Title    string
}

// parseIssueBody attempts to read the body produced by this command and extract
// known metadata from blockquote format lines. Only supports the new format:
// "> **Severity**: Error,  **Scanner**: Semgrep OSS"
// "> **File**: app.py, **Lines**: 11-29"
// Returns an OpenIssueReport with zero values when fields are missing.
func parseIssueBody(body string) OpenIssueReport {
	rep := OpenIssueReport{}

	// Extract rule ID from header format: "## 🐞 <ruleID>"
	if rid := extractRuleIDFromBody(body); rid != "" {
		rep.RuleID = rid
	}

	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)

		// Only process blockquote metadata lines
		if !strings.HasPrefix(line, "> ") {
			// Check for GitHub permalink URLs
			if rep.Permalink == "" && strings.HasPrefix(line, "https://github.com/") && strings.Contains(line, "/blob/") {
				rep.Permalink = line
			}
			// Capture first non-metadata line as description if empty
			if rep.Description == "" && line != "" && !strings.HasPrefix(line, "##") && !strings.HasPrefix(line, "<b>") {
				rep.Description = line
			}
			continue
		}

		// Remove "> " prefix and normalize bold markers
		content := strings.TrimSpace(strings.TrimPrefix(line, "> "))
		content = strings.ReplaceAll(content, "**", "")

		// Parse comma-separated metadata fields
		parts := strings.Split(content, ",")
		for _, part := range parts {
			segment := strings.TrimSpace(part)

			if strings.HasPrefix(segment, "Severity:") {
				rep.Severity = strings.TrimSpace(strings.TrimPrefix(segment, "Severity:"))
			} else if strings.HasPrefix(segment, "Scanner:") {
				rep.Scanner = strings.TrimSpace(strings.TrimPrefix(segment, "Scanner:"))
			} else if strings.HasPrefix(segment, "File:") {
				rep.FilePath = strings.TrimSpace(strings.TrimPrefix(segment, "File:"))
			} else if strings.HasPrefix(segment, "Lines:") {
				value := strings.TrimSpace(strings.TrimPrefix(segment, "Lines:"))
				rep.StartLine, rep.EndLine = parseLineRange(value)
			} else if strings.HasPrefix(segment, "Snippet SHA256:") {
				rep.Hash = strings.TrimSpace(strings.TrimPrefix(segment, "Snippet SHA256:"))
			}
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

// buildNewIssuesFromSARIF processes SARIF report and extracts high severity findings,
// returning structured data for creating new issues.
func buildNewIssuesFromSARIF(report *internalsarif.Report, options RunOptions, sourceFolderAbs string, repoMetadata *git.RepositoryMetadata, lg hclog.Logger) []NewIssueData {
	var newIssueData []NewIssueData

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

			// Warn about missing rule ID
			if strings.TrimSpace(ruleID) == "" {
				lg.Warn("SARIF result missing rule ID, skipping", "result_index", len(newIssueData))
				continue
			}

			fileURI, localPath := extractFileURIFromResult(res, sourceFolderAbs, repoMetadata)
			fileURI = filepath.ToSlash(strings.TrimSpace(fileURI))
			if fileURI == "" {
				fileURI = "<unknown>"
				lg.Warn("SARIF result missing file URI, using placeholder", "rule_id", ruleID)
			}
			line, endLine := extractRegionFromResult(res)

			// Warn about missing location information
			if line <= 0 {
				lg.Warn("SARIF result missing line information", "rule_id", ruleID, "file", fileURI)
			}

			snippetHash := computeSnippetHash(localPath, line, endLine)
			if snippetHash == "" && fileURI != "<unknown>" && line > 0 {
				lg.Warn("failed to compute snippet hash", "rule_id", ruleID, "file", fileURI, "line", line, "local_path", localPath)
			}

			scannerName := getScannerName(run)
			if scannerName == "" {
				lg.Warn("SARIF run missing scanner/tool name, using fallback", "rule_id", ruleID)
			}

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
			if link := buildGitHubPermalink(options, repoMetadata, fileURI, line, endLine); link != "" {
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
					if tagRefs := processSecurityTags(tags); len(tagRefs) > 0 {
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

			newIssueData = append(newIssueData, NewIssueData{
				Metadata: issuecorrelation.IssueMetadata{
					IssueID:     "",
					Scanner:     scannerName,
					RuleID:      ruleID,
					Severity:    level,
					Filename:    fileURI,
					StartLine:   line,
					EndLine:     endLine,
					SnippetHash: snippetHash,
				},
				Body:  body,
				Title: titleText,
			})
		}
	}

	return newIssueData
}

// buildKnownIssuesFromOpen converts open GitHub issues into correlation metadata,
// filtering for well-structured scanio-managed issues only.
func buildKnownIssuesFromOpen(openIssues map[int]OpenIssueEntry, lg hclog.Logger) []issuecorrelation.IssueMetadata {
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
	return knownIssues
}

// createUnmatchedIssues creates GitHub issues for new findings that don't correlate with existing issues.
// Returns the number of successfully created issues.
func createUnmatchedIssues(unmatchedNew []issuecorrelation.IssueMetadata, newIssues []issuecorrelation.IssueMetadata, newBodies, newTitles []string, options RunOptions, lg hclog.Logger) (int, error) {
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
	return created, nil
}

// closeUnmatchedIssues closes GitHub issues for known findings that don't correlate with current scan results.
// Returns an error if any issue closure fails.
func closeUnmatchedIssues(unmatchedKnown []issuecorrelation.IssueMetadata, options RunOptions, lg hclog.Logger) error {
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
			return errors.NewCommandError(options, nil, fmt.Errorf("close issue failed: %w", err), 2)
		}
	}
	return nil
}

// processSARIFReport iterates runs/results in the SARIF report and creates VCS issues for
// high severity findings. Returns number of created issues or an error.
func processSARIFReport(report *internalsarif.Report, options RunOptions, sourceFolderAbs string, repoMetadata *git.RepositoryMetadata, lg hclog.Logger, openIssues map[int]OpenIssueEntry) (int, error) {
	// Build list of new issues from SARIF using extracted function
	newIssueData := buildNewIssuesFromSARIF(report, options, sourceFolderAbs, repoMetadata, lg)

	// Extract metadata, bodies, and titles for correlation and issue creation
	newIssues := make([]issuecorrelation.IssueMetadata, len(newIssueData))
	newBodies := make([]string, len(newIssueData))
	newTitles := make([]string, len(newIssueData))

	for i, data := range newIssueData {
		newIssues[i] = data.Metadata
		newBodies[i] = data.Body
		newTitles[i] = data.Title
	}

	// Build list of known issues from the provided open issues data
	knownIssues := buildKnownIssuesFromOpen(openIssues, lg)

	// correlate
	corr := issuecorrelation.NewCorrelator(newIssues, knownIssues)
	corr.Process()

	// Create only unmatched new issues
	unmatchedNew := corr.UnmatchedNew()
	created, err := createUnmatchedIssues(unmatchedNew, newIssues, newBodies, newTitles, options, lg)
	if err != nil {
		return created, err
	}

	// Close unmatched known issues (open issues that did not correlate)
	unmatchedKnown := corr.UnmatchedKnown()
	if err := closeUnmatchedIssues(unmatchedKnown, options, lg); err != nil {
		return created, err
	}

	return created, nil
}
