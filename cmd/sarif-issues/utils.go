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

	"github.com/hashicorp/go-hclog"
	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/internal/git"
	internalsarif "github.com/scan-io-git/scan-io/internal/sarif"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"
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

// computeSnippetHash reads the snippet (single line or range) from a local filesystem path
// and returns its SHA256 hex string. Returns empty string on any error or if inputs are invalid.
func computeSnippetHash(localPath string, line, endLine int) string {
	if strings.TrimSpace(localPath) == "" || line <= 0 {
		return ""
	}
	data, err := os.ReadFile(localPath)
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
// current commit hash from repo metadata (falling back to collecting it
// directly when metadata is not provided). Returns empty string when any
// critical component is missing.
func buildGitHubPermalink(options RunOptions, repoMetadata *git.RepositoryMetadata, fileURI string, start, end int) string {
	ref := strings.TrimSpace(options.Ref)

	if ref == "" {
		if repoMetadata != nil && repoMetadata.CommitHash != nil && *repoMetadata.CommitHash != "" {
			ref = *repoMetadata.CommitHash
		} else if options.SourceFolder != "" {
			if md, err := git.CollectRepositoryMetadata(options.SourceFolder); err == nil && md.CommitHash != nil && *md.CommitHash != "" {
				ref = *md.CommitHash
			}
		}
	}

	if ref == "" || fileURI == "" || fileURI == "<unknown>" {
		return ""
	}

	path := filepath.ToSlash(fileURI)
	return internalsarif.BuildGitHubPermalink(options.Namespace, options.Repository, ref, path, start, end)
}

// ResolveSourceFolder resolves a source folder path to its absolute form for path calculations.
// It handles path expansion (e.g., ~) and absolute path resolution with graceful fallbacks.
// Returns an empty string if the input folder is empty or whitespace-only.
func ResolveSourceFolder(folder string, logger hclog.Logger) string {
	if folder := strings.TrimSpace(folder); folder != "" {
		expandedFolder, expandErr := files.ExpandPath(folder)
		if expandErr != nil {
			logger.Debug("failed to expand source folder; using raw value", "error", expandErr)
			expandedFolder = folder
		}
		if absFolder, absErr := filepath.Abs(expandedFolder); absErr != nil {
			logger.Debug("failed to resolve absolute source folder; using expanded value", "error", absErr)
			return expandedFolder
		} else {
			return filepath.Clean(absFolder)
		}
	}
	return ""
}

// ApplyEnvironmentFallbacks applies environment variable fallbacks to the run options.
// It sets namespace, repository, and ref from GitHub environment variables if not already provided.
func ApplyEnvironmentFallbacks(opts *RunOptions) {
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
}

// FormatCodeFlows formats code flows from SARIF results into collapsible markdown sections.
// Each thread flow is displayed in a separate <details> block with numbered steps and GitHub permalinks.
func FormatCodeFlows(result *sarif.Result, options RunOptions, repoMetadata *git.RepositoryMetadata, sourceFolderAbs string) string {
	if result == nil || len(result.CodeFlows) == 0 {
		return ""
	}

	var sections []string
	threadFlowCounter := 0

	for _, codeFlow := range result.CodeFlows {
		if codeFlow == nil || len(codeFlow.ThreadFlows) == 0 {
			continue
		}

		for _, threadFlow := range codeFlow.ThreadFlows {
			if threadFlow == nil || len(threadFlow.Locations) == 0 {
				continue
			}

			threadFlowCounter++
			var steps []string
			seenSteps := make(map[string]bool) // Track seen permalink+message combinations
			actualStepNum := 0                 // Track actual step number for sequential numbering

			for _, threadFlowLocation := range threadFlow.Locations {
				if threadFlowLocation == nil || threadFlowLocation.Location == nil {
					continue
				}

				location := threadFlowLocation.Location
				if location.PhysicalLocation == nil || location.PhysicalLocation.ArtifactLocation == nil {
					continue
				}

				// Extract file path and line information
				fileURI, _ := internalsarif.ExtractFileURIFromResult(&sarif.Result{
					Locations: []*sarif.Location{location},
				}, sourceFolderAbs, repoMetadata)

				if fileURI == "" {
					continue
				}

				// Extract line numbers
				startLine := 0
				endLine := 0
				if location.PhysicalLocation.Region != nil {
					if location.PhysicalLocation.Region.StartLine != nil {
						startLine = *location.PhysicalLocation.Region.StartLine
					}
					if location.PhysicalLocation.Region.EndLine != nil {
						endLine = *location.PhysicalLocation.Region.EndLine
					}
				}

				// Create GitHub permalink
				permalink := buildGitHubPermalink(options, repoMetadata, fileURI, startLine, endLine)

				// Format step with optional message text
				messageText := ""
				if location.Message != nil && location.Message.Text != nil && strings.TrimSpace(*location.Message.Text) != "" {
					messageText = strings.TrimSpace(*location.Message.Text)
				}

				// Create unique key for deduplication (permalink + message text)
				dedupKey := fmt.Sprintf("%s|%s", permalink, messageText)

				// Skip if we've already seen this exact combination
				if seenSteps[dedupKey] {
					continue
				}
				seenSteps[dedupKey] = true

				// Increment actual step number only when we add a step
				actualStepNum++

				// Format step with optional message text
				stepText := fmt.Sprintf("Step %d:", actualStepNum)
				if messageText != "" {
					stepText = fmt.Sprintf("Step %d: %s", actualStepNum, messageText)
				}

				// Add step text and permalink on separate lines
				if permalink != "" {
					steps = append(steps, stepText+"\n"+permalink)
				} else {
					steps = append(steps, stepText)
				}
			}

			if len(steps) > 0 {
				summary := fmt.Sprintf("Code Flow %d", threadFlowCounter)
				section := fmt.Sprintf("<details>\n<summary>%s</summary>\n\n%s\n</details>",
					summary, strings.Join(steps, "\n\n"))
				sections = append(sections, section)
			}
		}
	}

	if len(sections) == 0 {
		return ""
	}

	return strings.Join(sections, "\n\n")
}

// ApplyGitMetadataFallbacks applies git metadata fallbacks to the run options.
// It extracts namespace, repository, and ref from local git repository metadata
// when the corresponding flags are not already provided.
func ApplyGitMetadataFallbacks(opts *RunOptions, logger hclog.Logger) {
	// Determine the base folder for git metadata extraction
	baseFolder := strings.TrimSpace(opts.SourceFolder)
	if baseFolder == "" {
		// Use current working directory if source-folder is not provided
		if cwd, err := os.Getwd(); err == nil {
			baseFolder = cwd
		} else {
			logger.Debug("failed to get current working directory for git metadata extraction", "error", err)
			return
		}
	}

	// Collect git repository metadata
	repoMetadata, err := git.CollectRepositoryMetadata(baseFolder)
	if err != nil {
		logger.Debug("unable to collect git repository metadata", "error", err, "baseFolder", baseFolder)
		return
	}

	// Extract namespace and repository from git remote URL if not already set
	if strings.TrimSpace(opts.Namespace) == "" || strings.TrimSpace(opts.Repository) == "" {
		if repoMetadata.RepositoryFullName != nil && *repoMetadata.RepositoryFullName != "" {
			// Parse the repository URL to extract namespace and repository
			vcsURL, err := vcsurl.ParseForVCSType(*repoMetadata.RepositoryFullName, vcsurl.UnknownVCS)
			if err != nil {
				logger.Debug("failed to parse git repository URL", "error", err, "url", *repoMetadata.RepositoryFullName)
			} else {
				// Apply namespace if not already set
				if strings.TrimSpace(opts.Namespace) == "" && vcsURL.Namespace != "" {
					opts.Namespace = vcsURL.Namespace
					logger.Debug("auto-detected namespace from git metadata", "namespace", vcsURL.Namespace)
				}

				// Apply repository if not already set
				if strings.TrimSpace(opts.Repository) == "" && vcsURL.Repository != "" {
					opts.Repository = vcsURL.Repository
					logger.Debug("auto-detected repository from git metadata", "repository", vcsURL.Repository)
				}
			}
		}
	}

	// Extract commit hash for ref if not already set
	if strings.TrimSpace(opts.Ref) == "" {
		if repoMetadata.CommitHash != nil && *repoMetadata.CommitHash != "" {
			opts.Ref = *repoMetadata.CommitHash
			logger.Debug("auto-detected ref from git metadata", "ref", *repoMetadata.CommitHash)
		}
	}
}
