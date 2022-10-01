package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	"github.com/google/go-github/v47/github"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/shared"
)

// Here is a real implementation of VCS
type VCSGithub struct {
	logger hclog.Logger
}

func (g *VCSGithub) ListRepos(args shared.VCSListReposRequest) []string {
	g.logger.Debug("Entering ListRepos", "organization", args.Organization)
	client := github.NewClient(nil)
	opt := &github.RepositoryListByOrgOptions{Type: "public"}
	repos, _, err := client.Repositories.ListByOrg(context.Background(), args.Organization, opt)
	if err != nil {
		g.logger.Error("Error listing projects", "err", err)
		panic(err)
	}

	projects := []string{}
	for _, repo := range repos {
		projects = append(projects, *repo.HTMLURL)
	}

	return projects
}

func (g *VCSGithub) Fetch(args shared.VCSFetchRequest) bool {

	g.logger.Debug("Fetch called", "args", args)

	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
	}
	projectsFolder := filepath.Join(home, "/.scanio/projects")
	if _, err := os.Stat(projectsFolder); os.IsNotExist(err) {
		g.logger.Info("projectsFolder '%s' does not exists. Creating...", projectsFolder)
		if err := os.MkdirAll(projectsFolder, os.ModePerm); err != nil {
			panic(err)
		}
	}

	info, err := vcsurl.Parse(args.Project)
	if err != nil {
		g.logger.Error("unable to parse project '%s'", args.Project)
		panic(err)
	}

	targetFolder := filepath.Join(projectsFolder, info.ID)

	gitCloneOptions := &git.CloneOptions{
		// Auth:     pkCallback,
		// URL:      remote,
		Progress: os.Stdout,
		Depth:    1,
	}
	gitCloneOptions.URL, _ = info.Remote(vcsurl.HTTPS)
	if args.AuthType == "ssh" {
		gitCloneOptions.URL = fmt.Sprintf("git@%s:%s/%s.git", info.Host, info.Username, info.Name)

		pkCallback, err := ssh.NewSSHAgentAuth("git")
		if err != nil {
			g.logger.Info("NewSSHAgentAuth error", "err", err)
			return false
		}
		gitCloneOptions.Auth = pkCallback
	}

	_, err = git.PlainClone(targetFolder, false, gitCloneOptions)
	if err != nil {
		g.logger.Info("Error on Clone occured", "err", err, "targetFolder", targetFolder, "remote", gitCloneOptions.URL)
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

	VCS := &VCSGithub{
		logger: logger,
	}
	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		"vcs": &shared.VCSPlugin{Impl: VCS},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
