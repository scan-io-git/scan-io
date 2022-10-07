package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	//bitbucketv2 "github.com/ktrysmt/go-bitbucket"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/shared"
	"github.com/scan-io-git/scan-io/utils"
)

// Glogal variables for the plugin
var (
	username, token, vcsPort string
)

// Here is a real implementation of VCS
type VCSBitbucket struct {
	logger hclog.Logger
}

// Limit for Bitbucket v1 API page resonse
var opts = map[string]interface{}{
	"limit": 2000,
	"start": 0,
}

// Init function for checking an environment
func (g *VCSBitbucket) init(command string) {
	username := os.Getenv("BITBUCKET_USERNAME")
	token := os.Getenv("BITBUCKET_USERNAME")

	if len(username) == 0 {
		g.logger.Error("BITBUCKET_USERNAME or BITBUCKET_TOKEN is not provided in an environment.")
		panic("Env problems")
	} else if len(token) == 0 {
		g.logger.Error("BITBUCKET_USERNAME or BITBUCKET_TOKEN is not provided in an environment.")
		panic("Env problems")
	}
	if command == "fetch" {
		portPointer := &vcsPort
		*portPointer = os.Getenv("BITBUCKET_SSH_PORT")
		if len(vcsPort) == 0 {
			g.logger.Warn("BITBUCKET_SSH_PORT is not provided in an environment. Using default 22 ssh port")
			*portPointer = "22"
		}
	}
}

// Listing all project in Bitbucket v1 API
func (g *VCSBitbucket) listAllProjects(client *bitbucketv1.APIClient) []utils.BBProject {
	g.logger.Debug("Listing all projects")
	response, err := client.DefaultApi.GetProjects(opts)
	if err != nil {
		g.logger.Error("Listing projects is failed")
		panic(err.Error())
	}

	g.logger.Debug("Projects is listed")
	res, err := utils.GetProjectsResponse(response)
	if err != nil {
		g.logger.Error("Response parsing is failed")
		panic(err.Error())
	}

	var projectsList []utils.BBProject
	for _, bitbucketRepo := range res {
		projectsList = append(projectsList, utils.BBProject{Key: bitbucketRepo.Key, Link: bitbucketRepo.Links.Self[0].Href})

	}

	g.logger.Info("List of projects is ready")
	resultJson, _ := json.MarshalIndent(projectsList, "", "    ")
	g.logger.Debug(string(resultJson))

	return projectsList
}

// Resolving information about all repositories in a one project from Bitbucket v1 API
func (g *VCSBitbucket) resolveOneProject(client *bitbucketv1.APIClient, project string) []utils.BBReposLinks {
	g.logger.Debug("Resolving a particular project", "project", project)
	response, err := client.DefaultApi.GetRepositoriesWithOptions(project, opts)
	if err != nil {
		g.logger.Error("Resolving is failed")
		panic(err.Error())
	}

	g.logger.Debug("Project is resolved", "project", project)
	result, err := bitbucketv1.GetRepositoriesResponse(response)
	if err != nil {
		g.logger.Error("Response parsing is failed")
		panic(err.Error())
	}

	var resultList []utils.BBReposLinks
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

		resultList = append(resultList, utils.BBReposLinks{Name: repo.Name, HttpLink: http_link, SshLink: ssh_link})
	}

	g.logger.Info("List of repositories is ready.")
	resultJson, _ := json.MarshalIndent(resultList, "", "    ")
	g.logger.Debug(string(resultJson))

	return resultList
}

func (g *VCSBitbucket) ListRepos(args shared.VCSListReposRequest) []string {
	g.logger.Debug("Entering ListRepos", "args", args)
	g.init("list")

	baseURL := fmt.Sprintf("https://%s", args.VCSURL)
	basicAuth := bitbucketv1.BasicAuth{UserName: username, Password: token}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	ctx = context.WithValue(ctx, bitbucketv1.ContextBasicAuth, basicAuth)
	defer cancel()

	client := bitbucketv1.NewAPIClient(
		ctx,
		bitbucketv1.NewConfiguration(baseURL),
	)

	var repositories []string
	if len(args.Namespace) != 0 {
		g.logger.Info("Resolving a project")
		oneProjectData := g.resolveOneProject(client, args.Namespace)
		for _, repo := range oneProjectData {
			parsedUrl, err := url.Parse(repo.SshLink)
			if err != nil {
				panic(err)
			}
			path := strings.TrimSuffix(parsedUrl.Path, filepath.Ext(parsedUrl.Path))
			repositories = append(repositories, path)
		}

	} else {
		g.logger.Info("Listing all repos in all projects")
		projectsList := g.listAllProjects(client)

		for _, projectName := range projectsList {
			oneProjectData := g.resolveOneProject(client, projectName.Key)
			for _, repo := range oneProjectData {
				parsedUrl, err := url.Parse(repo.SshLink)
				if err != nil {
					panic(err)
				}
				path := strings.TrimSuffix(parsedUrl.Path, filepath.Ext(parsedUrl.Path))
				repositories = append(repositories, path)
			}
		}

	}

	return repositories
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

	gitCloneOptions.URL, _ = info.Remote(vcsurl.HTTPS)

	if args.AuthType == "ssh" {
		//what with 22 port using?
		gitCloneOptions.URL = fmt.Sprintf("git@%s:%s%s.git", info.Host, vcsPort, info.FullName)

		_, err := os.Stat(args.SSHKey)
		if err != nil {
			g.logger.Error("read file %s failed %s\n", args.SSHKey, err.Error())
			panic(err)
		}
		//todo add known hosts
		publicKeys, err := ssh.NewPublicKeysFromFile("git", args.SSHKey, "asdQWE123")
		if err != nil {
			g.logger.Error("generate publickeys failed: %s\n", err.Error())
			panic(err)
		}

		gitCloneOptions.Auth = publicKeys

		//todo add via agent
		// pkCallback, err := ssh.NewSSHAgentAuth("git")
		// if err != nil {
		// 	g.logger.Info("NewSSHAgentAuth error", "err", err)
		// 	return false
		// }

	} else {
		//format for BB https://bitbucket.com/scm/project/name.git
		gitCloneOptions.URL = fmt.Sprintf("https://%s/scm%s.git", info.Host, vcsPort, info.FullName)
		//g.logger.Debug(fmt.Sprintf("%#v", gitCloneOptions))
		gitCloneOptions.Auth = &http.BasicAuth{
			Username: username,
			Password: token,
		}

	}
	//TODO add logging from go-git
	_, err = git.PlainClone(args.TargetFolder, false, gitCloneOptions)

	if err != nil {
		g.logger.Info("Error on Clone occured", "err", err, "targetFolder", args.TargetFolder, "remote", gitCloneOptions.URL)
		return false
	}

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
	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		shared.PluginTypeVCS: &shared.VCSPlugin{Impl: VCS},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
