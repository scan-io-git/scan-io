package sarif

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/internal/git"
)

// NormalisedSubfolder extracts and normalizes the subfolder from repository metadata.
// It returns the subfolder path with forward slashes and no leading/trailing slashes.
// Returns empty string if metadata is nil or subfolder is empty.
func NormalisedSubfolder(md *git.RepositoryMetadata) string {
	if md == nil {
		return ""
	}
	sub := strings.Trim(md.Subfolder, "/\\")
	// Replace all backslashes with forward slashes for cross-platform compatibility
	sub = strings.ReplaceAll(sub, "\\", "/")
	return sub
}

// PathWithin checks if a path is within another path (root).
// It handles both absolute and relative paths, attempting to resolve them first.
// Returns true if path is within root, or if root is empty.
func PathWithin(path, root string) bool {
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

// ResolveRelativeLocalPath resolves a relative URI to a local filesystem path.
// It tries multiple base directories in order of preference:
// 1. repoRoot
// 2. repoRoot/subfolder (if subfolder is provided)
// 3. absSource
//
// For each base, it checks if the resolved path exists on the filesystem and is within repoRoot.
// If no path exists, it returns the first candidate that would be within repoRoot.
// Falls back to absSource-based path if all else fails.
//
// Parameters:
//   - cleanURI: the relative URI path (already cleaned)
//   - repoRoot: the repository root directory (optional)
//   - subfolder: the subfolder within the repository (optional)
//   - absSource: the absolute source folder path (optional)
func ResolveRelativeLocalPath(cleanURI, repoRoot, subfolder, absSource string) string {
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

	// Try each base directory, checking if the file exists
	for _, base := range bases {
		candidate := filepath.Clean(filepath.Join(base, candidateRel))
		if repoRoot != "" && !PathWithin(candidate, repoRoot) {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// If no file exists, return the first valid candidate path
	if len(bases) > 0 {
		candidate := filepath.Clean(filepath.Join(bases[0], candidateRel))
		if repoRoot == "" || PathWithin(candidate, repoRoot) {
			return candidate
		}
	}

	// Final fallback to absSource
	if absSource != "" {
		return filepath.Clean(filepath.Join(absSource, candidateRel))
	}
	return ""
}

// ConvertToRepoRelativePath converts a SARIF artifact URI to a repository-relative path.
// This function handles both absolute and relative URIs, normalizing them to repo-relative paths.
//
// The conversion process:
// 1. Normalizes the URI (removes file:// prefix, converts to OS path separators)
// 2. For absolute paths: calculates relative path from repoRoot or sourceFolder
// 3. For relative paths: resolves to absolute first, then calculates repo-relative path
// 4. Ensures subfolder prefix is included when scanning from a subdirectory
//
// Parameters:
//   - rawURI: the artifact URI from SARIF (may be absolute or relative, may have file:// prefix)
//   - repoMetadata: repository metadata containing RepoRootFolder and Subfolder (optional)
//   - sourceFolder: the source folder provided by the user (optional)
//
// Returns:
//   - A forward-slash separated path relative to the repository root
//   - Empty string if the URI is invalid or empty
func ConvertToRepoRelativePath(rawURI string, repoMetadata *git.RepositoryMetadata, sourceFolder string) string {
	rawURI = strings.TrimSpace(rawURI)
	if rawURI == "" {
		return ""
	}

	repoPath := ""
	subfolder := NormalisedSubfolder(repoMetadata)
	var repoRoot string
	if repoMetadata != nil && strings.TrimSpace(repoMetadata.RepoRootFolder) != "" {
		repoRoot = filepath.Clean(repoMetadata.RepoRootFolder)
	}
	absSource := strings.TrimSpace(sourceFolder)
	if absSource != "" {
		if abs, err := filepath.Abs(absSource); err == nil {
			absSource = abs
		} else {
			absSource = filepath.Clean(absSource)
		}
	}

	// Normalize URI to the host OS path representation
	osURI := filepath.FromSlash(rawURI)
	osURI = strings.TrimPrefix(osURI, "file://")
	cleanURI := filepath.Clean(osURI)

	if filepath.IsAbs(cleanURI) {
		// Absolute path: calculate repo-relative path
		localPath := cleanURI
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
		// Relative path: resolve to absolute first
		localPath := ResolveRelativeLocalPath(cleanURI, repoRoot, subfolder, absSource)

		if repoRoot != "" && localPath != "" && PathWithin(localPath, repoRoot) {
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
	return repoPath
}

// BuildGitHubPermalink constructs a GitHub permalink for a file and line range.
// It takes the core components needed for URL construction and handles the anchor format.
// Returns empty string if any critical component is missing.
//
// Parameters:
//   - namespace: GitHub namespace/organization
//   - repository: GitHub repository name
//   - ref: Git reference (commit hash, branch, or tag)
//   - repoRelativePath: file path relative to repository root (forward slashes)
//   - startLine: starting line number (1-based)
//   - endLine: ending line number (1-based, defaults to startLine if 0)
//
// Returns:
//   - GitHub permalink string in format: https://github.com/{namespace}/{repo}/blob/{ref}/{file}#L{start}-L{end}
//   - Empty string if any required parameter is missing
func BuildGitHubPermalink(namespace, repository, ref, repoRelativePath string, startLine, endLine int) string {
	// Validate required parameters
	if namespace == "" || repository == "" || ref == "" || repoRelativePath == "" {
		return ""
	}

	// Build base URL
	baseURL := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", namespace, repository, ref, repoRelativePath)

	// Handle line anchor
	if startLine <= 0 {
		return baseURL
	}

	if endLine <= 0 || endLine == startLine || endLine < startLine {
		return fmt.Sprintf("%s#L%d", baseURL, startLine)
	}

	return fmt.Sprintf("%s#L%d-L%d", baseURL, startLine, endLine)
}

// ExtractFileURIFromResult derives both the repository-relative path and local filesystem path
// for the first location in a SARIF result. When repository metadata is available the repo-relative
// path is anchored at the repository root; otherwise the function falls back to trimming the
// provided source folder (preserving legacy behaviour).
func ExtractFileURIFromResult(res *sarif.Result, absSourceFolder string, repoMetadata *git.RepositoryMetadata) (string, string) {
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

	// Use shared function to get repo-relative path
	repoPath := ConvertToRepoRelativePath(rawURI, repoMetadata, absSourceFolder)

	// Calculate local path for file operations (snippet hashing, etc.)
	localPath := CalculateLocalPath(rawURI, repoMetadata, absSourceFolder)

	return repoPath, localPath
}

// CalculateLocalPath determines the absolute local filesystem path for a SARIF URI.
// This is used for reading files for snippet hashing and other local file operations.
func CalculateLocalPath(rawURI string, repoMetadata *git.RepositoryMetadata, absSourceFolder string) string {
	subfolder := NormalisedSubfolder(repoMetadata)
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

	// Normalize URI to the host OS path representation
	osURI := filepath.FromSlash(rawURI)
	osURI = strings.TrimPrefix(osURI, "file://")
	cleanURI := filepath.Clean(osURI)

	if filepath.IsAbs(cleanURI) {
		return cleanURI
	}

	// Relative path - resolve to absolute
	return ResolveRelativeLocalPath(cleanURI, repoRoot, subfolder, absSource)
}

// extractRegionFromResult returns start and end line numbers (0 when not present)
// taken ExtractRegionFromResult the SARIF result's first location region.
func ExtractRegionFromResult(res *sarif.Result) (int, int) {
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

// DisplayRuleHeading returns the preferred human-friendly rule heading for the issue body:
// 1. rule.ShortDescription.Text when available.
// 2. rule.Name when available.
// 3. rule.ID as a fallback.
func DisplayRuleHeading(rule *sarif.ReportingDescriptor) string {
	if rule != nil {
		if rule.ShortDescription != nil && rule.ShortDescription.Text != nil {
			if heading := strings.TrimSpace(*rule.ShortDescription.Text); heading != "" {
				return heading
			}
		}
		if rule.Name != nil {
			if heading := strings.TrimSpace(*rule.Name); heading != "" {
				return heading
			}
		}
		// Parse ruleId from rule.ID instead of separate parameter
		return strings.TrimSpace(rule.ID)
	}
	return ""
}
