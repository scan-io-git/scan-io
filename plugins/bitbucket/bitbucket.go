package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	//"strings"
	"time"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/shared"
	"github.com/scan-io-git/scan-io/utils"
)

const ()

var opts = map[string]interface{}{
	"limit": 2000,
	"start": 0,
}

// Here is a real implementation of VCS
type VCSBitbucket struct {
	logger hclog.Logger
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

func (g *VCSBitbucket) ListAllRepos(project string) []string {
	g.logger.Debug("Entering ListAllRepos", "project", project)

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
	g.listOneProject(client, project)

	var projects []string
	return projects
}

func (g *VCSBitbucket) Fetch(project string) bool {

	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
		// return false
	}
	projectsFolder := filepath.Join(home, "/.scanio/projects")
	if _, err := os.Stat(projectsFolder); os.IsNotExist(err) {
		g.logger.Info("projectsFolder '%s' does not exists. Creating...", projectsFolder)
		if err := os.MkdirAll(projectsFolder, os.ModePerm); err != nil {
			panic(err)
			// return false
		}
	}

	info, err := vcsurl.Parse(project)
	if err != nil {
		g.logger.Error("unable to parse project '%s'", project)
		panic(err)
		// return false
	}

	targetFolder := filepath.Join(projectsFolder, info.ID)
	remote, _ := info.Remote(vcsurl.HTTPS)

	_, err = git.PlainClone(targetFolder, false, &git.CloneOptions{
		URL:      remote,
		Progress: os.Stdout,
		Depth:    1,
	})
	if err != nil {
		g.logger.Info("Error on Clone occured", "err", err, "targetFolder", targetFolder, "remote", remote)
		// panic(err)
		return false
	}

	g.logger.Info("finished", "remote", remote)

	return true
}

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
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
		"vcs": &shared.VCSPlugin{Impl: VCS},
	}

	// logger.Debug("message from plugin", "foo", "bar")

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
