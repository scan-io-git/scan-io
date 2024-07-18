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
	Namespace     string
	Repository    string
	HTTPRepoLink  string
	SSHRepoLink   string
	Raw           string
	PullRequestId string
	VCSType       VCSType
	ParsedURL     *url.URL
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
		return GenericVCS, fmt.Errorf("unknown VCS type for host: %s", host)
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
		return nil, fmt.Errorf("invalid scheme: %s", vcsURL.Raw)
	}

	// determine VCS type either from the input or from the URL Hostname
	effectiveVCSType := vcsType
	if effectiveVCSType == UnknownVCS {
		effectiveVCSType, _ = determineVCSType(vcsURL.ParsedURL.Hostname())
	}
	vcsURL.VCSType = effectiveVCSType

	// handle the URL based on the VCS type
	if effectiveVCSType == Bitbucket {
		return parseBitbucket(vcsURL)
	} else {
		return handleGenericVCS(vcsURL)
	}
}

// handleGenericVCS processes generic VCS URLs to extract repository information
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

// parseBitbucket processes Bitbucket URLs to extract repository information. The case is for a Bitbucket APIv1/Onprem URL format
func parseBitbucket(u VCSURL) (*VCSURL, error) {
	pathDirs := GetPathDirs(u.ParsedURL.Path)

	switch {
	case len(pathDirs) == 0:
		// Case for fetching the whole VCS - https://bitbucket.com/
		return &u, nil
	case len(pathDirs) == 2 && pathDirs[0] == "projects" && u.Protocol() == HTTP:
		// Case for working with a whole project from a Web UI URL format - https://bitbucket.com/projects/<project_name>
		u.Namespace = pathDirs[1]
		return &u, nil
	case len(pathDirs) > 3 && pathDirs[0] == "users" && pathDirs[2] == "repos" && u.Protocol() == HTTP:
		// Case for working with user repositories - https://bitbucket.com/users/<username>/repos/<repo_name>/browse
		u.Namespace = pathDirs[1]
		u.Repository = pathDirs[3]
		buildBitbucketURLs(&u, false, "", true)
		return &u, nil
	case len(pathDirs) > 4 && pathDirs[0] == "projects" && pathDirs[4] == "pull-requests" && u.Protocol() == HTTP:
		// PR fetching case - the type doesn't exist in SCM URLs - https://bitbucket.com/projects/<project_name>/repos/<repo_name>/pull-requests/<id>
		u.Namespace = pathDirs[1]
		u.Repository = pathDirs[3]
		u.PullRequestId = pathDirs[5]
		buildBitbucketURLs(&u, false, "", false)
		return &u, nil
	case len(pathDirs) > 3 && pathDirs[0] == "projects" && pathDirs[2] == "repos" && u.Protocol() == HTTP:
		// Case for working with a specific repo from a Web UI URL format - https://bitbucket.com/projects/<project_name>/repos/<repo_name>/browse
		u.Namespace = pathDirs[1]
		u.Repository = pathDirs[3]
		buildBitbucketURLs(&u, false, "", false)
		return &u, nil
	case len(pathDirs) >= 2 && u.Protocol() == HTTP && pathDirs[0] == "scm":
		// Case for SCM path - https://bitbucket.com/scm/<project_name>/
		u.Namespace = pathDirs[1]
		if len(pathDirs) > 2 {
			// https://bitbucket.com/scm/<project_name>/<repo_name>.git
			u.Repository = pathDirs[len(pathDirs)-1]
			buildBitbucketURLs(&u, false, "", false)
		}
		return &u, nil
	case u.Protocol() == SSH:
		// Case for SSH path - ssh://git@bitbucket.com:7989/<project_name>/<repo_name>.git
		// and ssh://git@git.bitbucket.com:7989/~<username>/<repo_name>.git
		u.Namespace = pathDirs[0]
		if len(pathDirs) > 1 {
			u.Repository = pathDirs[len(pathDirs)-1]
			buildBitbucketURLs(&u, u.ParsedURL.Port() != "", u.ParsedURL.Port(), false) // User can override a port if they use an ssh scheme format of URL
		}
		return &u, nil
	default:
		return &u, fmt.Errorf("invalid Bitbucket URL: %s", u.Raw)
	}
}

// buildBitbucketURLs sets the HTTP and SSH URLs for repositories.
func buildBitbucketURLs(u *VCSURL, usePort bool, port string, isUserRepo bool) {
	if isUserRepo {
		u.HTTPRepoLink = fmt.Sprintf("https://%s/users/%s/repos/%s/browse", u.ParsedURL.Hostname(), u.Namespace, u.Repository)
		u.SSHRepoLink = fmt.Sprintf("ssh://git@%s:7989/~%s/%s.git", u.ParsedURL.Hostname(), u.Namespace, u.Repository)
	} else {
		u.HTTPRepoLink = fmt.Sprintf("https://%s/scm/%s/%s.git", u.ParsedURL.Hostname(), u.Namespace, u.Repository)
		u.SSHRepoLink = fmt.Sprintf("ssh://git@%s:7989/%s/%s.git", u.ParsedURL.Hostname(), u.Namespace, u.Repository)
	}

	if usePort {
		u.SSHRepoLink = fmt.Sprintf("ssh://git@%s:%s/%s/%s.git", u.ParsedURL.Hostname(), port, u.Namespace, u.Repository)
		if isUserRepo {
			u.SSHRepoLink = fmt.Sprintf("ssh://git@%s:%s/~%s/%s.git", u.ParsedURL.Hostname(), port, u.Namespace, u.Repository)
		}
	}
}
