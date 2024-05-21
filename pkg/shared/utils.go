package shared

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"

	"log"
	"net/url"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/hashicorp/go-hclog"
)

type CustomData interface{}

type ScanReportData struct {
	ScanStarted  bool
	ScanPassed   bool
	ScanFailed   bool
	ScanCrashed  bool
	ScanDetails  interface{}
	ScanResults  interface{}
	ErrorDetails interface{}
}

func WriteJsonFile(outputFile string, logger hclog.Logger, data ...CustomData) {
	file, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer file.Close()

	datawriter := bufio.NewWriter(file)
	defer datawriter.Flush()

	resultJson, _ := json.MarshalIndent(data[0], "", "    ")
	datawriter.Write(resultJson)
	logger.Info("Results saved to file", "path", outputFile)

}

func ExtractRepositoryInfoFromURL(Url string, VCSPlugName string) (string, string, string, string, string, string, error) {
	var (
		namespace     string
		repository    string
		lastElement   string
		pathDirs      []string
		httpUrl       string
		sshUrl        string
		pullRequestId string
	)

	u, err := url.ParseRequestURI(Url)
	if err != nil {
		return "", "", "", "", "", "", err
	}

	vcsUrl := u.Hostname()
	scheme := u.Scheme

	// Split the path and remove empty elements
	for _, dir := range strings.Split(u.Path, "/") {
		if dir != "" {
			pathDirs = append(pathDirs, dir)
		}
	}
	if len(pathDirs) > 0 {
		lastElement = pathDirs[len(pathDirs)-1]
	}
	isHTTP := scheme == "http" || scheme == "https"

	switch VCSPlugName {
	case "bitbucket":
		// The case is for a Bitbucket APIv1 URL format
		// TODO
		// We can move building urls to just calling a list function
		// But bitbucketV1 library can't resolve a particular repo

		if len(pathDirs) == 0 && (isHTTP || scheme == "ssh") {
			// Case is working with a whole VCS
			return vcsUrl, namespace, repository, pullRequestId, Url, "", nil
		} else if len(pathDirs) == 2 && pathDirs[0] == "projects" && isHTTP {
			// Case is working with a whole project from a Web UI URL format
			// https://bitbucket.com/projects/<project_name>
			namespace = pathDirs[1]
			return vcsUrl, namespace, repository, Url, pullRequestId, "", nil
		} else if len(pathDirs) > 4 && pathDirs[0] == "projects" && pathDirs[4] == "pull-requests" && isHTTP {
			// PR fetching case - the type doesn't exist in SCM urls
			// https://bitbucket.com/projects/<project_name>/repos/<repo_name>/pull-requests/<id>
			namespace = pathDirs[1]
			repository = pathDirs[3]
			pullRequestId = pathDirs[5]
			httpUrl := fmt.Sprintf("https://%s/scm/%s/%s.git", vcsUrl, namespace, repository)
			sshUrl := fmt.Sprintf("ssh://git@%s:7989/%s/%s.git", vcsUrl, namespace, repository)
			return vcsUrl, namespace, repository, pullRequestId, httpUrl, sshUrl, nil
		} else if len(pathDirs) > 3 && pathDirs[0] == "projects" && pathDirs[2] == "repos" && isHTTP {
			// Case is working with a certain repo from a Web UI URL format
			// https://bitbucket.com/projects/<project_name>/repos/<repo_name>/browse
			namespace = pathDirs[1]
			repository = pathDirs[3]
			httpUrl := fmt.Sprintf("https://%s/scm/%s/%s.git", vcsUrl, namespace, repository)
			sshUrl := fmt.Sprintf("ssh://git@%s:7989/%s/%s.git", vcsUrl, namespace, repository)
			return vcsUrl, namespace, repository, pullRequestId, httpUrl, sshUrl, nil
		} else if len(pathDirs) >= 2 && isHTTP && pathDirs[0] == "scm" {
			// https://bitbucket.com/scm/<project_name>/
			namespace = pathDirs[1]
			if strings.Contains(lastElement, ".git") {
				// https://bitbucket.com/scm/<project_name>/<repo_name>.git
				repository = strings.TrimSuffix(lastElement, ".git")
				httpUrl = fmt.Sprintf("https://%s/scm/%s/%s.git", vcsUrl, namespace, repository)
				sshUrl = fmt.Sprintf("ssh://git@%s:7989/%s/%s.git", vcsUrl, namespace, repository)
			}
			return vcsUrl, namespace, repository, pullRequestId, httpUrl, sshUrl, nil
		} else if scheme == "ssh" {
			namespace = pathDirs[0]
			if strings.Contains(lastElement, ".git") {
				// ssh://git@bitbucket.com:7989/<project_name>/<repo_name>.git
				port := u.Port()
				repository = strings.TrimSuffix(lastElement, ".git")
				httpUrl = fmt.Sprintf("https://%s/scm/%s/%s.git", vcsUrl, namespace, repository)
				// User can override a port if he uses an ssh scheme format of URL
				sshUrl = fmt.Sprintf("ssh://git@%s:%s/%s/%s.git", vcsUrl, port, namespace, repository)
			}
			return vcsUrl, namespace, repository, pullRequestId, httpUrl, sshUrl, nil
		}
	case "github":
		if len(pathDirs) == 0 {
			// Case is working with a whole VCS
			return vcsUrl, namespace, repository, "", "", "", nil
		} else if len(pathDirs) == 1 {
			// Case is working with a whole project
			namespace = pathDirs[0]
			return vcsUrl, namespace, repository, "", "", "", nil
		} else if len(pathDirs) == 2 {
			// Case is working with a certain repo
			namespace = pathDirs[0]
			repository = pathDirs[1]
			httpUrl = fmt.Sprintf("https://%s/%s/%s.git", vcsUrl, namespace, repository)
			sshUrl = fmt.Sprintf("ssh://git@%s/%s/%s.git", vcsUrl, namespace, repository)
			return vcsUrl, namespace, repository, "", httpUrl, sshUrl, nil
		}
	case "gitlab":
		// Only case with certain repo supported for now
		if len(pathDirs) < 2 {
			return "", "", "", "", "", "", fmt.Errorf("unsupported format of gitlab url for %s", VCSPlugName)
		}
		namespace = path.Join(pathDirs[0 : len(pathDirs)-1]...)
		repository = pathDirs[len(pathDirs)-1]
		httpUrl = fmt.Sprintf("https://%s/%s/%s.git", vcsUrl, namespace, repository)
		// sshUrl = fmt.Sprintf("ssh://git@%s/%s/%s.git", vcsUrl, namespace, repository)
		sshUrl = fmt.Sprintf("git@%s:%s/%s.git", vcsUrl, namespace, repository)
		// sshUrl = fmt.Sprintf("ssh://git@%s:%s/%s.git", vcsUrl, namespace, repository)
		return vcsUrl, namespace, repository, pullRequestId, httpUrl, sshUrl, nil
	default:
		return "", "", "", "", "", "", fmt.Errorf("unsupported VCS plugin name: %s", VCSPlugName)
	}

	return "", "", "", "", "", "", fmt.Errorf("invalid URL: %s", Url)
}

func ContainsSubstring(target string, substrings []string) bool {
	for _, substring := range substrings {
		if strings.Contains(target, substring) {
			return true
		}
	}
	return false
}

func loadTemplateFromFile(filename string) (*template.Template, error) {
	templateData, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("comment").Parse(string(templateData))
	if err != nil {
		return nil, err
	}
	return tpl, nil
}

func CommentBuilder(data interface{}, path string) (string, error) {

	template, err := loadTemplateFromFile(path)
	if err != nil {
		return "", err
	}

	var result bytes.Buffer
	if err := template.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}
