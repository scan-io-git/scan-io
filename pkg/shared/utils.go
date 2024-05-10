package shared

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hashicorp/go-hclog"

	crssh "golang.org/x/crypto/ssh"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
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

// expandPath resolves paths that include a tilde (~) to the user's home directory.
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, path[2:]), nil
	}
	return path, nil
}

func getGitAuth(args *VCSFetchRequest, variables *EvnVariables, logger hclog.Logger) (transport.AuthMethod, error) {
	var auth transport.AuthMethod
	var err error

	switch args.AuthType {
	case "ssh-key":
		logger.Debug("Setting up SSH key authentication")

		// Handle paths starting with tilde, like ~/.ssh/id_rsa
		// https://gist.github.com/miguelmota/9ab72c5e342f833123c0b5cfd5aca468?permalink_comment_id=3953465#gistcomment-3953465
		if strings.HasPrefix(args.SSHKey, "~/") {
			dirname, _ := os.UserHomeDir()
			args.SSHKey = filepath.Join(dirname, args.SSHKey[2:])
		}

		if _, err := os.Stat(args.SSHKey); err != nil {
			logger.Error("Reading file with a key is failed ", "path", args.SSHKey, "error", err.Error())
			return nil, err
		}

		auth, err = ssh.NewPublicKeysFromFile("git", args.SSHKey, variables.SshKeyPassword)
		if err != nil {
			logger.Error("An extraction publickeys process is failed", "error", err.Error())
			return nil, err
		}

		auth.(*ssh.PublicKeys).HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
			HostKeyCallback: crssh.InsecureIgnoreHostKey(),
		}

	case "ssh-agent":
		logger.Debug("Setting up SSH agent authentication")

		auth, err = ssh.NewSSHAgentAuth("git")
		if err != nil {
			logger.Error("Failed to create SSH agent auth", "err", err)
			return nil, err
		}

		auth.(*ssh.PublicKeysCallback).HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
			HostKeyCallback: crssh.InsecureIgnoreHostKey(),
		}

	case "http":
		logger.Debug("Setting up HTTP authentication")
		auth = &http.BasicAuth{
			Username: variables.Username,
			Password: variables.Token,
		}

	default:
		err := fmt.Errorf("Unknown auth type: %s", args.AuthType)
		logger.Error("Problems with a git fetching process", "error", err)
	}
	return auth, err
}

func GitClone(args VCSFetchRequest, variables EvnVariables, logger hclog.Logger) (string, error) {
	info, err := vcsurl.Parse(args.CloneURL)
	if err != nil {
		logger.Error("Unable to parse VCS url", "VCSURL", args.CloneURL, "err", err)
		return "", err
	}

	branch := plumbing.ReferenceName(args.Branch)
	auth, err := getGitAuth(&args, &variables, logger)
	if err != nil {
		logger.Error("Unable to set up auth", "err", err)
		return "", err
	}

	// debug output from git cli
	output := logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
		ForceLevel:  hclog.Debug,
	})

	gitCloneOptions := &git.CloneOptions{
		Auth:          auth,
		URL:           args.CloneURL,
		ReferenceName: branch,
		Progress:      output,
		Depth:         1,
	}

	gitPullOptions := &git.PullOptions{
		Auth:          auth,
		ReferenceName: branch,
		Progress:      output,
		Depth:         1,
	}

	gitCheckoutOptions := &git.CheckoutOptions{
		Branch: branch,
		Force:  true,
	}

	gitFetchOptions := &git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth,
		Progress:   output,
		RefSpecs:   []config.RefSpec{config.RefSpec("+refs/*:refs/*")},
		Depth:      1,
	}

	logger.Debug("Fetching repo", "repo", info.Name, "branch", args.Branch, "targetFolder", args.TargetFolder)
	_, errClone := git.PlainClone(args.TargetFolder, false, gitCloneOptions)
	if errClone != nil && errClone != git.ErrRepositoryAlreadyExists {
		logger.Error("Error on Clone occured", "err", err, "targetFolder", args.TargetFolder, "remote", gitCloneOptions.URL)
		return "", err
	}

	r, err := git.PlainOpen(args.TargetFolder)
	if err != nil {
		logger.Error("Can't open repository on a disk", "err", err, "targetFolder", args.TargetFolder)
		return "", err
	}

	_ = r.Fetch(gitFetchOptions) // updating list of references

	w, err := r.Worktree()
	if err != nil {
		logger.Error("Error on Worktree occured", "err", err, "targetFolder", args.TargetFolder)
		return "", err
	}

	logger.Debug("Checkout a branch", "repo", info.Name, "targetFolder", args.TargetFolder, "branch", args.Branch)
	if err = w.Checkout(gitCheckoutOptions); err != nil {
		logger.Error("Error on Checkout occured", "err", err, "targetFolder", args.TargetFolder)
		return "", err
	}

	logger.Debug("Reseting local repo", "repo", info.Name, "targetFolder", args.TargetFolder)
	if err := w.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
		fmt.Println("Error on Checkout occured", "err", err, "targetFolder", args.TargetFolder)
		return "", err
	}

	if errClone != nil && errClone == git.ErrRepositoryAlreadyExists {
		logger.Debug("Pulling repo", "repo", info.Name, "targetFolder", args.TargetFolder, "branch", args.Branch)
		if err = w.Pull(gitPullOptions); err != nil {
			logger.Error("Error on Pull occured", "err", err, "targetFolder", args.TargetFolder)
			return "", err
		}
	}

	logger.Info("A fetch function finished", "repo", info.Name, "branch", args.Branch, "targetFolder", args.TargetFolder)
	return args.TargetFolder, nil
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

func IsCI() bool {
	if os.Getenv("CI") == "true" {
		return true
	}

	if os.Getenv("SCANIO_MODE") == "CI" {
		return true
	}

	return false
}

func Copy(srcPath, destPath string) error {
	srcInfo, err := os.Lstat(srcPath)
	if err != nil {
		return err
	}

	// Check if the source is a directory
	if srcInfo.IsDir() {
		return CopyDir(srcPath, destPath)
	}

	// Check if the source is a symlink
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return CopySymLink(srcPath, destPath)
	}

	// Assume the source is a regular file if not a directory or symlink
	return CopyFile(srcPath, destPath)
}

func CopyFile(srcFile, destFile string) error {
	destDir := filepath.Dir(destFile)
	if err := CreateIfNotExists(destDir, os.ModePerm); err != nil {
		return err
	}

	in, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func CopyDir(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	if err := CreateIfNotExists(destDir, os.ModePerm); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		if err := Copy(srcPath, destPath); err != nil {
			return err
		}
	}

	return nil
}

func CopySymLink(srcLink, destLink string) error {
	linkTarget, err := os.Readlink(srcLink)
	if err != nil {
		return err
	}

	return os.Symlink(linkTarget, destLink)
}

func CreateIfNotExists(path string, perm os.FileMode) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, perm)
	}
	return nil
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
