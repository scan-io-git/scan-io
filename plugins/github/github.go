package main

import (
	"context"
	"os"
	"strings"

	"github.com/google/go-github/v47/github"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
)

// TODO: Wrap it in a custom error handler to add to the stack trace.
// Metadata of the plugin
var (
	Version       = "unknown"
	GolangVersion = "unknown"
	BuildTime     = "unknown"
)

// VCSGitlab implements VCS operations for Gitlab.
type VCSGithub struct {
	logger       hclog.Logger
	globalConfig *config.Config
}

// newVCSGithub creates a new instance of VCSGithub.
func newVCSGithub(logger hclog.Logger) *VCSGithub {
	return &VCSGithub{
		logger: logger,
	}
}

// setGlobalConfig sets the global configuration for the VCSGithub instance.
func (g *VCSGithub) setGlobalConfig(globalConfig *config.Config) {
	g.globalConfig = globalConfig
}

func (g *VCSGithub) ListRepositories(args shared.VCSListRepositoriesRequest) ([]shared.RepositoryParams, error) {
	g.logger.Debug("Starting an all-repositories listing function", "args", args)

	if err := g.validateList(&args); err != nil {
		g.logger.Error("validation failed for listing repositories operation", "error", err)
		return nil, err
	}

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
	// if err := g.validateRetrievePRInformation(&args); err != nil {
	// 	g.logger.Error("validation failed for retrieving pull request information operation", "error", err)
	// 	return nil, err
	// }
	return shared.PRParams{}, errors.NewNotImplementedError("RetrievePRInformation", "GitHub plugin")
}

func (g *VCSGithub) AddRoleToPR(args shared.VCSAddRoleToPRRequest) (bool, error) {
	// if err := g.validateAddRoleToPR(&args); err != nil {
	// 	g.logger.Error("validation failed for adding a user to PR operation", "error", err)
	// 	return nil, err
	// }
	return false, errors.NewNotImplementedError("AddRoleToPR", "GitHub plugin")
}

func (g *VCSGithub) SetStatusOfPR(args shared.VCSSetStatusOfPRRequest) (bool, error) {
	// if err := g.validateSetStatusOfPR(&args); err != nil {
	// 	g.logger.Error("validation failed for setting a status to PR operation", "error", err)
	// 	return nil, err
	// }
	return false, errors.NewNotImplementedError("SetStatusOfPR", "GitHub plugin")
}

func (g *VCSGithub) AddCommentToPR(args shared.VCSAddCommentToPRRequest) (bool, error) {
	// if err := g.validateAddCommentToPR(&args); err != nil {
	// 	g.logger.Error("validation failed for adding a comment to PR operation", "error", err)
	// 	return nil, err
	// }
	return false, errors.NewNotImplementedError("AddCommentToPR", "GitHub plugin")
}

// Fetch retrieves code based on the provided VCSFetchRequest.
func (g *VCSGithub) Fetch(args shared.VCSFetchRequest) (shared.VCSFetchResponse, error) {
	var result shared.VCSFetchResponse

	if err := g.validateFetch(&args); err != nil {
		g.logger.Error("validation failed for fetch operation", "error", err)
		return shared.VCSFetchResponse{}, err
	}

	switch args.Mode {
	case "PRscan":
		return shared.VCSFetchResponse{}, errors.NewNotImplementedError("PRscan", "GitHub plugin")

	default:
		pluginConfigMap, err := shared.StructToMap(g.globalConfig.BitbucketPlugin)
		if err != nil {
			g.logger.Error("error converting struct to map", "error", err)
			return result, err
		}

		clientGit, err := git.New(g.logger, g.globalConfig, pluginConfigMap, &args)
		if err != nil {
			g.logger.Error("failed to initialize Git client", "error", err)
			return result, err
		}

		path, err := clientGit.CloneRepository(&args, "main")
		if err != nil {
			g.logger.Error("failed to clone repository", "error", err)
			return result, err
		}

		result.Path = path
	}

	return result, nil
}

// Setup initializes the global configuration for the VCSGithub instance.
func (g *VCSGithub) Setup(configData config.Config) (bool, error) {
	g.setGlobalConfig(&configData)
	if err := UpdateConfigFromEnv(g.globalConfig); err != nil {
		g.logger.Error("failed to update the global config from env variables", "error", err)
		return false, err
	}
	return true, nil
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	githubInstance := newVCSGithub(logger)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			shared.PluginTypeVCS: &shared.VCSPlugin{Impl: githubInstance},
		},
		Logger: logger,
	})
}
