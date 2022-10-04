package main

import (
	"context"
	"encoding/json"
	"os"
	//"path/filepath"
	"fmt"
	//"strings"
	"time"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/shared"
	"github.com/scan-io-git/scan-io/utils"
)

// Here is a real implementation of VCS
type VCSBitbucket struct {
	logger hclog.Logger
}

var opts = map[string]interface{}{
	"limit": 2000,
	"start": 0,
}

func (g *VCSBitbucket) listAllProjects(client *bitbucketv1.APIClient) []utils.BBProject {
	response, err := client.DefaultApi.GetProjects(opts)
	g.logger.Debug("ListAllRepos was read")
	if err != nil {
		panic(err.Error())
	}

	res, err := utils.GetProjectsResponse(response)
	if err != nil {
		panic(err.Error())
	}

	var projectsList []utils.BBProject
	for _, bitbucketRepo := range res {
		projectsList = append(projectsList, utils.BBProject{Key: bitbucketRepo.Key, Link: bitbucketRepo.Links.Self[0].Href})

	}
	resultJson, _ := json.MarshalIndent(projectsList, "", "    ")
	g.logger.Debug(string(resultJson))

	return projectsList
}

func (g *VCSBitbucket) listOneProject(client *bitbucketv1.APIClient, project string) []byte {

	response, err := client.DefaultApi.GetRepositoriesWithOptions(project, opts)
	g.logger.Debug("Project loaded", "project", project)
	if err != nil {
		panic(err.Error())
	}

	result, err := bitbucketv1.GetRepositoriesResponse(response)
	if err != nil {
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
	resultJson, _ := json.MarshalIndent(resultList, "", "    ")
	g.logger.Debug(string(resultJson))

	return resultJson
}

func (g *VCSBitbucket) ListRepos(args shared.VCSListReposRequest) []string {
	g.logger.Debug("Entering ListAllRepos", "project", args.Organization)

	basicAuth := bitbucketv1.BasicAuth{UserName: "", Password: ""}

	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Millisecond)
	ctx = context.WithValue(ctx, bitbucketv1.ContextBasicAuth, basicAuth)
	defer cancel()

	client := bitbucketv1.NewAPIClient(
		ctx,
		bitbucketv1.NewConfiguration(""),
	)

	//projectsList := g.listAllProjects(client)
	g.listAllProjects(client)

	//oneProject := g.listOneProject(client, project)
	g.listOneProject(client, args.Organization)

	var projects []string
	return projects
}

func (g *VCSBitbucket) Fetch(args shared.VCSFetchRequest) bool {

	g.logger.Debug("Fetch called", "args", args)

	info, err := vcsurl.Parse(fmt.Sprintf("https://%s/%s", args.VCSURL, args.Project))
	if err != nil {
		g.logger.Error("unable to parse project '%s'", args.Project)
		panic(err)
	}

	gitCloneOptions := &git.CloneOptions{
		// Auth:     pkCallback,
		// URL:      remote,
		Progress: os.Stdout,
		Depth:    1,
	}
	gitCloneOptions.URL, _ = info.Remote(vcsurl.HTTPS)
	if args.AuthType == "ssh" {
		gitCloneOptions.URL = fmt.Sprintf("git@%s:%s.git", info.Host, info.FullName)

		pkCallback, err := ssh.NewSSHAgentAuth("git")
		if err != nil {
			g.logger.Info("NewSSHAgentAuth error", "err", err)
			return false
		}
		gitCloneOptions.Auth = pkCallback
	}

	_, err = git.PlainClone(args.TargetFolder, false, gitCloneOptions)
	if err != nil {
		g.logger.Info("Error on Clone occured", "err", err, "targetFolder", args.TargetFolder, "remote", gitCloneOptions.URL)
		// panic(err)
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
