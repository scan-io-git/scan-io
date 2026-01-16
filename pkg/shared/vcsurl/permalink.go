package vcsurl

import (
	"errors"
	"fmt"
	"strings"
)

// Permalink builder errors
var (
	ErrMissingNamespace = errors.New("namespace is required")
	ErrMissingProject   = errors.New("project is required")
	ErrMissingRef       = errors.New("ref (branch, tag, or commit SHA) is required")
	ErrMissingFile      = errors.New("file path is required")
	ErrMissingHost      = errors.New("host is required for Generic/Unknown VCS type (no default available)")
)

// Default public hosts for each VCS type
var defaultHosts = map[VCSType]string{
	Github:    "github.com",
	Gitlab:    "gitlab.com",
	Bitbucket: "bitbucket.org",
}

// PermalinkParams holds parameters for building VCS file permalinks.
type PermalinkParams struct {
	VCSType   VCSType
	Host      string // Optional: defaults to public host for VCSType
	Namespace string
	Project   string
	Ref       string // Branch, tag, or commit SHA
	File      string // Repository-relative file path (forward slashes)
	StartLine int    // 1-based, 0 means no line anchor
	EndLine   int    // 1-based, 0 or equal to StartLine means single line
}

// validatePermalinkParams checks that all required parameters are present.
func validatePermalinkParams(p PermalinkParams) error {
	if p.Namespace == "" {
		return ErrMissingNamespace
	}
	if p.Project == "" {
		return ErrMissingProject
	}
	if p.Ref == "" {
		return ErrMissingRef
	}
	if p.File == "" {
		return ErrMissingFile
	}
	return nil
}

// resolveHost returns the host to use, falling back to default public hosts.
func resolveHost(vcsType VCSType, host string) (string, error) {
	if host != "" {
		return host, nil
	}
	if defaultHost, ok := defaultHosts[vcsType]; ok {
		return defaultHost, nil
	}
	return "", ErrMissingHost
}

// normalizeFilePath converts backslashes to forward slashes and trims leading slashes.
func normalizeFilePath(file string) string {
	return strings.TrimLeft(strings.ReplaceAll(file, "\\", "/"), "/")
}

// BuildPermalink generates a VCS file permalink from the given parameters.
// Returns an error if required parameters are missing.
//
// Supported VCS types and their URL formats:
//   - GitHub:    https://{host}/{ns}/{proj}/blob/{ref}/{file}#L{start}-L{end}
//   - GitLab:    https://{host}/{ns}/{proj}/-/blob/{ref}/{file}#L{start}-{end}
//   - Bitbucket: https://{host}/projects/{ns}/repos/{proj}/browse/{file}?at={ref}#{start}-{end}
//   - Generic:   Same as GitHub format
//
// For self-hosted instances, provide the Host parameter. If omitted, defaults to
// the public host (github.com, gitlab.com, bitbucket.org).
func BuildPermalink(p PermalinkParams) (string, error) {
	if err := validatePermalinkParams(p); err != nil {
		return "", err
	}

	host, err := resolveHost(p.VCSType, p.Host)
	if err != nil {
		return "", err
	}

	file := normalizeFilePath(p.File)

	switch p.VCSType {
	case Gitlab:
		return buildGitlabPermalink(host, p.Namespace, p.Project, p.Ref, file, p.StartLine, p.EndLine), nil
	case Bitbucket:
		return buildBitbucketPermalink(host, p.Namespace, p.Project, p.Ref, file, p.StartLine, p.EndLine), nil
	case Github, GenericVCS, UnknownVCS:
		fallthrough
	default:
		return buildGithubPermalink(host, p.Namespace, p.Project, p.Ref, file, p.StartLine, p.EndLine), nil
	}
}

// buildGithubPermalink constructs GitHub-style permalink.
// Format: https://{host}/{ns}/{proj}/blob/{ref}/{file}#L{start}-L{end}
func buildGithubPermalink(host, namespace, project, ref, file string, startLine, endLine int) string {
	baseURL := fmt.Sprintf("https://%s/%s/%s/blob/%s/%s", host, namespace, project, ref, file)
	anchor := buildLineAnchor(Github, startLine, endLine)
	return baseURL + anchor
}

// buildGitlabPermalink constructs GitLab-style permalink.
// Format: https://{host}/{ns}/{proj}/-/blob/{ref}/{file}#L{start}-{end}
func buildGitlabPermalink(host, namespace, project, ref, file string, startLine, endLine int) string {
	baseURL := fmt.Sprintf("https://%s/%s/%s/-/blob/%s/%s", host, namespace, project, ref, file)
	anchor := buildLineAnchor(Gitlab, startLine, endLine)
	return baseURL + anchor
}

// buildBitbucketPermalink constructs Bitbucket Server-style permalink.
// Format: https://{host}/projects/{ns}/repos/{proj}/browse/{file}?at={ref}#{start}-{end}
func buildBitbucketPermalink(host, namespace, project, ref, file string, startLine, endLine int) string {
	baseURL := fmt.Sprintf("https://%s/projects/%s/repos/%s/browse/%s?at=%s", host, namespace, project, file, ref)
	anchor := buildLineAnchor(Bitbucket, startLine, endLine)
	return baseURL + anchor
}

// buildLineAnchor constructs the line anchor portion of a permalink based on VCS type.
// Returns empty string if startLine <= 0.
//
// Anchor formats:
//   - GitHub/Generic: #L{start} or #L{start}-L{end}
//   - GitLab:         #L{start} or #L{start}-{end}
//   - Bitbucket:      #{start} or #{start}-{end}
func buildLineAnchor(vcsType VCSType, startLine, endLine int) string {
	if startLine <= 0 {
		return ""
	}

	// Normalize endLine: if not specified or less than start, treat as single line
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

	case Github, GenericVCS, UnknownVCS:
		fallthrough
	default:
		if endLine == startLine {
			return fmt.Sprintf("#L%d", startLine)
		}
		return fmt.Sprintf("#L%d-L%d", startLine, endLine)
	}
}
