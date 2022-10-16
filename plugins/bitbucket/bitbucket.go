package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	//bitbucketv2 "github.com/ktrysmt/go-bitbucket"

	"github.com/mitchellh/mapstructure"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/libs/vcs"
	"github.com/scan-io-git/scan-io/shared"
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
func (g *VCSBitbucket) init(command string) (vcs.EvnVariables, error) {
	var variables vcs.EvnVariables
	variables.Username = os.Getenv("BITBUCKET_USERNAME")
	variables.Token = os.Getenv("BITBUCKET_TOKEN")

	if (len(variables.Username) == 0) || (len(variables.Token) == 0) {
		err := fmt.Errorf("BITBUCKET_USERNAME or BITBUCKET_TOKEN is not provided in an environment.")
		g.logger.Error("Env problems", "error", err.Error())
		return variables, err
	}

	if command == "fetch" {
		variables.VcsPort = os.Getenv("BITBUCKET_SSH_PORT")
		variables.SshKeyPassword = os.Getenv("BITBUCKET_SSH_KEY_PASSWORD")

		if len(variables.VcsPort) == 0 {
			g.logger.Warn("BITBUCKET_SSH_PORT is not provided in an environment. Using default 22 ssh port")
			variables.VcsPort = "22"
		}
		if len(variables.SshKeyPassword) == 0 {
			g.logger.Warn("BITBUCKET_SSH_KEY_PASSOWRD is empty or not provided.")
		}
	}
	return variables, nil
}

// Listing all project in Bitbucket v1 API
func (g *VCSBitbucket) listAllProjects(client *bitbucketv1.APIClient) ([]vcs.ProjectParams, error) {
	g.logger.Info("Trying to list all projects..")
	response, err := client.DefaultApi.GetProjects(opts)
	if err != nil {
		g.logger.Error("Listing projects is failed", "error", err.Error())
		return nil, err
	}

	g.logger.Info("Projects is listed")
	res, err := getProjectsResponse(response)
	if err != nil {
		g.logger.Error("Response parsing is failed", "error", err.Error())
		return nil, err
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
	g.logger.Info("Resolving a particular project", "project", project)
	response, err := client.DefaultApi.GetRepositoriesWithOptions(project, opts)
	if err != nil {
		g.logger.Error("Resolving the project is failed", "project", project, "error", err.Error())
		return nil, err
	}

	g.logger.Info("Project is resolved", "project", project)
	result, err := bitbucketv1.GetRepositoriesResponse(response)
	if err != nil {
		g.logger.Error("Response parsing is failed", "project", project, "error", err.Error())
		return nil, err
	}

	var resultList []vcs.RepositoryParams
	for _, repo := range result {
		var httpLink string
		var sshLink string

		for _, clone_links := range repo.Links.Clone {

			if clone_links.Name == "http" {
				httpLink = clone_links.Href
			} else if clone_links.Name == "ssh" {
				sshLink = clone_links.Href
			} else {
				continue
			}
		}

		resultList = append(resultList, vcs.RepositoryParams{Namespace: project, RepoName: repo.Name, HttpLink: httpLink, SshLink: sshLink})
	}

	g.logger.Info("List of repositories is ready.")
	resultJson, _ := json.MarshalIndent(resultList, "", "    ")
	g.logger.Debug(string(resultJson))

	return resultList, nil
}

func (g *VCSBitbucket) ListRepos(args vcs.VCSListReposRequest) ([]vcs.RepositoryParams, error) {
	g.logger.Debug("Entering ListRepos", "args", args)
	variables, err := g.init("list")
	if err != nil {
		g.logger.Error("Init Listing all repos is failed", "error", err.Error())
		return nil, err
	}

	baseURL := fmt.Sprintf("https://%s/rest", args.VCSURL)
	basicAuth := bitbucketv1.BasicAuth{UserName: variables.Username, Password: variables.Token}

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
			g.logger.Error("Listing all repos is failed", "error", err.Error())
			return nil, err
		}
		for _, repo := range oneProjectData {
			repositories = append(repositories, repo)
		}

	} else {
		g.logger.Info("Listing all repos for all projects")
		projectsList, err := g.listAllProjects(client)
		if err != nil {
			g.logger.Error("Listing all repos is failed", "error", err.Error())
			return nil, err
		}

		for _, projectName := range projectsList {
			oneProjectData, err := g.resolveOneProject(client, projectName.Key)
			if err != nil {
				g.logger.Error("Listing all repos is failed", "error", err.Error())
				return nil, err
			}
			for _, repo := range oneProjectData {
				repositories = append(repositories, repo)
			}
		}

	}

	return repositories, nil
}

func (g *VCSBitbucket) Fetch(args vcs.VCSFetchRequest) error {
	variables, err := g.init("fetch")
	if err != nil {
		g.logger.Error("Fetching is failed", "error", err.Error())
		return err
	}

	_, err = vcs.GitClone(args, variables, g.logger)
	if err != nil {
		return err
	}
	return nil
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
		shared.PluginTypeVCS: &vcs.VCSPlugin{Impl: VCS},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
