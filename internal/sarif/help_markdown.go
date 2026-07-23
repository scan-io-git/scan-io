package sarif

import (
	"regexp"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

// FixPart represents one segment of a suggested-fix body: either a prose
// paragraph or a fenced code block extracted from markdown.
type FixPart struct {
	Type    string // "prose" or "code"
	Content string
	Lang    string // non-empty only for "code" parts that carry a language tag
}

var (
	urlRE          = regexp.MustCompile(`https?://\S+`)
	trailRE        = regexp.MustCompile(`[.,);>\]]+$`)
	refsHeadRE     = regexp.MustCompile(`(?i)^\s*(#{1,6}\s+references|\*\*references\*\*|references)\s*:?\s*$`)
	fixHeadRE      = regexp.MustCompile(`(?i)^\s*(#{1,6}\s+fix(es)?|\*\*fix(es)?\*\*|fix(es)?)\s*:?\s*$`)
	nextHeadRE     = regexp.MustCompile(`^\s*#{1,6}\s`)
	htmlRefsHeadRE = regexp.MustCompile(`(?i)<b>\s*references\s*:?\s*</b>`)
	htmlFixHeadRE  = regexp.MustCompile(`(?i)<b>\s*fix(es)?\s*:?\s*</b>`)
	// codeFenceRE matches a fenced code block: optional language tag on the
	// opening line, non-greedy body, closing fence.
	codeFenceRE = regexp.MustCompile("(?s)```(\\w*)\\n(.*?)\\n?```")
)

// normaliseMarkdown replaces HTML bold headings used by Semgrep (e.g. <b>References:</b>)
// with plain headings so the line-scan regexes can match them.
func normaliseMarkdown(md string) string {
	md = htmlRefsHeadRE.ReplaceAllString(md, "\nReferences\n")
	md = htmlFixHeadRE.ReplaceAllString(md, "\nFix\n")
	return md
}

// extractSection scans the normalised markdown for the first line that matches headingRE
// and returns all lines up to the next heading or end-of-input.
func extractSection(md string, headingRE *regexp.Regexp) []string {
	lines := strings.Split(normaliseMarkdown(md), "\n")
	var section []string
	inSection := false
	for _, line := range lines {
		if headingRE.MatchString(line) {
			inSection = true
			continue
		}
		if inSection {
			if nextHeadRE.MatchString(line) {
				break
			}
			section = append(section, line)
		}
	}
	return section
}

// extractReferencesFromMarkdown returns deduplicated http(s) URLs from the
// References section of a help.markdown string.
func extractReferencesFromMarkdown(md string) []string {
	if md == "" {
		return nil
	}
	section := extractSection(md, refsHeadRE)
	var urls []string
	for _, line := range section {
		for _, raw := range urlRE.FindAllString(line, -1) {
			// When the markdown link text is a URL itself, e.g. [https://a](https://b),
			// the regex captures "https://a](https://b" as one token. Use the part after "](").
			if i := strings.LastIndex(raw, "]("); i >= 0 {
				raw = raw[i+2:]
			}
			clean := trailRE.ReplaceAllString(raw, "")
			if clean != "" && strings.HasPrefix(clean, "http") {
				urls = append(urls, clean)
			}
		}
	}
	return urls
}

// extractFixFromMarkdown returns the text body of the Fix section in help.markdown,
// or "" when no Fix heading is found.
func extractFixFromMarkdown(md string) string {
	if md == "" {
		return ""
	}
	section := extractSection(md, fixHeadRE)
	trimmed := strings.TrimSpace(strings.Join(section, "\n"))
	return trimmed
}

// splitFixParts splits a raw fix string (which may contain markdown fenced code
// blocks) into an ordered slice of prose and code FixParts. Plain text with no
// fences produces a single prose part.
func splitFixParts(raw string) []FixPart {
	if raw == "" {
		return nil
	}
	var parts []FixPart
	pos := 0
	for _, loc := range codeFenceRE.FindAllStringSubmatchIndex(raw, -1) {
		// loc indices: [full-start full-end lang-start lang-end code-start code-end]
		if prose := strings.TrimSpace(raw[pos:loc[0]]); prose != "" {
			parts = append(parts, FixPart{Type: "prose", Content: prose})
		}
		lang := ""
		if loc[2] >= 0 {
			lang = raw[loc[2]:loc[3]]
		}
		code := ""
		if loc[4] >= 0 {
			code = raw[loc[4]:loc[5]]
		}
		parts = append(parts, FixPart{Type: "code", Content: code, Lang: lang})
		pos = loc[1]
	}
	if trailing := strings.TrimSpace(raw[pos:]); trailing != "" {
		parts = append(parts, FixPart{Type: "prose", Content: trailing})
	}
	return parts
}

// extractReferences returns up to maxRefs deduplicated http(s) URLs for a result,
// using a tiered fallback: result.properties.references → rule.helpUri → rule.help.markdown.
func extractReferences(result *sarif.Result, rule *sarif.ReportingDescriptor, maxRefs int) []string {
	if maxRefs <= 0 {
		return nil
	}

	var candidates []string

	// Tier 1: result.properties.references
	if result.Properties != nil {
		if raw, ok := result.Properties["references"]; ok {
			if list, ok := raw.([]any); ok {
				for _, item := range list {
					if s, ok := item.(string); ok && strings.HasPrefix(s, "http") {
						candidates = append(candidates, s)
					}
				}
				if len(candidates) > 0 {
					return finalize(candidates, maxRefs)
				}
			}
		}
	}

	if rule == nil {
		return nil
	}

	// Tier 2: rule.helpUri
	if rule.HelpURI != nil && strings.HasPrefix(*rule.HelpURI, "http") {
		return finalize([]string{*rule.HelpURI}, maxRefs)
	}

	// Tier 3: rule.help.markdown References section
	if rule.Help != nil && rule.Help.Markdown != nil {
		urls := extractReferencesFromMarkdown(*rule.Help.Markdown)
		if len(urls) > 0 {
			return finalize(urls, maxRefs)
		}
	}

	return nil
}

// extractFix returns a suggested fix string for a result.
// Precedence: result.properties.recommendation → rule.help.markdown Fix section.
func extractFix(result *sarif.Result, rule *sarif.ReportingDescriptor) string {
	// Tier 1: result.properties.recommendation
	if result.Properties != nil {
		if raw, ok := result.Properties["recommendation"]; ok {
			if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}

	// Tier 2: rule.help.markdown Fix section
	if rule != nil && rule.Help != nil && rule.Help.Markdown != nil {
		return extractFixFromMarkdown(*rule.Help.Markdown)
	}

	return ""
}

// finalize deduplicates URLs and caps the result at maxRefs.
func finalize(urls []string, maxRefs int) []string {
	seen := make(map[string]bool, len(urls))
	var out []string
	for _, u := range urls {
		clean := trailRE.ReplaceAllString(u, "")
		if clean != "" && !seen[clean] {
			seen[clean] = true
			out = append(out, clean)
		}
		if len(out) == maxRefs {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
