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

	// g.logger.Debug("repos details", "projects", projects)

	return projects
	// return strings.Join(projects, ",")
}

func (g *VCSGithub) Fetch(args shared.VCSFetchRequest) bool {

	g.logger.Debug("Fetch called", "args", args)

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

	info, err := vcsurl.Parse(args.Project)
	if err != nil {
		g.logger.Error("unable to parse project '%s'", args.Project)
		panic(err)
		// return false
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

	// ref, err := r.Head()
	// if err != nil {
	// 	g.logger.Info("Error retrieving Head", "err", err)
	// 	return false
	// }

	// commit, err := r.CommitObject(ref.Hash())
	// if err != nil {
	// 	g.logger.Info("Error getting Commit", "err", err)
	// 	return false
	// }

	// g.logger.Info("finished", "remote", gitCloneOptions.URL, "ref", ref, "hash", ref.Hash().String())

	return true
	// g.logger.Debug("message from VCSHello.Fetch")
	// return strings.Join(projects, ",")
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
// var handshakeConfig = plugin.HandshakeConfig{
// 	ProtocolVersion:  1,
// 	MagicCookieKey:   "BASIC_PLUGIN",
// 	MagicCookieValue: "hello",
// }

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

	// logger.Debug("message from plugin", "foo", "bar")

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
