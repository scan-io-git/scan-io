package vcs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"log"
	"os"

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
	logger.Info("Results saved to file", outputFile)

}

func GitClone(args VCSFetchRequest, variables EvnVariables, logger hclog.Logger) (bool, error) {

	info, err := vcsurl.Parse(fmt.Sprintf("https://%s/%s", args.VCSURL, args.Repository))
	if err != nil {
		logger.Error("Unable to parse VCS url info", "VCSURL", args.VCSURL, "project", args.Repository)

	}

	gitCloneOptions := &git.CloneOptions{
		Progress: os.Stdout,
		Depth:    1,
	}

	gitCloneOptions.URL = fmt.Sprintf("git@%s:%s%s.git", info.Host, variables.VcsPort, info.FullName)

	if args.AuthType == "ssh-key" {
		logger.Info("Making arrangements for ssh-key fetching", "repo", args.Repository)
		_, err := os.Stat(args.SSHKey)
		if err != nil {
			logger.Error("read file %s failed %s\n", args.SSHKey, err.Error())
			panic(err)
		}

		pkCallback, err := ssh.NewPublicKeysFromFile("git", args.SSHKey, variables.SshKeyPassword)
		if err != nil {
			logger.Error("generate publickeys failed: %s\n", err.Error())
			panic(err)
		}

		pkCallback.HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
			HostKeyCallback: crssh.InsecureIgnoreHostKey(),
		}

		gitCloneOptions.Auth = pkCallback
	} else if args.AuthType == "ssh-agent" {
		//g.logger.Info("Making arrangements for ssh-agent fetching", "repo", args.Project)
		pkCallback, err := ssh.NewSSHAgentAuth("git")
		if err != nil {
			logger.Error("NewSSHAgentAuth error", "err", err)
			panic(err)
		}

		pkCallback.HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
			HostKeyCallback: crssh.InsecureIgnoreHostKey(),
		}

		gitCloneOptions.Auth = pkCallback

	} else if args.AuthType == "http" {
		//gitCloneOptions.URL, _ = info.Remote(vcsurl.HTTPS)
		gitCloneOptions.URL = fmt.Sprintf("https://%s/scm%s.git", info.Host, info.FullName)

		gitCloneOptions.Auth = &http.BasicAuth{
			Username: variables.Username,
			Password: variables.Token,
		}
	} else {
		logger.Debug("Unknown auth type")
		panic("Unknown auth type")
	}

	//TODO add logging from go-git
	logger.Info("Fetching repo", "repo", args.Repository)
	_, err = git.PlainClone(args.TargetFolder, false, gitCloneOptions)

	if err != nil {
		logger.Info("Error on Clone occured", "err", err, "targetFolder", args.TargetFolder, "remote", gitCloneOptions.URL)
		return false, err
	}

	logger.Info("Fetch's ended", "repo", args.Repository)
	return true, nil
}
