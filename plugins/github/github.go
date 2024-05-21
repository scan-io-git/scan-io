package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v47/github"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// VCSGitlab implements VCS operations for Gitlab.
type VCSGithub struct {
	logger       hclog.Logger
	globalConfig *config.Config
}

func (g *VCSGithub) ListRepositories(args shared.VCSListRepositoriesRequest) ([]shared.RepositoryParams, error) {
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

func (g *VCSGithub) RetrievePRInformation(args shared.VCSRetrievePRInformationRequest) (shared.PRParams, error) {
	var result shared.PRParams
	err := fmt.Errorf("The function is not implemented got Github.")

	return result, err
}

func (g *VCSGithub) AddRoleToPR(args shared.VCSAddRoleToPRRequest) (bool, error) {
	err := fmt.Errorf("The function is not implemented got Github.")

	return false, err
}

func (g *VCSGithub) SetStatusOfPR(args shared.VCSSetStatusOfPRRequest) (bool, error) {
	err := fmt.Errorf("The function is not implemented got Github.")

	return false, err
}

func (g *VCSGithub) AddCommentToPR(args shared.VCSAddCommentToPRRequest) (bool, error) {
	err := fmt.Errorf("The function is not implemented got Github.")

	return false, err
}

func (g *VCSGithub) Fetch(args shared.VCSFetchRequest) (shared.VCSFetchResponse, error) {
	var result shared.VCSFetchResponse

	path, err := git.CloneRepository(g.logger, g.globalConfig, &args, "main")
	if err != nil {
		g.logger.Error("failed to clone repository", "error", err)
		return result, err
	}
	result.Path = path

	return result, nil
}

func (g *VCSGithub) Setup(configData config.Config) (bool, error) {
	g.globalConfig = &configData
	return true, nil
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
