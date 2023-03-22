package shared

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"

	crssh "golang.org/x/crypto/ssh"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
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
		logger.Error("Unable to parse VCS url info", "VCSURL", args.CloneURL)
		return err
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

	if args.AuthType == "ssh-key" {
		logger.Info("Making arrangements for an ssh-key fetching", "repo", info.Name)
		if _, err := os.Stat(args.SSHKey); err != nil {
			logger.Error("read file %s failed %s\n", args.SSHKey, err.Error())
			return err
		}

		pkCallback, err := ssh.NewPublicKeysFromFile("git", args.SSHKey, variables.SshKeyPassword)
		if err != nil {
			logger.Error("An extraction publickeys process is failed: %s\n", err.Error())
			return err
		}

		pkCallback.HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
			HostKeyCallback: crssh.InsecureIgnoreHostKey(),
		}

		gitCloneOptions.Auth, gitPullOptions.Auth = pkCallback, pkCallback
	} else if args.AuthType == "ssh-agent" {
		logger.Info("Making arrangements for an ssh-agent fetching", "repo", info.Name)
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
		logger.Error("Problems with the a git fetching process", "error", err)
		return err
	}

	logger.Info("Fetching repo", "repo", info.Name)
	_, err = git.PlainClone(args.TargetFolder, false, gitCloneOptions)
	if err != nil && err == git.ErrRepositoryAlreadyExists {
		//git checkout - check deleted files
		logger.Info("Repository is already exists on a disk", "repo", info.Name, "targetFolder", args.TargetFolder)

		r, err := git.PlainOpen(args.TargetFolder)
		if err != nil {
			logger.Info("Can't open repository on a disk", "err", err, "targetFolder", args.TargetFolder)
			return err
		}
		w, err := r.Worktree()
		if err != nil {
			logger.Info("Error on Wroktree occured", "err", err, "targetFolder", args.TargetFolder)
			return err
		}

		logger.Info("Reseting local repo", "repo", info.Name, "targetFolder", args.TargetFolder)
		//git reset --hard origin/master if someone delete files from disk
		if err := w.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
			fmt.Println("Error on Checkout occured", "err", err, "targetFolder", args.TargetFolder)
			return err
		}

		logger.Info("Pulling repo", "repo", info.Name, "targetFolder", args.TargetFolder)
		if err = w.Pull(gitPullOptions); err != nil {
			logger.Info("Error on Pull occured", "err", err, "targetFolder", args.TargetFolder)
			return err
		}
	} else if err != nil {
		logger.Info("Error on Clone occured", "err", err, "targetFolder", args.TargetFolder, "remote", gitCloneOptions.URL)
		return err
	}

	logger.Info("Fetch's ended", "repo", info.Name)
	return nil
}

func ExtractRepositoryInfoFromURL(Url string, VCSPlugName string) (string, string, string, string, string, error) {
	var namespace string
	var repository string
	var pathDirs []string

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
	lastElement := pathDirs[len(pathDirs)-1]
	isHTTP := scheme == "http" || scheme == "https"

	switch VCSPlugName {
	case "bitbucket":
		// The case is for a Bitbucket APIv1 URL format
		// TODO
		// We can move building urls to just calling a list function
		// But bitbucketV1 library can't resolve a particular repo

		if len(pathDirs) == 0 && isHTTP {
			// Case is working with a whole VCS
			return vcsUrl, namespace, repository, Url, "", nil
		} else if len(pathDirs) <= 2 && pathDirs[0] == "projects" && isHTTP {
			// Case is working with a whole project from a Web UI URL format
			// https://bitbucket.com/projects/<project_name>
			namespace = pathDirs[1]
			return vcsUrl, namespace, repository, Url, "", nil
		} else if len(pathDirs) >= 3 && pathDirs[0] == "projects" && pathDirs[2] == "repos" && isHTTP {
			// Case is working with a certain repo form a Web UI URL format
			// https://bitbucket.com/projects/<project_name>/repos/<repo_name>/browse
			namespace = pathDirs[1]
			repository = pathDirs[3]
			httpUrl := fmt.Sprintf("https://%s/scm/%s/%s.git", vcsUrl, namespace, repository)
			sshUrl := fmt.Sprintf("ssh://git@%s:7989/%s/%s.git", vcsUrl, namespace, repository)
			return vcsUrl, namespace, repository, httpUrl, sshUrl, nil
		} else if isHTTP && pathDirs[0] == "scm" && strings.Contains(lastElement, ".git") {
			// https://bitbucket.com/scm/<project_name>/<repo_name>.git
			namespace = pathDirs[1]
			repository = strings.TrimSuffix(pathDirs[2], ".git")
			httpUrl := fmt.Sprintf("https://%s/scm/%s/%s.git", vcsUrl, namespace, repository)
			sshUrl := fmt.Sprintf("ssh://git@%s:7989/%s/%s.git", vcsUrl, namespace, repository)
			return vcsUrl, namespace, repository, httpUrl, sshUrl, nil
		} else if scheme == "ssh" && strings.Contains(lastElement, ".git") {
			// ssh://git@bitbucket.com:7989/<project_name>/<repo_name>.git
			namespace = pathDirs[0]
			repository = strings.TrimSuffix(pathDirs[1], ".git")
			httpUrl := fmt.Sprintf("https://%s/scm/%s/%s.git", vcsUrl, namespace, repository) //
			sshUrl := fmt.Sprintf("ssh://git@%s:7989/%s/%s.git", vcsUrl, namespace, repository)
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
			return vcsUrl, namespace, repository, "", "", nil
		}
	case "gitlab":
		//TODO
		return "", "", "", "", "", fmt.Errorf("Gitlab s unsupported for work with an URL: %s", VCSPlugName)
	default:
		return "", "", "", "", "", fmt.Errorf("unsupported VCS plugin name: %s", VCSPlugName)
	}

	return "", "", "", "", "", fmt.Errorf("invalid URL: %s", Url)
}
