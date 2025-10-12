package vcsurl

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

type Protocol int

const (
	SSH Protocol = iota
	HTTP
)

type VCSType int

const (
	UnknownVCS VCSType = iota // UnknownVCS means that the type of VCS is unknown and not specified and should be determined from the URL
	GenericVCS                // Generic means that we should use generic handler and dynamic ignore vcs determination
	Github                    // Github means that the VCS is Github
	Gitlab                    // Gitlab means that the VCS is Gitlab
	Bitbucket                 // Bitbucket means that the VCS is Bitbucket
)

// getPathDirs splits the URL path into non-empty segments.
func GetPathDirs(path string) []string {
	var pathDirs []string
	for _, dir := range strings.Split(path, "/") {
		if dir != "" {
			pathDirs = append(pathDirs, dir)
		}
	}
	return pathDirs
}

// StringToVCSType converts a string to a VCSType
func StringToVCSType(s string) VCSType {
	switch s {
	case "github":
		return Github
	case "gitlab":
		return Gitlab
	case "bitbucket":
		return Bitbucket
	case "generic":
		return GenericVCS
	default:
		return UnknownVCS
	}
}

// define allows schemes: http, https and ssh
var validSchemes = []string{"http", "https", "ssh"}

// function to check whether the scheme is valid
func isValidScheme(scheme string) bool {
	for _, validScheme := range validSchemes {
		if scheme == validScheme {
			return true
		}
	}
	return false
}

// VCSURL represents a parsed VCS URL
type VCSURL struct {
	VCSType       VCSType
	Namespace     string
	Repository    string
	Branch        string
	PullRequestId string
	HTTPRepoLink  string
	SSHRepoLink   string
	ParsedURL     *url.URL
	Raw           string
	// Protocol      Protocol
	// FullName   string
	// Committish string
	// Username   string
}

// GetProtocol returns the protocol of the VCS URL (HTTP or SSH)
func (u *VCSURL) Protocol() Protocol {
	if u.ParsedURL.Scheme == "http" || u.ParsedURL.Scheme == "https" {
		return HTTP
	} else {
		return SSH
	}
}

// determineVCSType determines the VCS type based on the hostname
func determineVCSType(host string) (VCSType, error) {
	if strings.Contains(host, "github") {
		return Github, nil
	} else if strings.Contains(host, "gitlab") {
		return Gitlab, nil
	} else if strings.Contains(host, "bitbucket") {
		return Bitbucket, nil
	} else {
		return GenericVCS, fmt.Errorf("unknown VCS type for host: %q", host)
	}
}

// Parse parses a VCS URL and returns a VCSURL struct for unknown VCS Type
func Parse(raw string) (*VCSURL, error) {
	return ParseForVCSType(raw, UnknownVCS)
}

// ParseForVCSType parses a VCS URL and returns a VCSURL struct for a specific VCS Type
func ParseForVCSType(raw string, vcsType VCSType) (*VCSURL, error) {
	var vcsURL VCSURL
	vcsURL.Raw = raw

	// preparse special type of URLs like "git@<host>:<path>"
	spec := raw
	if parts := regexp.MustCompile(`^git@([^:]+)\:(.*)$`).FindStringSubmatch(spec); len(parts) == 3 {
		spec = fmt.Sprintf("ssh://%s/%s", parts[1], parts[2])
	}

	// strip .git suffix from the URL
	spec = strings.TrimSuffix(spec, ".git")

	// parse URL and save it as a struct field
	parsedURL, err := url.ParseRequestURI(spec)
	if err != nil {
		return nil, err
	}
	vcsURL.ParsedURL = parsedURL

	// validate scheme
	if !isValidScheme(vcsURL.ParsedURL.Scheme) {
		return nil, fmt.Errorf("invalid scheme: %q", vcsURL.Raw)
	}

	// determine VCS type either from the input or from the URL Hostname
	effectiveVCSType := vcsType
	if effectiveVCSType == UnknownVCS {
		effectiveVCSType, _ = determineVCSType(vcsURL.ParsedURL.Hostname())
	}
	vcsURL.VCSType = effectiveVCSType

	// handle the URL based on the VCS type
	switch effectiveVCSType {
	case Bitbucket:
		return parseBitbucket(vcsURL)
	case Github:
		return parseGithub(vcsURL)
	case Gitlab:
		return parseGitlab(vcsURL)
	default:
		return handleGenericVCS(vcsURL)
	}
}

func handleGenericVCS(u VCSURL) (*VCSURL, error) {
	pathDirs := GetPathDirs(u.ParsedURL.Path)

	// Case of working with the whole VCS
	if len(pathDirs) == 0 {
		return &u, nil
	}

	// Case of working with the whole project
	if len(pathDirs) == 1 {
		u.Namespace = pathDirs[0]
		return &u, nil
	}

	// Case of working with the certain repo
	u.Namespace = path.Join(pathDirs[0 : len(pathDirs)-1]...)
	u.Repository = pathDirs[len(pathDirs)-1]
	u.HTTPRepoLink = fmt.Sprintf("https://%s/%s/%s", u.ParsedURL.Hostname(), u.Namespace, u.Repository)
	u.SSHRepoLink = fmt.Sprintf("ssh://git@%s/%s/%s.git", u.ParsedURL.Hostname(), u.Namespace, u.Repository)
	return &u, nil
}

// parseGitlab processes Gitlab URLs to extract repository information.
func parseGitlab(u VCSURL) (*VCSURL, error) {
	pathDirs := GetPathDirs(u.ParsedURL.Path)

	// Search for "merge_requests" in pathDirs (excluding the first three segments)
	mergeRequestIndex, branchIndex := -1, -1
	for i := 3; i < len(pathDirs); i++ {
		if pathDirs[i] == "merge_requests" {
			mergeRequestIndex = i
			break
		} else if pathDirs[i] == "tree" {
			branchIndex = i
			break
		}
	}

	switch {
	// Case of working with the whole VCS - https://gitlab.com/
	case len(pathDirs) == 0:
		return &u, nil
	// Case for working with a root group - https://gitlab.com/<group_name>
	case len(pathDirs) == 1:
		u.Namespace = pathDirs[0]
		return &u, nil
	// Case for working with a specific repository - https://gitlab.com/<group>/<subgroup>/.../<project>
	// Assumes the last segment is the repository name.
	case len(pathDirs) >= 2: // TODO: add '-/tree/main' search and verification
		if mergeRequestIndex > 2 && mergeRequestIndex+1 < len(pathDirs) && pathDirs[mergeRequestIndex-1] == "-" {
			// MR fetching case - https://gitlab.com/<group_name>/../<project_name>/-/merge_requests/<id>
			u.Namespace = path.Join(pathDirs[:mergeRequestIndex-2]...)
			u.Repository = pathDirs[mergeRequestIndex-2]
			u.PullRequestId = pathDirs[mergeRequestIndex+1]
		} else if branchIndex > 2 && pathDirs[branchIndex-1] == "-" {
			// Repo + Branch fetching case - https://gitlab.com/<group_name>/<project_name>/-/tree/<branch_name>
			u.Namespace = path.Join(pathDirs[:branchIndex-2]...)
			u.Repository = pathDirs[branchIndex-2]
			u.Branch = strings.Join(pathDirs[branchIndex+1:], "/")
		} else {
			u.Namespace = path.Join(pathDirs[:len(pathDirs)-1]...)
			u.Repository = pathDirs[len(pathDirs)-1]
		}

		buildGenericURLs(&u)
		return &u, nil
	default:
		return &u, fmt.Errorf("invalid Gitlab URL: %q", u.Raw)
	}
}

// parseGithub processes Github URLs to extract repository information.
func parseGithub(u VCSURL) (*VCSURL, error) {
	pathDirs := GetPathDirs(u.ParsedURL.Path)

	switch {
	// Case of working with the whole VCS - https://github.com/
	case len(pathDirs) == 0:
		return &u, nil
	// Case for working with a whole project - https://github.com/<project_name>
	case len(pathDirs) == 1:
		u.Namespace = pathDirs[0]
		return &u, nil
	// PR fetching case - https://github.com/<project_name>/<repo_name>/pull/<id>
	// Case for working with a specific repo with a branch https://github.com/<project_name>/<repo_name>/tree/<branch_name>
	case len(pathDirs) > 3:
		u.Namespace = pathDirs[0]
		u.Repository = pathDirs[1]
		if pathDirs[2] == "pull" {
			u.PullRequestId = pathDirs[3]
		} else if pathDirs[2] == "tree" {
			u.Branch = strings.Join(pathDirs[3:], "/")
		}
		buildGenericURLs(&u)
		return &u, nil
	// Case for working with a specific repo - https://github.com/<project_name>/<repo_name>/
	case len(pathDirs) > 1:
		u.Namespace = pathDirs[0]
		u.Repository = pathDirs[1]
		buildGenericURLs(&u)
		return &u, nil
	default:
		return &u, fmt.Errorf("invalid Github URL: %q", u.Raw)
	}
}

// parseBitbucket processes Bitbucket URLs to extract repository information. The case is for a Bitbucket APIv1/Onprem URL format
func parseBitbucket(u VCSURL) (*VCSURL, error) {
	pathDirs := GetPathDirs(u.ParsedURL.Path)
	queryParams := u.ParsedURL.Query()

	switch {
	// Case for fetching the whole VCS - https://bitbucket.com/
	case len(pathDirs) == 0:
		return &u, nil
	// Case for working with a whole project from a Web UI URL format - https://bitbucket.com/projects/<project_name>
	case len(pathDirs) == 2 && pathDirs[0] == "projects" && u.Protocol() == HTTP:
		u.Namespace = pathDirs[1]
		return &u, nil
	// Case for working with user project - https://bitbucket.com/users/<username>
	case len(pathDirs) == 2 && pathDirs[0] == "users" && u.Protocol() == HTTP:
		u.Namespace = strings.Join([]string{pathDirs[0], pathDirs[1]}, "/")
		buildBitbucketURLs(&u, false, "", true)
		return &u, nil
	// Case for working with user repositories - https://bitbucket.com/users/<username>/repos/<repo_name>/browse?at=refs%2Fheads%2F<branch_name>
	case len(pathDirs) > 3 && pathDirs[0] == "users" && pathDirs[2] == "repos" && u.Protocol() == HTTP:
		u.Namespace = strings.Join([]string{pathDirs[0], pathDirs[1]}, "/")
		u.Repository = pathDirs[3]
		if refParam, exists := queryParams["at"]; exists && len(refParam) > 0 {
			u.Branch = refParam[0]
		}
		buildBitbucketURLs(&u, false, "", true)
		return &u, nil
	// PR fetching case - the type doesn't exist in SCM URLs - https://bitbucket.com/projects/<project_name>/repos/<repo_name>/pull-requests/<id>
	case len(pathDirs) > 5 && pathDirs[0] == "projects" && pathDirs[4] == "pull-requests" && u.Protocol() == HTTP:
		u.Namespace = pathDirs[1]
		u.Repository = pathDirs[3]
		u.PullRequestId = pathDirs[5]
		buildBitbucketURLs(&u, false, "", false)
		return &u, nil
	// Case for working with a specific repo from a Web UI URL format - https://bitbucket.com/projects/<project_name>/repos/<repo_name>/browse?at=refs%2Fheads%2F<branch_name>
	case len(pathDirs) > 3 && pathDirs[0] == "projects" && pathDirs[2] == "repos" && u.Protocol() == HTTP:
		u.Namespace = pathDirs[1]
		u.Repository = pathDirs[3]
		if refParam, exists := queryParams["at"]; exists && len(refParam) > 0 {
			u.Branch = refParam[0]
		}
		buildBitbucketURLs(&u, false, "", false)
		return &u, nil
	// Case for SCM path - https://bitbucket.com/scm/<project_name>/
	case len(pathDirs) >= 2 && u.Protocol() == HTTP && pathDirs[0] == "scm":
		u.Namespace = pathDirs[1]
		if len(pathDirs) > 2 {
			u.Repository = pathDirs[len(pathDirs)-1] // https://bitbucket.com/scm/<project_name>/<repo_name>.git
			buildBitbucketURLs(&u, false, "", false)
		}
		return &u, nil
	// Case for SSH path - ssh://git@bitbucket.com:7989/<project_name>/<repo_name>.git
	// and ssh://git@git.bitbucket.com:7989/~<username>/<repo_name>.git
	case u.Protocol() == SSH:
		u.Namespace = pathDirs[0]
		if len(pathDirs) > 1 {
			u.Repository = pathDirs[len(pathDirs)-1]
			buildBitbucketURLs(&u, u.ParsedURL.Port() != "", u.ParsedURL.Port(), false) // User can override a port if they use an ssh scheme format of URL
		}
		return &u, nil
	default:
		return &u, fmt.Errorf("invalid Bitbucket URL: %q", u.Raw)
	}
}

// buildGenericURLs sets the HTTP and SSH URLs for repositories.
func buildGenericURLs(u *VCSURL) {
	u.HTTPRepoLink = fmt.Sprintf("https://%s/%s/%s", u.ParsedURL.Hostname(), u.Namespace, u.Repository)
	u.SSHRepoLink = fmt.Sprintf("ssh://git@%s/%s/%s.git", u.ParsedURL.Hostname(), u.Namespace, u.Repository)
}

// buildBitbucketURLs sets the HTTP and SSH URLs for repositories.
func buildBitbucketURLs(u *VCSURL, usePort bool, port string, isUserRepo bool) {
	namespace := u.Namespace
	if strings.Contains(u.Namespace, "users/") {
		namespace = strings.TrimPrefix(u.Namespace, "users/")
	}
	if isUserRepo {
		u.HTTPRepoLink = fmt.Sprintf("https://%s/users/%s/repos/%s", u.ParsedURL.Hostname(), namespace, u.Repository)
		u.SSHRepoLink = fmt.Sprintf("ssh://git@%s:7989/~%s/%s.git", u.ParsedURL.Hostname(), namespace, u.Repository)
	} else {
		u.HTTPRepoLink = fmt.Sprintf("https://%s/projects/%s/repos/%s", u.ParsedURL.Hostname(), namespace, u.Repository)
		u.SSHRepoLink = fmt.Sprintf("ssh://git@%s:7989/%s/%s.git", u.ParsedURL.Hostname(), namespace, u.Repository)
	}

	if usePort {
		u.SSHRepoLink = fmt.Sprintf("ssh://git@%s:%s/%s/%s.git", u.ParsedURL.Hostname(), port, namespace, u.Repository)
		if isUserRepo {
			u.SSHRepoLink = fmt.Sprintf("ssh://git@%s:%s/~%s/%s.git", u.ParsedURL.Hostname(), port, namespace, u.Repository)
		}
	}
}
