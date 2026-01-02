package sarif

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/owenrumney/go-sarif/v2/sarif"

	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/pkg/shared/files"

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
func CollectIssuesFromFile(logger hclog.Logger, inputPath, sourceRoot, vcs, url string, includeSeverity []string, noSuppressions bool) ([]IssueData, error) {
	// Resolve source folder to absolute form for path calculations
	sourceFolderAbs := files.ResolveSourceFolder(sourceRoot, logger)
	report, err := ReadReport(inputPath, logger, sourceFolderAbs, noSuppressions)
	if err != nil {
		return nil, fmt.Errorf("read sarif report: %w", err)
	}
	return CollectIssues(report, vcs, includeSeverity)
}

// CollectIssues walks an already-loaded Report and emits a Finding per SARIF
// location, normalising artifact URIs and pulling message text/snippets.
func CollectIssues(report *Report, vcs string, includeSeverity []string) ([]IssueData, error) {
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
			if !isLevelAllowed(strings.ToLower(level), includeSeverity) {
				continue
			}

			ruleID := ""
			if res.RuleID != nil {
				ruleID = *res.RuleID
			}

			if strings.TrimSpace(ruleID) == "" {
				report.logger.Warn("SARIF result missing rule ID, skipping", "result_index", len(collected))
				continue
			}

			fileURI, localPath := ExtractFileURIFromResult(res, report.sourceFolder, repoMetadata)
			if fileURI == "" {
				fileURI = "<unknown>"
				report.logger.Warn("SARIF result missing file URI, using placeholder", "rule_id", ruleID)
			}

			line, endLine := ExtractRegionFromResult(res)
			if line <= 0 {
				report.logger.Warn("SARIF result missing line information", "rule_id", ruleID, "file", fileURI)
			}

			snippetHash := issuecorrelation.ComputeSnippetHash(localPath, line, endLine)
			if snippetHash == "" && fileURI != "<unknown>" && line > 0 {
				report.logger.Warn("failed to compute snippet hash", "rule_id", ruleID, "file", fileURI, "line", line, "local_path", localPath)
			}

			toolMeta, ok := ExtractToolNameAndVersionFromRun(run, reportToolMeta)
			if !ok {
				report.logger.Warn("SARIF run missing scanner/tool name, using fallback", "rule_id", ruleID)
			}

			sev := displaySeverity(level)
			var ruleDescriptor *sarif.ReportingDescriptor
			if r, ok := rulesByID[ruleID]; ok {
				ruleDescriptor = r
			}

			// build body and title with scanner name label
			ruleTitleComponent := displayRuleTitleComponent(ruleID, ruleDescriptor)
			titleText := buildIssueMarkdownTitle(toolMeta, sev, ruleTitleComponent, fileURI, line, endLine)

			// New body header and compact metadata blockquote
			header := ""
			if h := DisplayRuleHeading(ruleDescriptor); strings.TrimSpace(h) != "" {
				header = fmt.Sprintf("## 🐞 %s\n\n", h)
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

			var metaBuilder strings.Builder
			if trimmedID := strings.TrimSpace(ruleID); trimmedID != "" {
				metaBuilder.WriteString(fmt.Sprintf("> **Rule ID**: %s\n", trimmedID))
			}
			metaBuilder.WriteString(fmt.Sprintf(
				"> **Severity**: %s,  **Scanner**: %s\n", sev, scannerDisp,
			))
			metaBuilder.WriteString(fmt.Sprintf(
				"> **File**: %s, **Lines**: %s\n", fileDisp, linesDisp,
			))
			meta := metaBuilder.String()

			// Only use the new header and blockquote metadata
			body := header + meta
			var references []string

			// Add formatted result message if available
			// todo: adopt not only for github scenario
			primaryDetail := ""
			if res.Message.Markdown != nil || res.Message.Text != nil {
				formatOpts := MessageFormatOptions{
					SourceFolder: report.sourceFolder,
				}
				if formatted := FormatResultMessage(res, repoMetadata, formatOpts); formatted != "" {
					primaryDetail = formatted
				}
			}

			// issueDesc := getStringProp(res.Properties, "Description")
			// if issueDesc == "" && res.Message.Text != nil {
			// 	issueDesc = *res.Message.Text
			// }
			// primaryDetail := strings.TrimSpace(issueDesc)

			// Append issue description, falling back to rule help text if necessary
			if ruleDescriptor != nil {
				if detail, helpRefs := extractRuleDetail(ruleDescriptor); detail != "" || len(helpRefs) > 0 {
					if primaryDetail == "" {
						primaryDetail = detail
					}
					if len(helpRefs) > 0 {
						references = append(references, helpRefs...)
					}
				}
			}

			if primaryDetail != "" {
				body += fmt.Sprintf("\n\n### Description\n\n%s\n", primaryDetail)
			}

			if res.Locations[0].Properties["WebURL"] != nil {
				body += fmt.Sprintf("\n%s\n", res.Locations[0].Properties["WebURL"])
			}

			// Add code flow section if available
			// todo: adopt for common scenario
			// if codeFlowSection := FormatCodeFlows(res, options, repoMetadata, sourceFolderAbs); codeFlowSection != "" {
			// 	body += "\n\n---\n\n" + codeFlowSection + "\n\n---\n\n"
			// }

			// Append permalink if available
			// todo check issues with monorep subpath

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
					IssueID:     ruleID,
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

// isLevelAllowed checks if a SARIF level is in the allowed levels list
func isLevelAllowed(level string, allowedLevels []string) bool {
	//todo: revise to iclude all levels if its not specified
	// If no levels specified, default to "error" for backward compatibility
	if len(allowedLevels) == 0 {
		return strings.ToLower(level) == "error"
	}

	for _, allowed := range allowedLevels {
		if strings.ToLower(level) == strings.ToLower(allowed) {
			return true
		}
	}
	return false
}

// normalizeAndValidateLevels validates and normalizes severity levels input.
// Accepts both SARIF levels (error, warning, note, none) and display levels (High, Medium, Low, Info).
// Returns normalized SARIF levels and an error if mixing formats is detected.
func NormalizeAndValidateLevels(levels []string) ([]string, error) {
	if len(levels) == 0 {
		return []string{"error"}, nil
	}

	var sarifLevels []string
	var displayLevels []string
	var normalized []string

	// Check each level and categorize
	for _, level := range levels {
		normalizedLevel := strings.ToLower(strings.TrimSpace(level))

		// Check if it's a SARIF level
		if isSARIFLevel(normalizedLevel) {
			sarifLevels = append(sarifLevels, normalizedLevel)
			normalized = append(normalized, normalizedLevel)
		} else if isDisplayLevel(normalizedLevel) {
			displayLevels = append(displayLevels, normalizedLevel)
			// Convert display level to SARIF level
			sarifLevel := displayToSARIFLevel(normalizedLevel)
			normalized = append(normalized, sarifLevel)
		} else {
			return nil, fmt.Errorf("invalid severity level '%s'. Valid SARIF levels: error, warning, note, none. Valid display levels: high, medium, low, info", level)
		}
	}

	// Check for mixing formats
	if len(sarifLevels) > 0 && len(displayLevels) > 0 {
		return nil, fmt.Errorf("cannot mix SARIF levels (error, warning, note, none) with display levels (High, Medium, Low, Info)")
	}

	return normalized, nil
}

// isSARIFLevel checks if the normalized level is a valid SARIF level
func isSARIFLevel(level string) bool {
	switch level {
	case "error", "warning", "note", "none":
		return true
	default:
		return false
	}
}

// isDisplayLevel checks if the normalized level is a valid display level
func isDisplayLevel(level string) bool {
	switch level {
	case "high", "medium", "low", "info":
		return true
	default:
		return false
	}
}

// displayToSARIFLevel converts a display level to its corresponding SARIF level
func displayToSARIFLevel(displayLevel string) string {
	switch displayLevel {
	case "high":
		return "error"
	case "medium":
		return "warning"
	case "low":
		return "note"
	case "info":
		return "none"
	default:
		return displayLevel
	}
}

// displayRuleTitleComponent returns the identifier segment to embed in the GitHub issue title.
// Prefers rule.Name when available; falls back to ruleID.
func displayRuleTitleComponent(ruleID string, rule *sarif.ReportingDescriptor) string {
	if rule != nil && rule.Name != nil {
		if name := strings.TrimSpace(*rule.Name); name != "" {
			return name
		}
	}
	return strings.TrimSpace(ruleID)
}

// extractRuleDetail returns a detail string (markdown/plain) and optional reference links.
// Prefers rule.Help.Markdown when available; falls back to rule.FullDescription.Text.
func extractRuleDetail(rule *sarif.ReportingDescriptor) (string, []string) {
	if rule == nil {
		return "", nil
	}

	if rule.Help != nil && rule.Help.Markdown != nil {
		if hm := strings.TrimSpace(*rule.Help.Markdown); hm != "" {
			if detail, refs := parseRuleHelpMarkdown(hm); strings.TrimSpace(detail) != "" || len(refs) > 0 {
				return detail, refs
			}
		}
	}

	if rule.FullDescription != nil && rule.FullDescription.Text != nil {
		if fd := strings.TrimSpace(*rule.FullDescription.Text); fd != "" {
			return fd, nil
		}
	}
	return "", nil
}
