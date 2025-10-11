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
	base := fmt.Sprintf("https://github.com/%s/%s", options.Namespace, options.Repository)
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
	anchor := ""
	if start > 0 {
		anchor = fmt.Sprintf("#L%d", start)
		if end > start {
			anchor = fmt.Sprintf("%s-L%d", anchor, end)
		}
	}

	return fmt.Sprintf("%s/blob/%s/%s%s", base, ref, path, anchor)
}

// extractFileURIFromResult derives both the repository-relative path and local filesystem path
// for the first location in a SARIF result. When repository metadata is available the repo-relative
// path is anchored at the repository root; otherwise the function falls back to trimming the
// provided source folder (preserving legacy behaviour).
func extractFileURIFromResult(res *sarif.Result, absSourceFolder string, repoMetadata *git.RepositoryMetadata) (string, string) {
	if res == nil || len(res.Locations) == 0 {
		return "", ""
	}
	loc := res.Locations[0]
	if loc.PhysicalLocation == nil {
		return "", ""
	}
	art := loc.PhysicalLocation.ArtifactLocation
	if art == nil || art.URI == nil {
		return "", ""
	}
	rawURI := strings.TrimSpace(*art.URI)
	if rawURI == "" {
		return "", ""
	}

	repoPath := ""
	localPath := ""
	subfolder := normalisedSubfolder(repoMetadata)
	var repoRoot string
	if repoMetadata != nil && strings.TrimSpace(repoMetadata.RepoRootFolder) != "" {
		repoRoot = filepath.Clean(repoMetadata.RepoRootFolder)
	}
	absSource := strings.TrimSpace(absSourceFolder)
	if absSource != "" {
		if abs, err := filepath.Abs(absSource); err == nil {
			absSource = abs
		} else {
			absSource = filepath.Clean(absSource)
		}
	}

	// Normalise URI to the host OS path representation
	osURI := filepath.FromSlash(rawURI)
	osURI = strings.TrimPrefix(osURI, "file://")
	cleanURI := filepath.Clean(osURI)

	if filepath.IsAbs(cleanURI) {
		localPath = cleanURI
		if repoRoot != "" {
			if rel, err := filepath.Rel(repoRoot, localPath); err == nil {
				if rel != "." && !strings.HasPrefix(rel, "..") {
					repoPath = filepath.ToSlash(rel)
				}
			}
		}
		if repoPath == "" && absSource != "" {
			if rel, err := filepath.Rel(absSource, localPath); err == nil {
				repoPath = filepath.ToSlash(rel)
			}
		}
		if repoPath == "" {
			repoPath = filepath.ToSlash(strings.TrimPrefix(localPath, string(filepath.Separator)))
		}
	} else {
		localPath = resolveRelativeLocalPath(cleanURI, repoRoot, subfolder, absSource)

		if repoRoot != "" && localPath != "" && pathWithin(localPath, repoRoot) {
			if rel, err := filepath.Rel(repoRoot, localPath); err == nil {
				if rel != "." {
					repoPath = filepath.ToSlash(rel)
				}
			}
		}

		if repoPath == "" {
			normalised := strings.TrimLeft(filepath.ToSlash(cleanURI), "./")
			if subfolder != "" && !strings.HasPrefix(normalised, subfolder+"/") && normalised != subfolder {
				repoPath = filepath.ToSlash(filepath.Join(subfolder, normalised))
			} else {
				repoPath = filepath.ToSlash(normalised)
			}
		}
	}

	repoPath = strings.TrimLeft(repoPath, "/")
	repoPath = filepath.ToSlash(repoPath)
	return repoPath, localPath
}

func resolveRelativeLocalPath(cleanURI, repoRoot, subfolder, absSource string) string {
	candidateRel := cleanURI
	var bases []string
	seen := map[string]struct{}{}

	addBase := func(base string) {
		if base == "" {
			return
		}
		if abs, err := filepath.Abs(base); err == nil {
			base = abs
		} else {
			base = filepath.Clean(base)
		}
		if _, ok := seen[base]; ok {
			return
		}
		seen[base] = struct{}{}
		bases = append(bases, base)
	}

	addBase(repoRoot)
	if repoRoot != "" && subfolder != "" {
		addBase(filepath.Join(repoRoot, filepath.FromSlash(subfolder)))
	}
	addBase(absSource)

	for _, base := range bases {
		candidate := filepath.Clean(filepath.Join(base, candidateRel))
		if repoRoot != "" && !pathWithin(candidate, repoRoot) {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	if len(bases) > 0 {
		candidate := filepath.Clean(filepath.Join(bases[0], candidateRel))
		if repoRoot == "" || pathWithin(candidate, repoRoot) {
			return candidate
		}
	}

	if absSource != "" {
		return filepath.Clean(filepath.Join(absSource, candidateRel))
	}
	return ""
}

func pathWithin(path, root string) bool {
	if root == "" {
		return true
	}
	cleanPath, err1 := filepath.Abs(path)
	cleanRoot, err2 := filepath.Abs(root)
	if err1 != nil || err2 != nil {
		cleanPath = filepath.Clean(path)
		cleanRoot = filepath.Clean(root)
	}
	if cleanPath == cleanRoot {
		return true
	}
	rootWithSep := cleanRoot + string(filepath.Separator)
	return strings.HasPrefix(cleanPath, rootWithSep)
}

func normalisedSubfolder(md *git.RepositoryMetadata) string {
	if md == nil {
		return ""
	}
	sub := strings.Trim(md.Subfolder, "/\\")
	return filepath.ToSlash(sub)
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
