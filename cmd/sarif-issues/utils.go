package sarifissues

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/internal/git"
)

// parseLineRange parses line range from strings like "123" or "123-456".
// Returns (start, end) where end equals start for single line numbers.
func parseLineRange(value string) (int, int) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "-") {
		parts := strings.SplitN(value, "-", 2)
		if len(parts) == 2 {
			start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil {
				return start, end
			}
		}
	} else {
		if line, err := strconv.Atoi(value); err == nil {
			return line, line
		}
	}
	return 0, 0
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

// generateOWASPSlug creates a URL-safe slug from OWASP title text.
// Converts spaces to underscores and removes non-alphanumeric characters except hyphens and underscores.
func generateOWASPSlug(title string) string {
	slug := strings.ReplaceAll(strings.TrimSpace(title), " ", "_")
	clean := make([]rune, 0, len(slug))
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			clean = append(clean, r)
		}
	}
	return string(clean)
}

// processSecurityTags converts security tags (CWE, OWASP) into reference links.
// Returns a slice of markdown reference links for recognized security identifiers.
func processSecurityTags(tags []string) []string {
	var tagRefs []string
	for _, tag := range tags {
		t := strings.TrimSpace(tag)
		if t == "" {
			continue
		}

		// Process CWE tags
		if m := cweRegex.FindStringSubmatch(t); len(m) == 2 {
			num := m[1]
			url := fmt.Sprintf("https://cwe.mitre.org/data/definitions/%s.html", num)
			tagRefs = append(tagRefs, fmt.Sprintf("- [%s](%s)", t, url))
			continue
		}

		// Process OWASP tags
		if m := owaspRegex.FindStringSubmatch(t); len(m) == 4 {
			rank := m[1]
			year := m[2]
			title := m[3]
			slug := generateOWASPSlug(title)
			url := fmt.Sprintf("https://owasp.org/Top10/A%s_%s-%s/", rank, year, slug)
			tagRefs = append(tagRefs, fmt.Sprintf("- [%s](%s)", t, url))
			continue
		}
	}
	return tagRefs
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
