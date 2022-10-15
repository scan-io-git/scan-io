package main

import (
	"context"
	"encoding/json"
	"fmt"

	//"net/url"
	"os"
	//"path/filepath"
	//"strings"
	"time"

	crssh "golang.org/x/crypto/ssh"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	//bitbucketv2 "github.com/ktrysmt/go-bitbucket"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/mitchellh/mapstructure"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/library/vcs"
	"github.com/scan-io-git/scan-io/shared"
)

// Global variables for the plugin
var (
	username, token, vcsPort, sshKeyPassword string
)

type VCSBitbucket struct {
	logger hclog.Logger
}

// Limit for Bitbucket v1 API page resonse
var opts = map[string]interface{}{
	"limit": 2000,
	"start": 0,
}

func getProjectsResponse(r *bitbucketv1.APIResponse) ([]bitbucketv1.Project, error) {
	var m []bitbucketv1.Project
	err := mapstructure.Decode(r.Values["values"], &m)
	return m, err
}

// Init function for checking an environment
func (g *VCSBitbucket) init(command string) {
	username = os.Getenv("BITBUCKET_USERNAME")
	token = os.Getenv("BITBUCKET_TOKEN")

	if len(username) == 0 {
		g.logger.Error("BITBUCKET_USERNAME or BITBUCKET_TOKEN is not provided in an environment.")
		panic("Env problems")
	} else if len(token) == 0 {
		g.logger.Error("BITBUCKET_USERNAME or BITBUCKET_TOKEN is not provided in an environment.")
		panic("Env problems")
	}
	if command == "fetch" {
		vcsPort = os.Getenv("BITBUCKET_SSH_PORT")
		sshKeyPassword = os.Getenv("BITBUCKET_SSH_KEY_PASSWORD")

		if len(vcsPort) == 0 {
			g.logger.Warn("BITBUCKET_SSH_PORT is not provided in an environment. Using default 22 ssh port")
			vcsPort = "22"
		}
		if len(sshKeyPassword) == 0 {
			g.logger.Warn("BITBUCKET_SSH_KEY_PASSOWRD is empty or not provided.")
		}
	}
}

// Listing all project in Bitbucket v1 API
func (g *VCSBitbucket) listAllProjects(client *bitbucketv1.APIClient) ([]vcs.ProjectParams, error) {
	g.logger.Debug("Listing all projects")
	response, err := client.DefaultApi.GetProjects(opts)
	if err != nil {
		g.logger.Error("Listing projects is failed")
		g.logger.Error("Listing projects error", "err", err)
		return nil, err
	}

	g.logger.Debug("Projects is listed")
	res, err := getProjectsResponse(response)
	if err != nil {
		g.logger.Error("Response parsing is failed")
		panic(err.Error())
	}

	var projectsList []vcs.ProjectParams
	for _, bitbucketRepo := range res {
		projectsList = append(projectsList, vcs.ProjectParams{Key: bitbucketRepo.Key, Name: bitbucketRepo.Name, Link: bitbucketRepo.Links.Self[0].Href})

	}

	g.logger.Info("List of projects is ready")
	resultJson, _ := json.MarshalIndent(projectsList, "", "    ")
	g.logger.Debug(string(resultJson))

	return projectsList, nil
}

// Resolving information about all repositories in a one project from Bitbucket v1 API
func (g *VCSBitbucket) resolveOneProject(client *bitbucketv1.APIClient, project string) ([]vcs.RepositoryParams, error) {
	g.logger.Debug("Resolving a particular project", "project", project)
	response, err := client.DefaultApi.GetRepositoriesWithOptions(project, opts)
	if err != nil {
		g.logger.Error("Resolving is failed")
		//panic(err.Error())
		return nil, err
	}

	g.logger.Debug("Project is resolved", "project", project)
	result, err := bitbucketv1.GetRepositoriesResponse(response)
	if err != nil {
		g.logger.Error("Response parsing is failed")
		panic(err.Error())
	}

	var resultList []vcs.RepositoryParams
	for _, repo := range result {
		var http_link string
		var ssh_link string

		for _, clone_links := range repo.Links.Clone {

			if clone_links.Name == "http" {
				http_link = clone_links.Href
			} else if clone_links.Name == "ssh" {
				ssh_link = clone_links.Href
			} else {
				continue
			}
		}

		resultList = append(resultList, vcs.RepositoryParams{Name: repo.Name, HttpLink: http_link, SshLink: ssh_link})
	}

	g.logger.Info("List of repositories is ready.")
	resultJson, _ := json.MarshalIndent(resultList, "", "    ")
	g.logger.Debug(string(resultJson))

	return resultList, nil
}

func (g *VCSBitbucket) ListReposRunner(args shared.VCSListReposRequest) ([]vcs.RepositoryParams, error) {
	g.logger.Debug("Entering ListRepos", "args", args)
	g.init("list")

	baseURL := fmt.Sprintf("https://%s/rest", args.VCSURL)
	basicAuth := bitbucketv1.BasicAuth{UserName: username, Password: token}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	ctx = context.WithValue(ctx, bitbucketv1.ContextBasicAuth, basicAuth)
	defer cancel()

	client := bitbucketv1.NewAPIClient(
		ctx,
		bitbucketv1.NewConfiguration(baseURL),
	)

	var repositories []vcs.RepositoryParams
	if len(args.Namespace) != 0 {
		g.logger.Info("Resolving a project")
		oneProjectData, err := g.resolveOneProject(client, args.Namespace)
		if err != nil {
			g.logger.Error("222222222")
			g.logger.Error("Listing all repos error", "err", err)
			return nil, err
		}
		for _, repo := range oneProjectData {
			// parsedUrl, err := url.Parse(repo.SshLink)
			// if err != nil {
			// 	panic(err)
			// }
			// path := strings.TrimSuffix(parsedUrl.Path, filepath.Ext(parsedUrl.Path))
			repositories = append(repositories, repo)
		}

	} else {
		g.logger.Info("Listing all repos in all projects")
		projectsList, err := g.listAllProjects(client)
		if err != nil {
			g.logger.Error("222222222")
			g.logger.Error("Listing all repos error", "err", err)
			return nil, err
		}

		for _, projectName := range projectsList {
			oneProjectData, err := g.resolveOneProject(client, projectName.Key)
			if err != nil {
				g.logger.Error("222222222")
				g.logger.Error("Listing all repos error", "err", err.Error())
				return nil, err
			}
			for _, repo := range oneProjectData {
				// parsedUrl, err := url.Parse(repo.SshLink)
				// if err != nil {
				// 	panic(err)
				// }
				// path := strings.TrimSuffix(parsedUrl.Path, filepath.Ext(parsedUrl.Path))

				repositories = append(repositories, repo)
			}
		}

	}

	return repositories, nil
}

func (g *VCSBitbucket) ListRepos(args shared.VCSListReposRequest) vcs.ListFuncResult {
	g.logger.Debug("Entering ListRepos", "args", args)
	g.init("list")

	repositories, err := g.ListReposRunner(args)
	if err != nil {
		g.logger.Debug("Cal")
		return vcs.ListFuncResult{Result: nil, Status: "FAILED", Message: err.Error()}
	}

	// g.logger.Info("End")
	// resultJson, _ := json.MarshalIndent(result, "", "    ")
	// g.logger.Debug(string(resultJson))

	return vcs.ListFuncResult{Result: repositories, Status: "OK", Message: ""}
}

func (g *VCSBitbucket) Fetch(args shared.VCSFetchRequest) bool {
	g.init("fetch")

	info, err := vcsurl.Parse(fmt.Sprintf("https://%s/%s", args.VCSURL, args.Project))
	if err != nil {
		g.logger.Error("Unable to parse VCS url info", "VCSURL", args.VCSURL, "project", args.Project)
		panic(err)
	}

	gitCloneOptions := &git.CloneOptions{
		Progress: os.Stdout,
		Depth:    1,
	}

	gitCloneOptions.URL = fmt.Sprintf("git@%s:%s%s.git", info.Host, vcsPort, info.FullName)

	if args.AuthType == "ssh-key" {
		g.logger.Info("Making arrangements for ssh-key fetching", "repo", args.Project)
		_, err := os.Stat(args.SSHKey)
		if err != nil {
			g.logger.Error("read file %s failed %s\n", args.SSHKey, err.Error())
			panic(err)
		}

		pkCallback, err := ssh.NewPublicKeysFromFile("git", args.SSHKey, sshKeyPassword)
		if err != nil {
			g.logger.Error("generate publickeys failed: %s\n", err.Error())
			panic(err)
		}

		pkCallback.HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
			HostKeyCallback: crssh.InsecureIgnoreHostKey(),
		}

		gitCloneOptions.Auth = pkCallback
	} else if args.AuthType == "ssh-agent" {
		g.logger.Info("Making arrangements for ssh-agent fetching", "repo", args.Project)
		pkCallback, err := ssh.NewSSHAgentAuth("git")
		if err != nil {
			g.logger.Error("NewSSHAgentAuth error", "err", err)
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
			Username: username,
			Password: token,
		}
	} else {
		g.logger.Debug("Unknown auth type")
		panic("Unknown auth type")
	}

	//TODO add logging from go-git
	g.logger.Info("Fetching repo", "repo", args.Project)
	_, err = git.PlainClone(args.TargetFolder, false, gitCloneOptions)

	if err != nil {
		g.logger.Info("Error on Clone occured", "err", err, "targetFolder", args.TargetFolder, "remote", gitCloneOptions.URL)
		return false
	}

	g.logger.Info("Fetch's ended", "repo", args.Project)
	return true
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	VCS := &VCSBitbucket{
		logger: logger,
	}

	var pluginMap = map[string]plugin.Plugin{
		shared.PluginTypeVCS: &shared.VCSPlugin{Impl: VCS},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
