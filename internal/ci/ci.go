// Package ci provides helpers for discovering CI metadata.
package ci

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// CIKind represents the type of CI.
type CIKind int

const (
	// CIUnknown indicates the CI provider could not be identified.
	CIUnknown CIKind = iota
	// CIGitHub identifies GitHub CI environments.
	CIGitHub
	// CIGitLab identifies GitLab CI environments.
	CIGitLab
	// CIBitbucket identifies Bitbucket CI environments.
	CIBitbucket
)

// LookupFunc fetches environment variables and defaults to os.Getenv.
type LookupFunc func(string) string

// CIEnvironment captures canonical CI metadata derived from environment variables.
type CIEnvironment struct {
	Kind               CIKind // Kind identifies the CI provider.
	CI                 bool   // CI reports whether the execution runs inside a CI environment.
	CommitHash         string // CommitHash is the tip commit that triggered the job.
	VCSServerURL       string // VCSServerURL is the scheme and host of the VCS server (e.g. https://vcs.domain/).
	Reference          string // Reference is the fully qualified git reference (e.g. refs/heads/main).
	ReferenceName      string // ReferenceName is the short reference or branch name.
	RepositoryName     string // RepositoryName is the repository slug without namespace.
	RepositoryFullName string // RepositoryFullName is the namespace-qualified repository name.
	RepositoryFullPath string // RepositoryFullPath is the full web URL for the repository.
	Namespace          string // Namespace is the owner or project namespace.
}

// String returns the human-readable string representation of a CIKind.
func (c CIKind) String() string {
	switch c {
	case CIGitHub:
		return "github"
	case CIGitLab:
		return "gitlab"
	case CIBitbucket:
		return "bitbucket"
	default:
		return "unknown"
	}
}

// ParseCIKind converts a string identifier into a CIKind value.
func ParseCIKind(raw string) (CIKind, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "github":
		return CIGitHub, nil
	case "gitlab":
		return CIGitLab, nil
	case "bitbucket":
		return CIBitbucket, nil
	default:
		return CIUnknown, fmt.Errorf("unsupported ci kind %q", raw)
	}
}

// DetectCIKind attempts to infer the CI provider from well-known environment variables.
func DetectCIKind() CIKind {
	return detectCIKindWithLookup(os.Getenv)
}

func detectCIKindWithLookup(lookup LookupFunc) CIKind {
	if lookup == nil {
		lookup = os.Getenv
	}

	if lookup("GITHUB_REPOSITORY") != "" || lookup("GITHUB_SHA") != "" {
		return CIGitHub
	}
	if strings.EqualFold(lookup("GITLAB_CI"), "true") || lookup("CI_PROJECT_PATH") != "" {
		return CIGitLab
	}
	if lookup("BITBUCKET_WORKSPACE") != "" || lookup("BITBUCKET_REPO_SLUG") != "" {
		return CIBitbucket
	}

	return CIUnknown
}

// GetCIDefaultEnvVars returns CI environment variables for the provided kind using the process environment.
func GetCIDefaultEnvVars(kind CIKind) (CIEnvironment, error) {
	return getCIDefaultEnvVars(kind, os.Getenv)
}

// getCIDefaultEnvVars resolves CI environment variables with the supplied lookup function.
func getCIDefaultEnvVars(kind CIKind, lookup LookupFunc) (CIEnvironment, error) {
	if lookup == nil {
		lookup = os.Getenv
	}

	switch kind {
	case CIGitHub:
		return extractGitHubVariables(lookup), nil
	case CIGitLab:
		return extractGitLabVariables(lookup), nil
	case CIBitbucket:
		return extractBitbucketVariables(lookup), nil
	default:
		return CIEnvironment{}, fmt.Errorf("unsupported ci kind: %s", kind)
	}
}

// extractGitHubVariables builds the CIEnvironment from GitHub-specific variables.
// See https://docs.github.com/en/actions/reference/workflows-and-actions/variables.
func extractGitHubVariables(lookup LookupFunc) CIEnvironment {
	ci, _ := strconv.ParseBool(lookup("CI"))

	fullName := lookup("GITHUB_REPOSITORY")
	repoName := ""
	if i := strings.LastIndex(fullName, "/"); i >= 0 && i < len(fullName)-1 {
		repoName = fullName[i+1:]
	}

	serverURL := lookup("GITHUB_SERVER_URL")
	fullPath := ""
	if serverURL != "" && fullName != "" {
		fullPath = serverURL + "/" + fullName
	}

	return CIEnvironment{
		Kind:               CIGitHub,
		CI:                 ci,
		CommitHash:         lookup("GITHUB_SHA"),
		VCSServerURL:       serverURL,                         // VCSServerURL includes only the scheme and host.
		Reference:          lookup("GITHUB_REF"),              // Reference stores the fully qualified ref (e.g., refs/heads/main).
		ReferenceName:      lookup("GITHUB_REF_NAME"),         // ReferenceName stores the short ref or branch name.
		RepositoryName:     repoName,                          // RepositoryName stores only the repository slug.
		RepositoryFullName: fullName,                          // RepositoryFullName stores the namespace and repository.
		RepositoryFullPath: fullPath,                          // RepositoryFullPath stores the HTTPS URL to the repository.
		Namespace:          lookup("GITHUB_REPOSITORY_OWNER"), // Namespace stores the owner or organization name.
	}
}

// extractGitLabVariables builds the CIEnvironment from GitLab-specific variables.
// See https://docs.gitlab.com/ci/variables/predefined_variables/.
func extractGitLabVariables(lookup LookupFunc) CIEnvironment {
	ci, _ := strconv.ParseBool(lookup("CI"))

	var fullRef, refName string
	if tag := lookup("CI_COMMIT_TAG"); tag != "" {
		// Tag pipeline.
		fullRef = "refs/tags/" + tag
		refName = tag
	} else if mrRef := lookup("CI_MERGE_REQUEST_REF_PATH"); mrRef != "" {
		// Merge request pipeline (e.g., refs/merge-requests/42/head).
		fullRef = mrRef
		if iid := lookup("CI_MERGE_REQUEST_IID"); iid != "" {
			refName = iid
		} else {
			// Fallback: source branch name if IID isn’t present for some reason.
			refName = lookup("CI_MERGE_REQUEST_SOURCE_BRANCH_NAME")
		}
	} else {
		refName = lookup("CI_COMMIT_REF_NAME")
		if refName != "" {
			fullRef = "refs/heads/" + refName
		}
	}

	return CIEnvironment{
		Kind:               CIGitLab,
		CI:                 ci,
		CommitHash:         lookup("CI_COMMIT_SHA"),
		VCSServerURL:       lookup("CI_SERVER_URL"),
		Reference:          fullRef,                        // Reference stores the fully qualified ref.
		ReferenceName:      refName,                        // ReferenceName stores the short ref or branch name.
		RepositoryName:     lookup("CI_PROJECT_NAME"),      // RepositoryName stores only the project slug.
		RepositoryFullName: lookup("CI_PROJECT_PATH"),      // RepositoryFullName stores the namespace and project name.
		RepositoryFullPath: lookup("CI_PROJECT_URL"),       // RepositoryFullPath stores the HTTPS URL to the project.
		Namespace:          lookup("CI_PROJECT_NAMESPACE"), // Namespace stores the namespace or group path.
	}
}

// extractBitbucketVariables builds the CIEnvironment from Bitbucket-specific variables.
// See https://support.atlassian.com/bitbucket-cloud/docs/variables-and-secrets/.
func extractBitbucketVariables(lookup LookupFunc) CIEnvironment {
	ci, _ := strconv.ParseBool(lookup("CI"))

	var reference, refName string
	if tag := lookup("BITBUCKET_TAG"); tag != "" {
		reference = "refs/tags/" + tag
		refName = tag
	} else if branch := lookup("BITBUCKET_BRANCH"); branch != "" {
		reference = "refs/heads/" + branch
		refName = branch
	} else if pr := lookup("BITBUCKET_PR_ID"); pr != "" {
		reference = "refs/pull/" + pr
		refName = pr
	}

	origin := lookup("BITBUCKET_GIT_HTTP_ORIGIN")
	u, err := url.Parse(origin)
	var serverURL string
	if err == nil && u.Scheme != "" && u.Host != "" {
		serverURL = u.Scheme + "://" + u.Host
	}

	return CIEnvironment{
		Kind:               CIBitbucket,
		CI:                 ci,
		CommitHash:         lookup("BITBUCKET_COMMIT"),
		VCSServerURL:       serverURL,                          // VCSServerURL includes only the scheme and host.
		Reference:          reference,                          // Reference stores the fully qualified ref.
		ReferenceName:      refName,                            // ReferenceName stores the short ref or branch name.
		RepositoryName:     lookup("BITBUCKET_REPO_SLUG"),      // RepositoryName stores only the repository slug.
		RepositoryFullName: lookup("BITBUCKET_REPO_FULL_NAME"), // RepositoryFullName stores the workspace and repository.
		RepositoryFullPath: origin,                             // RepositoryFullPath stores the HTTPS URL to the repository.
		Namespace:          lookup("BITBUCKET_WORKSPACE"),      // Namespace stores the workspace identifier.
	}
}
