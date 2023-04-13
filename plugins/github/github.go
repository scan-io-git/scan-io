package main

import (
	"context"
	"os"
	"strings"

	"github.com/google/go-github/v47/github"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

type VCSGithub struct {
	logger hclog.Logger
}

func (g *VCSGithub) ListRepos(args shared.VCSListReposRequest) ([]shared.RepositoryParams, error) {
	g.logger.Debug("Starting an all-repositories listing function", "args", args)
	client := github.NewClient(nil)
	opt := &github.RepositoryListByOrgOptions{Type: "public"}
	repos, _, err := client.Repositories.ListByOrg(context.Background(), args.Namespace, opt)
	if err != nil {
		g.logger.Error("A particular organisation function is failed", "error", err)
		return nil, err
	}

	reposParams := make([]shared.RepositoryParams, len(repos))
	for i, repo := range repos {
		parts := strings.Split(*repo.FullName, "/")
		reposParams[i] = shared.RepositoryParams{
			Namespace: strings.Join(parts[:len(parts)-1], "/"),
			RepoName:  *repo.Name,
			SshLink:   *repo.SSHURL,
			HttpLink:  *repo.CloneURL,
		}
	}

	return reposParams, nil
}

func (g *VCSGithub) Fetch(args shared.VCSFetchRequest) error {
	//variables, err := g.init("fetch")
	variables := shared.EvnVariables{}
	// if err != nil {
	// 	g.logger.Error("Fetching is failed", "error", err)
	// 	return err
	// }

	err := shared.GitClone(args, variables, g.logger)
	if err != nil {
		g.logger.Error("A fetching function is failed", "error", err)
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

	var pluginMap = map[string]plugin.Plugin{
		shared.PluginTypeVCS: &shared.VCSPlugin{Impl: VCS},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
