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
	"github.com/scan-io-git/scan-io/libs/vcs"
	"github.com/scan-io-git/scan-io/shared"
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

func (g *VCSGitlab) ListRepos(args vcs.VCSListReposRequest) ([]vcs.RepositoryParams, error) {
	g.logger.Debug("Entering ListRepos 2", "args", args)

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

	reposParams := []vcs.RepositoryParams{}

	page := 1
	perPage := 100
	for i, groupID := range allGroups {
		g.logger.Debug("Getting list of projects for a group", "#", i+1, "groupID", groupID)

		page = 1
		perPage = 100
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
				reposParams = append(reposParams, vcs.RepositoryParams{
					Namespace: project.Namespace.FullPath,
					RepoName:  project.Name,
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

func (g *VCSGitlab) Fetch(args vcs.VCSFetchRequest) error {

	url := fmt.Sprintf("https://%s/%s", args.VCSURL, args.Repository)
	g.logger.Debug("Fetch", "url", url)
	info, err := vcsurl.Parse(url)
	if err != nil {
		g.logger.Error("Unable to parse VCS url info", "VCSURL", args.VCSURL, "Repository", args.Repository)
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

	VCS := &VCSGitlab{
		logger: logger,
	}
	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		shared.PluginTypeVCS: &vcs.VCSPlugin{Impl: VCS},
	}

	// logger.Debug("message from plugin", "foo", "bar")

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
