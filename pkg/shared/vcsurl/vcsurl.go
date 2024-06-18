package vcsurl

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

func ExtractRepositoryInfoFromURL(Url string, VCSPlugName string) (shared.RepositoryParams, error) {
	var repoInfo shared.RepositoryParams
	u, err := url.ParseRequestURI(Url)
	if err != nil {
		return repoInfo, err
	}

	repoInfo.VCSUrl = u.Hostname()
	scheme := u.Scheme
	pathDirs := getPathDirs(u.Path)
	isHTTP := scheme == "http" || scheme == "https"

	switch VCSPlugName {
	case "bitbucket":
		return handleBitbucket(repoInfo, scheme, Url, u.Port(), isHTTP, pathDirs)
	case "github":
		return handleGithub(repoInfo, pathDirs)
	case "gitlab":
		return handleGitlab(repoInfo, pathDirs)
	default:
		return repoInfo, fmt.Errorf("unsupported VCS plugin name: %s", VCSPlugName)
	}
}

func getPathDirs(p string) []string {
	var pathDirs []string
	for _, dir := range strings.Split(p, "/") {
		if dir != "" {
			pathDirs = append(pathDirs, dir)
		}
	}
	return pathDirs
}

// The case is for a Bitbucket APIv1/Onprem URL format
func handleBitbucket(repoInfo shared.RepositoryParams, scheme, Url, port string, isHTTP bool, pathDirs []string) (shared.RepositoryParams, error) {
	// Case of fetching the whole VCS - https://bitbucket.com/ ???
	if len(pathDirs) == 0 && (isHTTP || scheme == "ssh") {
		repoInfo.HTTPLink = Url
		return repoInfo, nil
	}

	// Case is working with a whole project from a Web UI URL format - https://bitbucket.com/projects/<project_name>
	if len(pathDirs) == 2 && pathDirs[0] == "projects" && isHTTP {
		repoInfo.Namespace = pathDirs[1]
		repoInfo.HTTPLink = Url
		return repoInfo, nil
	}

	// PR fetching case - the type doesn't exist in SCM urls - https://bitbucket.com/projects/<project_name>/repos/<repo_name>/pull-requests/<id>
	if len(pathDirs) > 4 && pathDirs[0] == "projects" && pathDirs[4] == "pull-requests" && isHTTP {
		repoInfo.Namespace = pathDirs[1]
		repoInfo.Repository = pathDirs[3]
		repoInfo.PullRequestId = pathDirs[5]
		setBitbucketURLs(&repoInfo)
		return repoInfo, nil
	}

	// Case is working with a certain repo from a Web UI URL format - https://bitbucket.com/projects/<project_name>/repos/<repo_name>/browse
	if len(pathDirs) > 3 && pathDirs[0] == "projects" && pathDirs[2] == "repos" && isHTTP {
		repoInfo.Namespace = pathDirs[1]
		repoInfo.Repository = pathDirs[3]
		setBitbucketURLs(&repoInfo)
		return repoInfo, nil
	}

	// https://bitbucket.com/scm/<project_name>/
	if len(pathDirs) >= 2 && isHTTP && pathDirs[0] == "scm" {
		repoInfo.Namespace = pathDirs[1]
		// https://bitbucket.com/scm/<project_name>/<repo_name>.git
		if strings.HasSuffix(pathDirs[len(pathDirs)-1], ".git") {
			repoInfo.Repository = strings.TrimSuffix(pathDirs[len(pathDirs)-1], ".git")
			setBitbucketURLs(&repoInfo)
		}
		return repoInfo, nil
	}

	// ssh://git@bitbucket.com:7989/<project_name>/<repo_name>.git
	if scheme == "ssh" {
		repoInfo.Namespace = pathDirs[0]
		if strings.HasSuffix(pathDirs[len(pathDirs)-1], ".git") {
			repoInfo.Repository = strings.TrimSuffix(pathDirs[len(pathDirs)-1], ".git")
			setBitbucketURLsWithPort(&repoInfo, port) // User can override a port if he uses an ssh scheme format of URL
		}
		return repoInfo, nil
	}

	return repoInfo, fmt.Errorf("invalid Bitbucket URL: %s", Url)
}

func setBitbucketURLs(repoInfo *shared.RepositoryParams) {
	repoInfo.HTTPLink = fmt.Sprintf("https://%s/scm/%s/%s.git", repoInfo.VCSUrl, repoInfo.Namespace, repoInfo.Repository)
	repoInfo.SSHLink = fmt.Sprintf("ssh://git@%s:7989/%s/%s.git", repoInfo.VCSUrl, repoInfo.Namespace, repoInfo.Repository)
}

func setBitbucketURLsWithPort(repoInfo *shared.RepositoryParams, port string) {
	repoInfo.HTTPLink = fmt.Sprintf("https://%s/scm/%s/%s.git", repoInfo.VCSUrl, repoInfo.Namespace, repoInfo.Repository)
	repoInfo.SSHLink = fmt.Sprintf("ssh://git@%s:%s/%s/%s.git", repoInfo.VCSUrl, port, repoInfo.Namespace, repoInfo.Repository)
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
