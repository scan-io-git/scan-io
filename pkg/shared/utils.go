package shared

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"

	crssh "golang.org/x/crypto/ssh"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func WriteJsonFile(data ListFuncResult, outputFile string, logger hclog.Logger) {
	file, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer file.Close()

	datawriter := bufio.NewWriter(file)
	defer datawriter.Flush()

	resultJson, _ := json.MarshalIndent(data, "", "    ")
	datawriter.Write(resultJson)
	logger.Info("Results saved to file", "path", outputFile)

}

func GitClone(args VCSFetchRequest, variables EvnVariables, logger hclog.Logger) error {
	info, err := vcsurl.Parse(args.CloneURL)
	if err != nil {
		logger.Error("Unable to parse VCS url", "VCSURL", args.CloneURL)
		return err
	}

	// handle paths starting with tilda, like ~/.ssh/id_rsa
	// https://gist.github.com/miguelmota/9ab72c5e342f833123c0b5cfd5aca468?permalink_comment_id=3953465#gistcomment-3953465
	SSHKey := args.SSHKey
	if strings.HasPrefix(SSHKey, "~/") {
		dirname, _ := os.UserHomeDir()
		SSHKey = filepath.Join(dirname, SSHKey[2:])
	}

	//debug output from git cli
	output := logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
		ForceLevel:  hclog.Debug,
	})

	gitCloneOptions := &git.CloneOptions{
		Progress: output,
		Depth:    1,
	}
	gitPullOptions := &git.PullOptions{
		Progress: output,
		Depth:    1,
	}

	gitCloneOptions.URL = args.CloneURL
	if args.Branch != "" {
		referenceNamePrefix := "refs/heads/%s"
		customBranch := plumbing.ReferenceName(fmt.Sprintf(referenceNamePrefix, args.Branch))
		gitCloneOptions.ReferenceName = customBranch
		gitPullOptions.ReferenceName = customBranch
	}

	if args.AuthType == "ssh-key" {
		logger.Debug("Making arrangements for an ssh-key fetching", "repo", info.Name, "branch", args.Branch)
		if _, err := os.Stat(SSHKey); err != nil {
			logger.Error("Reading file with a key is failed ", "path", SSHKey, "error", err.Error())
			return err
		}

		pkCallback, err := ssh.NewPublicKeysFromFile("git", SSHKey, variables.SshKeyPassword)
		if err != nil {
			logger.Error("An extraction publickeys process is failed", "error", err.Error())
			return err
		}

		pkCallback.HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
			HostKeyCallback: crssh.InsecureIgnoreHostKey(),
		}

		gitCloneOptions.Auth, gitPullOptions.Auth = pkCallback, pkCallback
	} else if args.AuthType == "ssh-agent" {
		logger.Debug("Making arrangements for an ssh-agent fetching", "repo", info.Name, "branch", args.Branch)
		pkCallback, err := ssh.NewSSHAgentAuth("git")
		if err != nil {
			logger.Error("NewSSHAgentAuth error", "err", err)
			return err
		}

		pkCallback.HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
			HostKeyCallback: crssh.InsecureIgnoreHostKey(),
		}

		gitCloneOptions.Auth, gitPullOptions.Auth = pkCallback, pkCallback

	} else if args.AuthType == "http" {
		basicAuth := &http.BasicAuth{
			Username: variables.Username,
			Password: variables.Token,
		}
		gitCloneOptions.Auth, gitPullOptions.Auth = basicAuth, basicAuth
	} else {
		err := fmt.Errorf("Unknown auth type")
		logger.Error("Problems with a git fetching process", "error", err)
		return err
	}

	logger.Debug("Fetching repo", "repo", info.Name, "branch", args.Branch, "targetFolder", args.TargetFolder)
	_, err = git.PlainClone(args.TargetFolder, false, gitCloneOptions)
	if err != nil && err == git.ErrRepositoryAlreadyExists {
		//git checkout - check deleted files
		logger.Warn("Repository is already exists on a disk", "repo", info.Name, "targetFolder", args.TargetFolder)

		r, err := git.PlainOpen(args.TargetFolder)
		if err != nil {
			logger.Error("Can't open repository on a disk", "err", err, "targetFolder", args.TargetFolder)
			return err
		}
		w, err := r.Worktree()
		if err != nil {
			logger.Error("Error on Worktree occured", "err", err, "targetFolder", args.TargetFolder)
			return err
		}

		logger.Debug("Reseting local repo", "repo", info.Name, "targetFolder", args.TargetFolder)
		//git reset --hard origin/master if someone delete files from disk
		if err := w.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
			fmt.Println("Error on Checkout occured", "err", err, "targetFolder", args.TargetFolder)
			return err
		}

		logger.Debug("Pulling repo", "repo", info.Name, "targetFolder", args.TargetFolder, "branch", args.Branch)
		if err = w.Pull(gitPullOptions); err != nil {
			logger.Error("Error on Pull occured", "err", err, "targetFolder", args.TargetFolder)
			return err
		}
	} else if err != nil {
		logger.Error("Error on Clone occured", "err", err, "targetFolder", args.TargetFolder, "remote", gitCloneOptions.URL)
		return err
	}

	logger.Info("A fetch function finished", "repo", info.Name, "branch", args.Branch, "targetFolder", args.TargetFolder)
	return nil
}

func ExtractRepositoryInfoFromURL(Url string, VCSPlugName string) (string, string, string, string, string, error) {
	var namespace string
	var repository string
	var lastElement string
	var pathDirs []string
	var httpUrl string
	var sshUrl string

	u, err := url.ParseRequestURI(Url)
	if err != nil {
		return "", "", "", "", "", err
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
			return vcsUrl, namespace, repository, Url, "", nil
		} else if len(pathDirs) == 2 && pathDirs[0] == "projects" && isHTTP {
			// Case is working with a whole project from a Web UI URL format
			// https://bitbucket.com/projects/<project_name>
			namespace = pathDirs[1]
			return vcsUrl, namespace, repository, Url, "", nil
		} else if len(pathDirs) > 3 && pathDirs[0] == "projects" && pathDirs[2] == "repos" && isHTTP {
			// Case is working with a certain repo form a Web UI URL format
			// https://bitbucket.com/projects/<project_name>/repos/<repo_name>/browse
			namespace = pathDirs[1]
			repository = pathDirs[3]
			httpUrl := fmt.Sprintf("https://%s/scm/%s/%s.git", vcsUrl, namespace, repository)
			sshUrl := fmt.Sprintf("ssh://git@%s:7989/%s/%s.git", vcsUrl, namespace, repository)
			return vcsUrl, namespace, repository, httpUrl, sshUrl, nil
		} else if len(pathDirs) >= 2 && isHTTP && pathDirs[0] == "scm" {
			// https://bitbucket.com/scm/<project_name>/
			namespace = pathDirs[1]
			if strings.Contains(lastElement, ".git") {
				// https://bitbucket.com/scm/<project_name>/<repo_name>.git
				repository = strings.TrimSuffix(lastElement, ".git")
				httpUrl = fmt.Sprintf("https://%s/scm/%s/%s.git", vcsUrl, namespace, repository)
				sshUrl = fmt.Sprintf("ssh://git@%s:7989/%s/%s.git", vcsUrl, namespace, repository)
			}
			return vcsUrl, namespace, repository, httpUrl, sshUrl, nil
		} else if scheme == "ssh" {
			namespace = pathDirs[0]
			if strings.Contains(lastElement, ".git") {
				// ssh://git@bitbucket.com:7989/<project_name>/<repo_name>.git
				port := u.Port()
				repository = strings.TrimSuffix(lastElement, ".git")
				httpUrl = fmt.Sprintf("https://%s/scm/%s/%s.git", vcsUrl, namespace, repository)
				// User can override a port if he use an ssh scheme format of URL
				sshUrl = fmt.Sprintf("ssh://git@%s:%s/%s/%s.git", vcsUrl, port, namespace, repository)
			}
			return vcsUrl, namespace, repository, httpUrl, sshUrl, nil
		}
	case "github":
		if len(pathDirs) == 0 {
			// Case is working with a whole VCS
			return vcsUrl, namespace, repository, "", "", nil
		} else if len(pathDirs) == 1 {
			// Case is working with a whole project
			namespace = pathDirs[0]
			return vcsUrl, namespace, repository, "", "", nil
		} else if len(pathDirs) == 2 {
			// Case is working with a certain repo
			namespace = pathDirs[0]
			repository = pathDirs[1]
			httpUrl = fmt.Sprintf("https://%s/%s/%s.git", vcsUrl, namespace, repository)
			sshUrl = fmt.Sprintf("ssh://git@%s/%s/%s.git", vcsUrl, namespace, repository)
			return vcsUrl, namespace, repository, httpUrl, sshUrl, nil
		}
	case "gitlab":
		// Only case with certain repo supported for now
		if len(pathDirs) < 2 {
			return "", "", "", "", "", fmt.Errorf("unsupported format of gitlab url for %s", VCSPlugName)
		}
		namespace = path.Join(pathDirs[0 : len(pathDirs)-1]...)
		repository = pathDirs[len(pathDirs)-1]
		httpUrl = fmt.Sprintf("https://%s/%s/%s.git", vcsUrl, namespace, repository)
		// sshUrl = fmt.Sprintf("ssh://git@%s/%s/%s.git", vcsUrl, namespace, repository)
		sshUrl = fmt.Sprintf("git@%s:%s/%s.git", vcsUrl, namespace, repository)
		// sshUrl = fmt.Sprintf("ssh://git@%s:%s/%s.git", vcsUrl, namespace, repository)
		return vcsUrl, namespace, repository, httpUrl, sshUrl, nil
	default:
		return "", "", "", "", "", fmt.Errorf("unsupported VCS plugin name: %s", VCSPlugName)
	}

	return "", "", "", "", "", fmt.Errorf("invalid URL: %s", Url)
}
