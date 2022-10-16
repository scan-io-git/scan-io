package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	"github.com/google/go-github/v47/github"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/libs/vcs"
	"github.com/scan-io-git/scan-io/shared"
)

// Here is a real implementation of VCS
type VCSGithub struct {
	logger hclog.Logger
}

func (g *VCSGithub) ListRepos(args vcs.VCSListReposRequest) ([]vcs.RepositoryParams, error) {
	g.logger.Debug("Entering ListRepos", "organization", args.Namespace)
	client := github.NewClient(nil)
	opt := &github.RepositoryListByOrgOptions{Type: "public"}
	repos, _, err := client.Repositories.ListByOrg(context.Background(), args.Namespace, opt)
	if err != nil {
		g.logger.Error("Error listing projects", "err", err)
		return nil, err
	}

	reposParams := make([]vcs.RepositoryParams, len(repos))
	for i, repo := range repos {
		parts := strings.Split(*repo.FullName, "/")
		reposParams[i] = vcs.RepositoryParams{
			Namespace: strings.Join(parts[:len(parts)-1], "/"),
			RepoName:  *repo.Name,
			SshLink:   *repo.SSHURL,
			HttpLink:  *repo.CloneURL,
		}
	}

	return reposParams, nil
}

func (g *VCSGithub) Fetch(args vcs.VCSFetchRequest) error {

	g.logger.Debug("Fetch called", "args", args)

	info, err := vcsurl.Parse(fmt.Sprintf("https://%s/%s", args.VCSURL, args.Repository))
	if err != nil {
		g.logger.Error("unable to parse project '%s'", args.Repository)
		return err
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
			return err
		}
		gitCloneOptions.Auth = pkCallback
	}

	_, err = git.PlainClone(args.TargetFolder, false, gitCloneOptions)
	if err != nil {
		g.logger.Info("Error on Clone occured", "err", err, "targetFolder", args.TargetFolder, "remote", gitCloneOptions.URL)
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

	VCS := &VCSGithub{
		logger: logger,
	}
	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		shared.PluginTypeVCS: &vcs.VCSPlugin{Impl: VCS},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
