package shared

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestGithubClonePublic(t *testing.T) {

	// temp dir for fetching
	dir, err := ioutil.TempDir("", "TestGithubClonePublic")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	// prep args
	args := VCSFetchRequest{
		CloneURL:     "https://github.com/gin-gonic/gin.git",
		Branch:       "",
		AuthType:     "http",
		SSHKey:       "",
		TargetFolder: dir,
	}
	vars := EvnVariables{}
	logger := NewLogger("test")

	// function check
	err = GitClone(args, vars, logger)
	if err != nil {
		t.Error(err)
	}
}

func TestGithubClonePrivateWithSSHKey(t *testing.T) {

	// temp dir for fetching
	dir, err := ioutil.TempDir("", "TestGithubClonePrivateWithSSHKey")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	// prep args
	args := VCSFetchRequest{
		CloneURL:     "git@github.com:scan-io-git/scan-io.git",
		Branch:       "",
		AuthType:     "ssh-key",
		SSHKey:       "~/.ssh/id_ed25519",
		TargetFolder: dir,
	}
	vars := EvnVariables{}
	logger := NewLogger("test")

	// function check
	err = GitClone(args, vars, logger)
	if err != nil {
		t.Error(err)
	}
}

func TestGithubClonePrivateWithSSHAgent(t *testing.T) {

	// temp dir for fetching
	dir, err := ioutil.TempDir("", "TestGithubClonePrivateWithSSHAgent")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	// prep args
	args := VCSFetchRequest{
		CloneURL:     "git@github.com:scan-io-git/scan-io.git",
		Branch:       "",
		AuthType:     "ssh-agent",
		SSHKey:       "",
		TargetFolder: dir,
	}
	vars := EvnVariables{}
	logger := NewLogger("test")

	// function check
	err = GitClone(args, vars, logger)
	if err != nil {
		t.Error(err)
	}
}
