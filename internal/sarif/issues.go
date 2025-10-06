package sarif

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/owenrumney/go-sarif/v2/sarif"

	"github.com/scan-io-git/scan-io/internal/git"

	issuecorrelation "github.com/scan-io-git/scan-io/pkg/issuecorrelation"
)

// IssueData represents a SARIF finding mapped into Scanio's issue model
// alongside rendered title/body strings ready to publish as comments or issues.
type IssueData struct {
	Metadata issuecorrelation.IssueMetadata
	Title    string
	Body     string
}

// CollectIssuesFromFile loads a SARIF file (reusing ReadReport) and returns
// the flattened findings. sourceRoot lets callers trim absolute URIs to a
// repo-relative path; set noSuppressions to true to drop suppressed results.
func CollectIssuesFromFile(logger hclog.Logger, inputPath, sourceRoot, vcs, url string, noSuppressions bool) ([]IssueData, error) {
	report, err := ReadReport(inputPath, logger, sourceRoot, noSuppressions)
	if err != nil {
		return nil, fmt.Errorf("read sarif report: %w", err)
	}
	return CollectIssues(report, vcs)
}

// CollectIssues walks an already-loaded Report and emits a Finding per SARIF
// location, normalising artifact URIs and pulling message text/snippets.
func CollectIssues(report *Report, vcs string) ([]IssueData, error) {
	if report == nil || report.Report == nil || len(report.Runs) == 0 {
		return nil, fmt.Errorf("sarif report has no runs")
	}

	repoMetadata, err := git.CollectRepositoryMetadata(report.sourceFolder)
	if err != nil {
		return nil, fmt.Errorf("can't collect repository metadata: %w", err)
	}

	locationBuilder, err := NewLocationURLBuilder(repoMetadata, vcs)
	if err != nil {
		report.logger.Warn("failed to prepare location URL builder", "error", err)
		locationBuilder = func(*sarif.Location) string { return "" }
	}

	locationWebURLCallback := func(location *sarif.Location) string {
		return locationBuilder(location)
	}

	report.EnrichResultsLevelProperty()
	report.EnrichResultsTitleProperty()
	report.EnrichResultsCodeFlowProperty(locationWebURLCallback)
	report.EnrichResultsLocationURIProperty(locationWebURLCallback)
	report.SortResultsByLevel()

	reportToolMeta, _ := report.ExtractToolNameAndVersion()

	collected := make([]IssueData, 0)

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
			// todo: severity filtering
			// if strings.ToLower(level) != "error" {
			// 	continue
			// }

			ruleID := ""
			if res.RuleID != nil {
				ruleID = *res.RuleID
			}

			if strings.TrimSpace(ruleID) == "" {
				report.logger.Warn("SARIF result missing rule ID, skipping", "result_index", len(collected))
				continue
			}

			fileURI := filepath.ToSlash(extractFileURIFromResult(res, report.sourceFolder))
			if fileURI == "" {
				fileURI = "<unknown>"
				report.logger.Warn("SARIF result missing file URI, using placeholder", "rule_id", ruleID)
			}

			line, endLine := extractRegionFromResult(res)
			snippetHash := computeSnippetHash(fileURI, line, endLine, report.sourceFolder)
			if snippetHash == "" && fileURI != "<unknown>" && line > 0 {
				report.logger.Warn("failed to compute snippet hash", "rule_id", ruleID, "file", fileURI, "line", line)
			}

			toolMeta, ok := extractToolNameAndVersionFromRun(run, reportToolMeta)
			if !ok {
				report.logger.Warn("SARIF run missing scanner/tool name, using fallback", "rule_id", ruleID)
			}

			sev := displaySeverity(level)

			// build body and title with scanner name label
			titleText := buildIssueMarkdownTitle(toolMeta, sev, ruleID, fileURI, line, endLine)

			// New body header and compact metadata blockquote
			header := ""
			if strings.TrimSpace(ruleID) != "" {
				header = fmt.Sprintf("## 🐞 %s\n\n", ruleID)
			}

			scannerDisp := "SARIF"
			if toolMeta != nil && toolMeta.Name != "" {
				var parts []string

				name := strings.TrimSpace(toolMeta.Name)
				if name != "" {
					parts = append(parts, name)
				}
				if toolMeta.Version != nil {
					version := strings.TrimSpace(*toolMeta.Version)
					if version != "" {
						parts = append(parts, "ver "+version)
					}
				}

				if len(parts) > 0 {
					scannerDisp = strings.Join(parts, " ")
				}
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

			issueDesc := getStringProp(res.Properties, "Description")
			if issueDesc == "" && res.Message.Text != nil {
				issueDesc = *res.Message.Text
			}

			// Append issue description, falling back to rule help text if necessary
			primaryDetail := strings.TrimSpace(issueDesc)
			if r, ok := rulesByID[ruleID]; ok && r != nil && r.Help != nil && r.Help.Markdown != nil {
				if hm := strings.TrimSpace(*r.Help.Markdown); hm != "" {
					detail, helpRefs := parseRuleHelpMarkdown(hm)
					if primaryDetail == "" {
						primaryDetail = detail
					}
					if len(helpRefs) > 0 {
						references = append(references, helpRefs...)
					}
				}
			}
			if primaryDetail != "" {
				body += "\n\n" + primaryDetail
			}

			// Append permalink if available
			// todo check issues with monorep subpath
			if res.Locations[0].Properties["WebURL"] != nil {
				body += fmt.Sprintf("\n%s\n", res.Locations[0].Properties["WebURL"])
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
				body += "\n\n**References:**\n" + strings.Join(references, "\n")
			}

			// Add a second snippet hash right before the scanio-managed note, as a blockquote
			if snippetHash != "" {
				body += fmt.Sprintf("\n\n> **Snippet SHA256**: %s\n", snippetHash)
			}
			body += "\n" + AnnotationByVCS(vcs)

			collected = append(collected, IssueData{
				Metadata: issuecorrelation.IssueMetadata{
					IssueID:     "",
					Scanner:     scannerDisp,
					RuleID:      ruleID,
					Severity:    level,
					Filename:    fileURI,
					StartLine:   line,
					EndLine:     endLine,
					SnippetHash: snippetHash,
				},
				Title: titleText,
				Body:  body,
			})
		}
	}
	return collected, nil
}

// buildIssueMarkdownTitle creates a concise issue title using scanner name (fallback to SARIF),
// severity, ruleID and location info. It formats as "[<scanner>][<severity>][<ruleID>] at"
// and includes a range when endLine > line.
func buildIssueMarkdownTitle(toolMeta *ToolMetadata, severity, ruleID, fileURI string, line, endLine int) string {
	label := strings.TrimSpace(toolMeta.Name)
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

	if props := loc.PhysicalLocation.Region.Properties; props != nil {
		if v, ok := props["StartLine"].(int); ok {
			start = v
		} else if v, ok := props["startLine"].(int); ok {
			start = v
		}
		if v, ok := props["EndLine"].(int); ok {
			end = v
		} else if v, ok := props["endLine"].(int); ok {
			end = v
		}
		// If we resolved both from properties there's no need to consult the pointers below.
		if start > 0 || end > 0 {
			return start, end
		}
	}

	if loc.PhysicalLocation.Region.StartLine != nil {
		start = *loc.PhysicalLocation.Region.StartLine
	}
	if loc.PhysicalLocation.Region.EndLine != nil {
		end = *loc.PhysicalLocation.Region.EndLine
	}
	return start, end
}
