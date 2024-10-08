package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xanzy/go-gitlab"
	// "github.com/google/go-github/v47/github"
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
type VCSGitlab struct {
	logger       hclog.Logger
	globalConfig *config.Config
}

// newVCSGitlab creates a new instance of VCSGitlab.
func newVCSGitlab(logger hclog.Logger) *VCSGitlab {
	return &VCSGitlab{
		logger: logger,
	}
}

// setGlobalConfig sets the global configuration for the VCSGitlab instance.
func (g *VCSGitlab) setGlobalConfig(globalConfig *config.Config) {
	g.globalConfig = globalConfig
}

func getGitlabClient(vcsBaseURL string) (*gitlab.Client, error) {
	baseURL := fmt.Sprintf("https://%s/api/v4", vcsBaseURL)
	return gitlab.NewClient(os.Getenv("GITLAB_TOKEN"), gitlab.WithBaseURL(baseURL))
}

func (g VCSGitlab) getGitlabGroups(gitlabClient *gitlab.Client, searchNamespace string) ([]int, error) {

	allGroups := []int{}
	page := 1
	perPage := 100
	for {
		groups, _, err := gitlabClient.Groups.ListGroups(&gitlab.ListGroupsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
			OrderBy:      gitlab.String("id"),
			Sort:         gitlab.String("asc"),
			AllAvailable: gitlab.Bool(true),
			Search:       &searchNamespace,
		})
		if err != nil {
			g.logger.Warn("gitlab ListGroups error", "err", err, "page", page)
			return nil, err
		}

		for _, group := range groups {
			if len(searchNamespace) == 0 || group.FullPath == searchNamespace {
				allGroups = append(allGroups, group.ID)
			}
		}

		if len(groups) < perPage {
			break
		}

		page += 1
	}

	return allGroups, nil

}

func (g *VCSGitlab) ListRepositories(args shared.VCSListRepositoriesRequest) ([]shared.RepositoryParams, error) {
	g.logger.Debug("Starting an all-repositories listing function", "args", args)
	if err := g.validateList(&args); err != nil {
		g.logger.Error("validation failed for listing repositories operation", "error", err)
		return nil, err
	}

	gitlabClient, err := getGitlabClient(args.RepoParam.Domain)
	if err != nil {
		g.logger.Error("Failed to create gitlab Client", "error", err)
		return nil, err
	}

	allGroups, err := g.getGitlabGroups(gitlabClient, args.RepoParam.Namespace)
	if err != nil {
		g.logger.Error("Failed to get list of Gitlab groups", "error", err)
		return nil, err
	}
	g.logger.Debug("Collected groups", "total", len(allGroups))

	reposParams := []shared.RepositoryParams{}

	for i, groupID := range allGroups {
		g.logger.Info("Getting list of projects for a group", "page", i+1, "totalPages", len(allGroups), "groupID", groupID)

		page := 1
		perPage := 100
		for {
			projects, _, err := gitlabClient.Groups.ListGroupProjects(groupID, &gitlab.ListGroupProjectsOptions{
				ListOptions: gitlab.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
				OrderBy: gitlab.String("id"),
				Sort:    gitlab.String("asc"),
			})
			if err != nil {
				g.logger.Error("Gitlab ListGroups error", "page", page, "error", err)
				return nil, err
			}

			for _, project := range projects {
				if len(args.Language) != 0 {
					langauges, _, err := gitlabClient.Projects.GetProjectLanguages(project.ID)
					if err != nil {
						g.logger.Error("gitlab GetProjectLanguages error", "err", err, "project.ID", project.ID, "project.PathWithNamespace", project.PathWithNamespace)
						return nil, err
					}

					match := false
					for lang := range *langauges {
						if strings.ToLower(lang) == args.Language {
							match = true
							break
						}
					}

					if !match {
						continue
					}
				}
				reposParams = append(reposParams, shared.RepositoryParams{
					Namespace:  project.Namespace.FullPath,
					Repository: project.Path,
					HTTPLink:   project.HTTPURLToRepo,
					SSHLink:    project.SSHURLToRepo,
				})
			}

			if len(projects) < perPage {
				break
			}

			page += 1
		}
	}

	return reposParams, nil
}

func (g *VCSGitlab) RetrievePRInformation(args shared.VCSRetrievePRInformationRequest) (shared.PRParams, error) {
	// if err := g.validateRetrievePRInformation(&args); err != nil {
	// 	g.logger.Error("validation failed for retrieving pull request information operation", "error", err)
	// 	return nil, err
	// }
	return shared.PRParams{}, errors.NewNotImplementedError("RetrievePRInformation", "Gitlab plugin")
}

func (g *VCSGitlab) AddRoleToPR(args shared.VCSAddRoleToPRRequest) (bool, error) {
	// if err := g.validateAddRoleToPR(&args); err != nil {
	// 	g.logger.Error("validation failed for adding a user to PR operation", "error", err)
	// 	return nil, err
	// }
	return false, errors.NewNotImplementedError("AddRoleToPR", "Gitlab plugin")
}

func (g *VCSGitlab) SetStatusOfPR(args shared.VCSSetStatusOfPRRequest) (bool, error) {
	// if err := g.validateSetStatusOfPR(&args); err != nil {
	// 	g.logger.Error("validation failed for setting a status to PR operation", "error", err)
	// 	return nil, err
	// }
	return false, errors.NewNotImplementedError("SetStatusOfPR", "Gitlab plugin")
}

func (g *VCSGitlab) AddCommentToPR(args shared.VCSAddCommentToPRRequest) (bool, error) {
	// if err := g.validateAddCommentToPR(&args); err != nil {
	// 	g.logger.Error("validation failed for adding a comment to PR operation", "error", err)
	// 	return nil, err
	// }
	return false, errors.NewNotImplementedError("AddCommentToPR", "Gitlab plugin")
}

// Fetch retrieves code based on the provided VCSFetchRequest.
func (g *VCSGitlab) Fetch(args shared.VCSFetchRequest) (shared.VCSFetchResponse, error) {
	var result shared.VCSFetchResponse

	if err := g.validateFetch(&args); err != nil {
		g.logger.Error("validation failed for fetch operation", "error", err)
		return shared.VCSFetchResponse{}, err
	}

	switch args.Mode {
	case "fetchPR":
		return shared.VCSFetchResponse{}, errors.NewNotImplementedError("fetchPR", "Gitlab plugin")

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

		path, err := clientGit.CloneRepository(&args)
		if err != nil {
			g.logger.Error("failed to clone repository", "error", err)
			return result, err
		}

		result.Path = path
	}

	return result, nil
}

// Setup initializes the global configuration for the VCSGitlab instance.
func (g *VCSGitlab) Setup(configData config.Config) (bool, error) {
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

	gitlabInstance := newVCSGitlab(logger)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			shared.PluginTypeVCS: &shared.VCSPlugin{Impl: gitlabInstance},
		},
		Logger: logger,
	})
}
