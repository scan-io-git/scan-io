package main

import (
	"fmt"
	"os"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	// "github.com/google/go-github/v47/github"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/shared"
	"github.com/xanzy/go-gitlab"
)

// Here is a real implementation of VCS
type VCSGitlab struct {
	logger hclog.Logger
}

func (g *VCSGitlab) ListRepos(args shared.VCSListReposRequest) []string {
	g.logger.Debug("Entering ListRepos", "args", args)

	baseURL := fmt.Sprintf("https://%s/api/v4", args.VCSURL)
	git, err := gitlab.NewClient(os.Getenv("GITLAB_TOKEN"), gitlab.WithBaseURL(baseURL))
	if err != nil {
		g.logger.Warn("Failed to create gitlab Client", "err", err)
		return []string{}
	}

	projectsList := []string{}
	page := 1
	perPage := 100
	if args.Limit != 0 && args.Limit < perPage {
		perPage = args.Limit
	}
	for {

		listOptions := gitlab.ListOptions{
			Page:    page,
			PerPage: perPage,
		}
		idString := "id"
		projects, _, err := git.Projects.ListProjects(&gitlab.ListProjectsOptions{ListOptions: listOptions, OrderBy: &idString})
		if err != nil {
			g.logger.Warn("gitlab ListProject error", "err", err, "page", page)
			return []string{}
		}

		for _, project := range projects {
			projectsList = append(projectsList, project.PathWithNamespace)
		}

		if len(projects) < perPage {
			break
		}

		if args.Limit != 0 && len(projectsList) >= args.Limit {
			g.logger.Debug("gitlab ListProject call", "len(projectsList)", len(projectsList))
			return projectsList[0:args.Limit]
		}

		g.logger.Debug("gitlab ListProject call", "page", page, "projects[0].ID", projects[0].ID)

		page += 1
	}

	return projectsList
}

func (g *VCSGitlab) Fetch(args shared.VCSFetchRequest) bool {

	info, err := vcsurl.Parse(fmt.Sprintf("https://%s%s", args.VCSURL, args.Project))
	if err != nil {
		g.logger.Error("Unable to parse VCS url info", "VCSURL", args.VCSURL, "project", args.Project)
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

	VCS := &VCSGitlab{
		logger: logger,
	}
	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		shared.PluginTypeVCS: &shared.VCSPlugin{Impl: VCS},
	}

	// logger.Debug("message from plugin", "foo", "bar")

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
