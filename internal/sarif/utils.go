package sarif

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const semgrepPromoFooter = "#### 💎 Enable cross-file analysis and Pro rules for free at <a href='https://sg.run/pro'>sg.run/pro</a>\n\n"

// Compiled regex patterns for security tag parsing
var (
	cweRegex   = regexp.MustCompile(`^CWE-(\d+)\b`)
	owaspRegex = regexp.MustCompile(`^OWASP[- ]?A(\d{2}):(\d{4})\s*-\s*(.+)$`)
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

// function that calculates md5 hash for a given text
func calculateMD5Hash(text string) string {
	hash := md5.New()
	io.WriteString(hash, text)
	return hex.EncodeToString(hash.Sum(nil))
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
