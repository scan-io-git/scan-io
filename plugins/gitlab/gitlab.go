package main

import (
	"fmt"
	"os"
	"strings"

	// "github.com/google/go-github/v47/github"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/xanzy/go-gitlab"
)

// Here is a real implementation of VCS
type VCSGitlab struct {
	logger hclog.Logger
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

func (g *VCSGitlab) ListRepos(args shared.VCSListReposRequest) ([]shared.RepositoryParams, error) {
	g.logger.Debug("Entering ListRepos", "args", args)

	gitlabClient, err := getGitlabClient(args.VCSURL)
	if err != nil {
		g.logger.Warn("Failed to create gitlab Client", "err", err)
		return nil, err
	}

	allGroups, err := g.getGitlabGroups(gitlabClient, args.Namespace)
	if err != nil {
		g.logger.Warn("Failed to get list of Gitlab groups", "err", err)
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
				g.logger.Warn("gitlab ListGroups error", "err", err, "page", page)
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
					Namespace: project.Namespace.FullPath,
					RepoName:  project.Path,
					HttpLink:  project.HTTPURLToRepo,
					SshLink:   project.SSHURLToRepo,
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

func (g *VCSGitlab) Fetch(args shared.VCSFetchRequest) error {
	//variables, err := g.init("fetch")
	variables := shared.EvnVariables{}
	// if err != nil {
	// 	g.logger.Error("Fetching is failed", "error", err)
	// 	return err
	// }

	err := shared.GitClone(args, variables, g.logger)
	if err != nil {
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
