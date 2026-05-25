package vcsurl

import (
	"fmt"
	"strings"
)

// normalizeFilePath converts backslashes to forward slashes and trims leading slashes.
func normalizeFilePath(file string) string {
	return strings.TrimLeft(strings.ReplaceAll(file, "\\", "/"), "/")
}

// buildLineAnchor returns the dialect-correct URL fragment for a line range.
// Returns empty string when startLine <= 0.
//
//   - GitHub/Generic: #L{start} or #L{start}-L{end}
//   - GitLab:         #L{start} or #L{start}-{end}
//   - Bitbucket DC:   #{start}  or #{start}-{end}
func buildLineAnchor(vcsType VCSType, startLine, endLine int) string {
	if startLine <= 0 {
		return ""
	}
	if endLine <= 0 || endLine < startLine {
		endLine = startLine
	}
	switch vcsType {
	case Gitlab:
		if endLine == startLine {
			return fmt.Sprintf("#L%d", startLine)
		}
		return fmt.Sprintf("#L%d-%d", startLine, endLine)
	case Bitbucket:
		if endLine == startLine {
			return fmt.Sprintf("#%d", startLine)
		}
		return fmt.Sprintf("#%d-%d", startLine, endLine)
	default: // Github, GenericVCS, UnknownVCS
		if endLine == startLine {
			return fmt.Sprintf("#L%d", startLine)
		}
		return fmt.Sprintf("#L%d-L%d", startLine, endLine)
	}
}

// FilePermalink returns a stable file-at-ref link with an optional line range.
// Returns empty string when HTTPRepoLink, ref, or filePath is empty.
// endLine == 0 or endLine == startLine produces a single-line anchor.
func (u *VCSURL) FilePermalink(ref, filePath string, startLine, endLine int) string {
	if u.HTTPRepoLink == "" || ref == "" || filePath == "" {
		return ""
	}
	file := normalizeFilePath(filePath)
	anchor := buildLineAnchor(u.VCSType, startLine, endLine)
	switch u.VCSType {
	case Gitlab:
		return fmt.Sprintf("%s/-/blob/%s/%s%s", u.HTTPRepoLink, ref, file, anchor)
	case Bitbucket:
		return fmt.Sprintf("%s/browse/%s?at=%s%s", u.HTTPRepoLink, file, ref, anchor)
	default: // Github, GenericVCS, UnknownVCS
		return fmt.Sprintf("%s/blob/%s/%s%s", u.HTTPRepoLink, ref, file, anchor)
	}
}

// PRDiffLink returns a deep link to a specific line inside a PR diff.
// Returns empty string when PullRequestId is empty.
// For Bitbucket DC, line < 1 also returns empty string (single-line anchor required).
// For GitHub and GitLab a best-effort PR files/diffs tab link is returned regardless of line.
func (u *VCSURL) PRDiffLink(filePath string, line int) string {
	if u.HTTPRepoLink == "" || u.PullRequestId == "" {
		return ""
	}
	switch u.VCSType {
	case Bitbucket:
		if line < 1 || filePath == "" {
			return ""
		}
		file := normalizeFilePath(filePath)
		// Fragment encodes the file path; ?t=LINE is part of the fragment value.
		return fmt.Sprintf("%s/pull-requests/%s/diff#%s?t=%d", u.HTTPRepoLink, u.PullRequestId, file, line)
	case Gitlab:
		return fmt.Sprintf("%s/-/merge_requests/%s/diffs", u.HTTPRepoLink, u.PullRequestId)
	default: // Github, GenericVCS
		return fmt.Sprintf("%s/pull/%s/files", u.HTTPRepoLink, u.PullRequestId)
	}
}

// BranchURL returns the per-VCS branch tree URL.
// Returns empty string when HTTPRepoLink or branch is empty.
func (u *VCSURL) BranchURL(branch string) string {
	if u.HTTPRepoLink == "" || branch == "" {
		return ""
	}
	switch u.VCSType {
	case Bitbucket:
		return fmt.Sprintf("%s/browse?at=refs%%2Fheads%%2F%s", u.HTTPRepoLink, branch)
	case Gitlab:
		return fmt.Sprintf("%s/-/tree/%s", u.HTTPRepoLink, branch)
	default: // Github, GenericVCS
		return fmt.Sprintf("%s/tree/%s", u.HTTPRepoLink, branch)
	}
}

// CommitURL returns the bare commit summary page (no file, no line anchor).
// Returns empty string when HTTPRepoLink or sha is empty.
func (u *VCSURL) CommitURL(sha string) string {
	if u.HTTPRepoLink == "" || sha == "" {
		return ""
	}
	switch u.VCSType {
	case Bitbucket:
		return fmt.Sprintf("%s/commits/%s", u.HTTPRepoLink, sha)
	case Gitlab:
		return fmt.Sprintf("%s/-/commit/%s", u.HTTPRepoLink, sha)
	default: // Github, GenericVCS
		return fmt.Sprintf("%s/commit/%s", u.HTTPRepoLink, sha)
	}
}

// PRURL returns the PR landing page URL.
// Returns empty string when HTTPRepoLink or PullRequestId is empty.
func (u *VCSURL) PRURL() string {
	if u.HTTPRepoLink == "" || u.PullRequestId == "" {
		return ""
	}
	switch u.VCSType {
	case Bitbucket:
		return fmt.Sprintf("%s/pull-requests/%s", u.HTTPRepoLink, u.PullRequestId)
	case Gitlab:
		return fmt.Sprintf("%s/-/merge_requests/%s", u.HTTPRepoLink, u.PullRequestId)
	default: // Github, GenericVCS
		return fmt.Sprintf("%s/pull/%s", u.HTTPRepoLink, u.PullRequestId)
	}
}

// LineAnchor returns the dialect-correct URL fragment for a single line number.
// Intended for use by template helper functions that append an anchor to a base URL.
func (u *VCSURL) LineAnchor(line int) string {
	return buildLineAnchor(u.VCSType, line, line)
}
