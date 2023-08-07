package main

import (
	"encoding/json"
	"fmt"
	"os"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	//bitbucketv2 "github.com/ktrysmt/go-bitbucket"

	"github.com/mitchellh/mapstructure"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/pkg/shared"
)

type VCSBitbucket struct {
	logger hclog.Logger
}

// Limit for Bitbucket v1 API page response
var opts = map[string]interface{}{
	"limit": maxLimitElements,
	"start": startElement,
}

func getProjectsResponse(r *bitbucketv1.APIResponse) ([]bitbucketv1.Project, error) {
	var m []bitbucketv1.Project
	err := mapstructure.Decode(r.Values["values"], &m)
	return m, err
}

// Listing all project in Bitbucket v1 API
func (g *VCSBitbucket) listAllProjects(client *bitbucketv1.APIClient) ([]shared.ProjectParams, error) {
	g.logger.Info("Starting to list all projects..")
	response, err := client.DefaultApi.GetProjects(opts)
	if err != nil {
		g.logger.Error("A listing projects function is failed", "error", err)
		return nil, err
	}

	g.logger.Info("Projects is listed")
	res, err := getProjectsResponse(response)
	if err != nil {
		g.logger.Error("A response parsing function is failed", "error", err)
		return nil, err
	}

	var projectsList []shared.ProjectParams
	for _, bitbucketRepo := range res {
		projectsList = append(projectsList, shared.ProjectParams{Key: bitbucketRepo.Key, Name: bitbucketRepo.Name, Link: bitbucketRepo.Links.Self[0].Href})
	}

	g.logger.Debug("The list of projects is ready")

	resultJson, _ := json.MarshalIndent(projectsList, "", "    ")
	g.logger.Debug(string(resultJson))

	return projectsList, nil
}

// Resolving information about all repositories in one project from Bitbucket v1 API
func (g *VCSBitbucket) resolveOneProject(client *bitbucketv1.APIClient, project string) ([]shared.RepositoryParams, error) {
	g.logger.Info("Resolving a particular project", "project", project)
	response, err := client.DefaultApi.GetRepositoriesWithOptions(project, opts)
	if err != nil {
		g.logger.Error("Resolving the project is failed", "project", project, "error", err)
		return nil, err
	}

	g.logger.Debug("Project is resolved", "project", project)
	result, err := bitbucketv1.GetRepositoriesResponse(response)
	if err != nil {
		g.logger.Error("Response parsing is failed", "project", project, "error", err)
		return nil, err
	}

	var resultList []shared.RepositoryParams
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

		resultList = append(resultList, shared.RepositoryParams{Namespace: project, RepoName: repo.Name, HttpLink: httpLink, SshLink: sshLink})
	}

	g.logger.Debug("The list of repositories is ready")

	resultJson, _ := json.MarshalIndent(resultList, "", "    ")
	g.logger.Debug(string(resultJson))

	return resultList, nil
}

func (g *VCSBitbucket) ListRepos(args shared.VCSListReposRequest) ([]shared.RepositoryParams, error) {
	g.logger.Debug("Starting execution of an all-repositories listing function", "args", args)
	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("An init stage of an all repositories listing function is failed", "error", err)
		return nil, err
	}

	client, cancel := BBClient(args.VCSURL, variables)
	defer cancel()

	var repositories []shared.RepositoryParams
	if len(args.Namespace) != 0 {
		oneProjectData, err := g.resolveOneProject(client, args.Namespace)
		if err != nil {
			g.logger.Error("The particular repository function is failed", "error", err)
			return nil, err
		}
		for _, repo := range oneProjectData {
			repositories = append(repositories, repo)
		}

	} else {
		projectsList, err := g.listAllProjects(client)
		if err != nil {
			g.logger.Error("The all-repositories listing function is failed", "error", err)
			return nil, err
		}

		for _, projectName := range projectsList {
			oneProjectData, err := g.resolveOneProject(client, projectName.Key)
			if err != nil {
				g.logger.Error("The listing of all repositories function is failed", "error", err)
				return nil, err
			}
			for _, repo := range oneProjectData {
				repositories = append(repositories, repo)
			}
		}

	}

	return repositories, nil
}

func (g *VCSBitbucket) RetrivePRInformation(args shared.VCSRetrivePRInformationRequest) (shared.PRParams, error) {
	g.logger.Debug("Starting retrive information about a PR", "args", args)

	var pr bitbucketv1.PullRequest
	var result shared.PRParams
	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("An init stage of retriving information about a PR is failed", "error", err)
		return result, err
	}

	client, cancel := BBClient(args.VCSURL, variables)
	defer cancel()

	g.logger.Info("Retriving information a particular PR", "PR", fmt.Sprintf("%v/%v/%v/%v", args.VCSURL, args.Namespace, args.Repository, args.PullRequestId))
	rawResponse, err := client.DefaultApi.GetPullRequest(args.Namespace, args.Repository, args.PullRequestId)
	if err != nil {
		g.logger.Error("Getting information about PR is failed", "error", err)
		return result, err
	}

	pr, err = bitbucketv1.GetPullRequestResponse(rawResponse)
	if err != nil {
		g.logger.Error("Parsing information about PR is failed", "error", err)
		return result, err
	}

	result = shared.PRParams{
		PullRequestId: pr.ID,
		Title:         pr.Title,
		Description:   pr.Description,
		State:         pr.State,
		AuthorEmail:   pr.Author.User.EmailAddress,
		AuthorName:    pr.Author.User.DisplayName,
		SelfLink:      pr.Links.Self[0].Href,
		CreatedDate:   pr.CreatedDate,
		UpdatedDate:   pr.UpdatedDate,
		FromRef: shared.RefPRInf{
			ID:           pr.FromRef.ID,
			LatestCommit: pr.FromRef.LatestCommit,
		},
		ToRef: shared.RefPRInf{
			ID:           pr.ToRef.ID,
			LatestCommit: pr.ToRef.LatestCommit,
		},
	}
	g.logger.Info("Information about particular PR is retrived", "PR", result.SelfLink)

	return result, nil
}

func (g *VCSBitbucket) AddReviewerToPR(args shared.VCSAddReviewerToPRRequest) (shared.PRParams, error) {
	g.logger.Debug("Starting retrive information about a PR", "args", args)

	var pr bitbucketv1.PullRequest
	var result shared.PRParams
	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("An init stage of retriving information about a PR is failed", "error", err)
		return result, err
	}

	client, cancel := BBClient(args.VCSURL, variables)
	defer cancel()

	g.logger.Info("Retriving information a particular PR", "PR", fmt.Sprintf("%v/%v/%v/%v", args.VCSURL, args.Namespace, args.Repository, args.PullRequestId))
	rawResponse, err := client.DefaultApi.GetPullRequest(args.Namespace, args.Repository, args.PullRequestId)
	if err != nil {
		g.logger.Error("Getting information about PR is failed", "error", err)
		return result, err
	}

	pr, err = bitbucketv1.GetPullRequestResponse(rawResponse)
	if err != nil {
		g.logger.Error("Parsing information about PR is failed", "error", err)
		return result, err
	}

	result = shared.PRParams{
		PullRequestId: pr.ID,
		Title:         pr.Title,
		Description:   pr.Description,
		State:         pr.State,
		AuthorEmail:   pr.Author.User.EmailAddress,
		AuthorName:    pr.Author.User.DisplayName,
		SelfLink:      pr.Links.Self[0].Href,
		CreatedDate:   pr.CreatedDate,
		UpdatedDate:   pr.UpdatedDate,
		FromRef: shared.RefPRInf{
			ID:           pr.FromRef.ID,
			LatestCommit: pr.FromRef.LatestCommit,
		},
		ToRef: shared.RefPRInf{
			ID:           pr.ToRef.ID,
			LatestCommit: pr.ToRef.LatestCommit,
		},
	}
	g.logger.Info("Information about particular PR is retrived", "PR", result.SelfLink)

	return result, nil
}

func (g *VCSBitbucket) Fetch(args shared.VCSFetchRequest) error {
	variables, err := g.init("fetch", args.AuthType)
	if err != nil {
		g.logger.Error("An init function for a fetching function is failed", "error", err)
		return err
	}

	err = shared.GitClone(args, variables, g.logger)
	if err != nil {
		g.logger.Error("The fetching function is failed", "error", err)
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
		shared.PluginTypeVCS: &shared.VCSPlugin{Impl: VCS},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
