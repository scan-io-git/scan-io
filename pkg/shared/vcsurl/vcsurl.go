package vcsurl

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

// ExtractRepositoryInfoFromURL extracts repository information from a given URL based on the VCS plugin name.
func ExtractRepositoryInfoFromURL(urlStr, vcsPlugName string) (shared.RepositoryParams, error) {
	var repoInfo shared.RepositoryParams
	u, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return repoInfo, err
	}

	repoInfo.VCSUrl = u.Hostname()
	scheme := u.Scheme
	pathDirs := getPathDirs(u.Path)
	isHTTP := scheme == "http" || scheme == "https"

	switch vcsPlugName {
	case "bitbucket":
		return handleBitbucket(repoInfo, scheme, urlStr, u.Port(), isHTTP, pathDirs)
	case "github":
		return handleGithub(repoInfo, pathDirs)
	case "gitlab":
		return handleGitlab(repoInfo, pathDirs)
	default:
		return repoInfo, fmt.Errorf("unsupported VCS plugin name: %s", vcsPlugName)
	}
}

// getPathDirs splits the URL path into non-empty segments.
func getPathDirs(path string) []string {
	var pathDirs []string
	for _, dir := range strings.Split(path, "/") {
		if dir != "" {
			pathDirs = append(pathDirs, dir)
		}
	}
	return pathDirs
}

// handleBitbucket processes Bitbucket URLs to extract repository information. The case is for a Bitbucket APIv1/Onprem URL format
func handleBitbucket(repoInfo shared.RepositoryParams, scheme, urlStr, port string, isHTTP bool, pathDirs []string) (shared.RepositoryParams, error) {
	switch {
	case len(pathDirs) == 0 && (isHTTP || scheme == "ssh"):
		// Case for fetching the whole VCS - https://bitbucket.com/
		repoInfo.HTTPLink = urlStr
		return repoInfo, nil
	case len(pathDirs) == 2 && pathDirs[0] == "projects" && isHTTP:
		// Case for working with a whole project from a Web UI URL format - https://bitbucket.com/projects/<project_name>
		repoInfo.Namespace = pathDirs[1]
		repoInfo.HTTPLink = urlStr
		return repoInfo, nil
	case len(pathDirs) > 3 && pathDirs[0] == "users" && pathDirs[2] == "repos" && isHTTP:
		// Case for working with user repositories - https://bitbucket.com/users/<username>/repos/<repo_name>/browse
		repoInfo.Namespace = pathDirs[1]
		repoInfo.Repository = pathDirs[3]
		setBitbucketURLs(&repoInfo, false, "", true)
		return repoInfo, nil
	case len(pathDirs) > 4 && pathDirs[0] == "projects" && pathDirs[4] == "pull-requests" && isHTTP:
		// PR fetching case - the type doesn't exist in SCM URLs - https://bitbucket.com/projects/<project_name>/repos/<repo_name>/pull-requests/<id>
		repoInfo.Namespace = pathDirs[1]
		repoInfo.Repository = pathDirs[3]
		repoInfo.PullRequestId = pathDirs[5]
		setBitbucketURLs(&repoInfo, false, "", false)
		return repoInfo, nil
	case len(pathDirs) > 3 && pathDirs[0] == "projects" && pathDirs[2] == "repos" && isHTTP:
		// Case for working with a specific repo from a Web UI URL format - https://bitbucket.com/projects/<project_name>/repos/<repo_name>/browse
		repoInfo.Namespace = pathDirs[1]
		repoInfo.Repository = pathDirs[3]
		setBitbucketURLs(&repoInfo, false, "", false)
		return repoInfo, nil
	case len(pathDirs) >= 2 && isHTTP && pathDirs[0] == "scm":
		// Case for SCM path - https://bitbucket.com/scm/<project_name>/
		repoInfo.Namespace = pathDirs[1]
		if strings.HasSuffix(pathDirs[len(pathDirs)-1], ".git") {
			// https://bitbucket.com/scm/<project_name>/<repo_name>.git
			repoInfo.Repository = strings.TrimSuffix(pathDirs[len(pathDirs)-1], ".git")
			setBitbucketURLs(&repoInfo, false, "", false)
		}
		return repoInfo, nil
	case scheme == "ssh":
		// Case for SSH path - ssh://git@bitbucket.com:7989/<project_name>/<repo_name>.git
		// and ssh://git@git.bitbucket.com:7989/~<username>/<repo_name>.git
		repoInfo.Namespace = pathDirs[0]
		if strings.HasSuffix(pathDirs[len(pathDirs)-1], ".git") {
			repoInfo.Repository = strings.TrimSuffix(pathDirs[len(pathDirs)-1], ".git")
			setBitbucketURLs(&repoInfo, true, port, strings.HasPrefix(pathDirs[0], "~")) // User can override a port if they use an ssh scheme format of URL
		}
		return repoInfo, nil
	default:
		return repoInfo, fmt.Errorf("invalid Bitbucket URL: %s", urlStr)
	}
}

// setBitbucketURLs sets the HTTP and SSH URLs for repositories.
func setBitbucketURLs(repoInfo *shared.RepositoryParams, usePort bool, port string, isUserRepo bool) {
	if isUserRepo {
		repoInfo.HTTPLink = fmt.Sprintf("https://%s/users/%s/repos/%s/browse", repoInfo.VCSUrl, repoInfo.Namespace, repoInfo.Repository)
		repoInfo.SSHLink = fmt.Sprintf("ssh://git@%s:7989/~%s/%s.git", repoInfo.VCSUrl, repoInfo.Namespace, repoInfo.Repository)
	} else {
		repoInfo.HTTPLink = fmt.Sprintf("https://%s/scm/%s/%s.git", repoInfo.VCSUrl, repoInfo.Namespace, repoInfo.Repository)
		repoInfo.SSHLink = fmt.Sprintf("ssh://git@%s:7989/%s/%s.git", repoInfo.VCSUrl, repoInfo.Namespace, repoInfo.Repository)
	}

	if usePort {
		repoInfo.SSHLink = fmt.Sprintf("ssh://git@%s:%s/%s/%s.git", repoInfo.VCSUrl, port, repoInfo.Namespace, repoInfo.Repository)
		if isUserRepo {
			repoInfo.SSHLink = fmt.Sprintf("ssh://git@%s:%s/~%s/%s.git", repoInfo.VCSUrl, port, repoInfo.Namespace, repoInfo.Repository)
		}
	}
}

func handleGithub(repoInfo shared.RepositoryParams, pathDirs []string) (shared.RepositoryParams, error) {
	// Case of working with the whole VCS
	if len(pathDirs) == 0 {
		return repoInfo, nil
	}

	// Case of working with the whole project
	if len(pathDirs) == 1 {
		repoInfo.Namespace = pathDirs[0]
		return repoInfo, nil
	}

	// Case of working with the certain repo
	if len(pathDirs) == 2 {
		repoInfo.Namespace = pathDirs[0]
		repoInfo.Repository = pathDirs[1]
		repoInfo.HTTPLink = fmt.Sprintf("https://%s/%s/%s.git", repoInfo.VCSUrl, repoInfo.Namespace, repoInfo.Repository)
		repoInfo.SSHLink = fmt.Sprintf("ssh://git@%s/%s/%s.git", repoInfo.VCSUrl, repoInfo.Namespace, repoInfo.Repository)
		return repoInfo, nil
	}

	return repoInfo, fmt.Errorf("invalid Github URL: %s", repoInfo.HTTPLink)
}

func handleGitlab(repoInfo shared.RepositoryParams, pathDirs []string) (shared.RepositoryParams, error) {
	if len(pathDirs) < 2 {
		return repoInfo, fmt.Errorf("unsupported format of Gitlab URL")
	}

	repoInfo.Namespace = path.Join(pathDirs[0 : len(pathDirs)-1]...)
	repoInfo.Repository = pathDirs[len(pathDirs)-1]
	repoInfo.HTTPLink = fmt.Sprintf("https://%s/%s/%s.git", repoInfo.VCSUrl, repoInfo.Namespace, repoInfo.Repository)
	repoInfo.SSHLink = fmt.Sprintf("git@%s:%s/%s.git", repoInfo.VCSUrl, repoInfo.Namespace, repoInfo.Repository)

	return repoInfo, nil
}
