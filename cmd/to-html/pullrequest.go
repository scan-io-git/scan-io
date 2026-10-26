package tohtml

import (
	"os"
	"strings"

	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"
)

// resolvePullRequestID returns the pull-request id for the report.
// Precedence: explicit flag value, then CI environment variables keyed on the
// detected VCS type. Returns "" when nothing supplies an id.
func resolvePullRequestID(flagValue string, vcsType vcsurl.VCSType) string {
	if id := strings.TrimSpace(flagValue); id != "" {
		return id
	}
	switch vcsType {
	case vcsurl.Github:
		return prIDFromGitHubRef(os.Getenv("GITHUB_REF"))
	case vcsurl.Gitlab:
		return strings.TrimSpace(os.Getenv("CI_MERGE_REQUEST_IID"))
	case vcsurl.Bitbucket:
		return strings.TrimSpace(os.Getenv("BITBUCKET_PR_ID"))
	default:
		if id := prIDFromGitHubRef(os.Getenv("GITHUB_REF")); id != "" {
			return id
		}
		if id := strings.TrimSpace(os.Getenv("CI_MERGE_REQUEST_IID")); id != "" {
			return id
		}
		return strings.TrimSpace(os.Getenv("BITBUCKET_PR_ID"))
	}
}

// prIDFromGitHubRef extracts N from a GitHub PR ref like "refs/pull/123/merge".
// Returns "" if ref is not a pull-request ref.
func prIDFromGitHubRef(ref string) string {
	ref = strings.TrimSpace(ref)
	const prefix = "refs/pull/"
	if !strings.HasPrefix(ref, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(ref, prefix)
	if idx := strings.IndexByte(rest, '/'); idx > 0 {
		return rest[:idx]
	}
	return ""
}
