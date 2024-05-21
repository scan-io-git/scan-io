package git

// import (
// 	"io/ioutil"
// 	"os"
// 	"testing"

// 	"github.com/scan-io-git/scan-io/pkg/shared/config"
// 	"github.com/scan-io-git/scan-io/pkg/shared/logger"
// )

// var TestAppConfig *config.Config

// func skipCI(t *testing.T) {
// 	if os.Getenv("CI") != "" {
// 		t.Skip("Skipping testing in CI environment")
// 	}
// }

// func TestGitlabClonePrivateWithSSHAgent(t *testing.T) {

// 	skipCI(t)

// 	// temp dir for fetching
// 	dir, err := ioutil.TempDir("", "TestGitlabClonePrivateWithSSHAgent")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer os.RemoveAll(dir)

// 	// prep args
// 	args := VCSFetchRequest{
// 		CloneURL:     "git@gitlab.com:scanio-demo/juice-shop.git",
// 		Branch:       "",
// 		AuthType:     "ssh-key",
// 		SSHKey:       "~/.ssh/id_rsa",
// 		TargetFolder: dir,
// 	}
// 	vars := EvnVariables{}
// 	logger := logger.NewLogger(TestAppConfig, "test")

// 	// function check
// 	_, err = GitClone(args, vars, logger)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGithubClonePublic(t *testing.T) {

// 	// temp dir for fetching
// 	dir, err := ioutil.TempDir("", "TestGithubClonePublic")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer os.RemoveAll(dir)

// 	// prep args
// 	args := VCSFetchRequest{
// 		CloneURL:     "https://github.com/gin-gonic/gin.git",
// 		Branch:       "",
// 		AuthType:     "http",
// 		SSHKey:       "",
// 		TargetFolder: dir,
// 	}
// 	vars := EvnVariables{}
// 	logger := logger.NewLogger(TestAppConfig, "test")

// 	// function check
// 	_, err = GitClone(args, vars, logger)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGithubClonePrivateWithSSHKey(t *testing.T) {

// 	skipCI(t)

// 	// temp dir for fetching
// 	dir, err := ioutil.TempDir("", "TestGithubClonePrivateWithSSHKey")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer os.RemoveAll(dir)

// 	// prep args
// 	args := VCSFetchRequest{
// 		CloneURL:     "git@github.com:scan-io-git/scan-io.git",
// 		Branch:       "",
// 		AuthType:     "ssh-key",
// 		SSHKey:       "~/.ssh/id_ed25519",
// 		TargetFolder: dir,
// 	}
// 	vars := EvnVariables{}
// 	logger := logger.NewLogger(TestAppConfig, "test")

// 	// function check
// 	_, err = GitClone(args, vars, logger)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGithubClonePrivateWithSSHAgent(t *testing.T) {

// 	skipCI(t)

// 	// temp dir for fetching
// 	dir, err := ioutil.TempDir("", "TestGithubClonePrivateWithSSHAgent")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer os.RemoveAll(dir)

// 	// prep args
// 	args := VCSFetchRequest{
// 		CloneURL:     "git@github.com:scan-io-git/scan-io.git",
// 		Branch:       "",
// 		AuthType:     "ssh-agent",
// 		SSHKey:       "",
// 		TargetFolder: dir,
// 	}
// 	vars := EvnVariables{}
// 	logger := logger.NewLogger(TestAppConfig, "test")

// 	// function check
// 	_, err = GitClone(args, vars, logger)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }
