package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"github.com/scan-io-git/scan-io/internal/bitbucket"
	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

type VCSBitbucket struct {
	logger       hclog.Logger
	globalConfig *config.Config
}

// listRepositoriesForProject fetches repositories for a given project.
func (g *VCSBitbucket) listRepositoriesForProject(client *bitbucket.Client, project string) ([]shared.RepositoryParams, error) {
	repositories, err := client.Repositories.List(project)
	if err != nil {
		g.logger.Error("failed to retrieve repository for the project", "project", project, "error", err)
		return nil, err
	}
	return toRepositoryParams(repositories), nil
}

// listReposForAllProjects fetches repositories for all projects.
func (g *VCSBitbucket) listRepositoriesForAllProjects(client *bitbucket.Client) ([]shared.RepositoryParams, error) {
	// Fetch all projects from the Bitbucket API
	projects, err := client.Projects.List()
	if err != nil {
		g.logger.Error("failed to list all projects", "error", err)
		return nil, err
	}

	var result []shared.RepositoryParams
	for _, project := range *projects {
		repos, err := g.listRepositoriesForProject(client, project.Key)
		if err != nil {
			g.logger.Error("Failed to list repositories for project. Continue...", "project", project.Key, "error", err)
			continue
		}
		result = append(result, repos...)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("list of repositories is empty")
	}
	return result, nil
}

// ListRepos handles listing repositories based on the provided VCSListReposRequest.
// It distinguishes between listing repos for a specific project or all projects.
func (g *VCSBitbucket) ListRepos(args shared.VCSListReposRequest) ([]shared.RepositoryParams, error) {
	g.logger.Debug("Starting execution of an all-repositories listing function", "args", args)
	if err := g.validateList(&args); err != nil {
		g.logger.Error("validation failed for listing repositories operation", "error", err)
		return nil, err
	}

	authInfo := bitbucket.AuthInfo{
		Username: g.globalConfig.BitbucketPlugin.BitbucketUsername,
		Token:    g.globalConfig.BitbucketPlugin.BitbucketToken,
	}

	client, err := bitbucket.New(g.logger, args.VCSURL, authInfo, g.globalConfig)
	if err != nil {
		g.logger.Error("initialization Bitbucket client failed", "error", err)
		return nil, err
	}

	var result []shared.RepositoryParams
	if len(args.Namespace) > 0 {
		result, err = g.listRepositoriesForProject(client, args.Namespace)
		if err != nil {
			g.logger.Error("The particular repository function is failed", "error", err)
			return nil, err
		}
		return result, nil
	} else {
		result, err = g.listRepositoriesForAllProjects(client)
		if err != nil {
			g.logger.Error("The particular repository function is failed", "error", err)
			return nil, err
		}
		return result, nil
	}
}

// RetrivePRInformation handles retriving PR information based on the provided VCSRetrivePRInformationRequest.
func (g *VCSBitbucket) RetrivePRInformation(args shared.VCSRetrivePRInformationRequest) (shared.PRParams, error) {
	g.logger.Debug("Starting retrive information about a PR", "args", args)

	if err := g.validateRetrivePRInformation(&args); err != nil {
		g.logger.Error("validation failed for retrieving pull request information operation", "error", err)
		return shared.PRParams{}, err
	}

	authInfo := bitbucket.AuthInfo{
		Username: g.globalConfig.BitbucketPlugin.BitbucketUsername,
		Token:    g.globalConfig.BitbucketPlugin.BitbucketToken,
	}
	client, err := bitbucket.New(g.logger, args.VCSURL, authInfo, g.globalConfig)
	if err != nil {
		g.logger.Error("initialization Bitbucket client failed", "error", err)
		return shared.PRParams{}, err
	}

	prData, err := client.PullRequests.Get(args.Namespace, args.Repository, args.PullRequestId)
	if err != nil {
		g.logger.Error("Failed to retrieve information about the PR", "PRID", args.PullRequestId, "error", err)
		return shared.PRParams{}, err
	}

	return convertToPRParams(prData), nil
}

// AddRoleToPR handles adding specified role to PR based on the provided VCSAddRoleToPRRequest.
func (g *VCSBitbucket) AddRoleToPR(args shared.VCSAddRoleToPRRequest) (bool, error) {
	g.logger.Debug("Starting to add a reviewer to a PR", "args", args)

	if err := g.validateAddRoleToPR(&args); err != nil {
		g.logger.Error("validation failed for adding a user to PR operation", "error", err)
		return false, err
	}

	authInfo := bitbucket.AuthInfo{
		Username: g.globalConfig.BitbucketPlugin.BitbucketUsername,
		Token:    g.globalConfig.BitbucketPlugin.BitbucketToken,
	}

	client, err := bitbucket.New(g.logger, args.VCSURL, authInfo, g.globalConfig)
	if err != nil {
		g.logger.Error("initialization Bitbucket client failed", "error", err)
		return false, err
	}

	prData, err := client.PullRequests.Get(args.Namespace, args.Repository, args.PullRequestId)
	if err != nil {
		g.logger.Error("Failed to retrieve information about the PR", "PRID", args.PullRequestId, "error", err)
		return false, err
	}

	if _, err := prData.AddRole(args.Role, args.Login); err != nil {
		g.logger.Error("Failed to add role to PR", "error", err)
		return false, err
	}

	g.logger.Info("User successfully added to the PR", "user", args.Login, "role", args.Role)
	return true, nil
}

// SetStatusOfPR handles setting a status of PR based on the provided VCSSetStatusOfPRRequest.
func (g *VCSBitbucket) SetStatusOfPR(args shared.VCSSetStatusOfPRRequest) (bool, error) {
	g.logger.Debug("Starting changing a status of PR", "args", args)

	if err := g.validateSetStatusOfPR(&args); err != nil {
		g.logger.Error("validation failed for setting a status to PR operation", "error", err)
		return false, err
	}

	authInfo := bitbucket.AuthInfo{
		Username: g.globalConfig.BitbucketPlugin.BitbucketUsername,
		Token:    g.globalConfig.BitbucketPlugin.BitbucketToken,
	}

	client, err := bitbucket.New(g.logger, args.VCSURL, authInfo, g.globalConfig)
	if err != nil {
		g.logger.Error("initialization Bitbucket client failed", "error", err)
		return false, err
	}

	prData, err := client.PullRequests.Get(args.Namespace, args.Repository, args.PullRequestId)
	if err != nil {
		g.logger.Error("Failed to retrieve information about the PR", "PRID", args.PullRequestId, "error", err)
		return false, err
	}
	g.logger.Info("Changing status of a particular PR", "PR", fmt.Sprintf("%v/%v/%v/%v", args.VCSURL, args.Namespace, args.Repository, args.PullRequestId))

	user, err := prData.SetStatus(args.Status, args.Login)
	if err != nil {
		g.logger.Error("Failed to set the status of the PR", "error", err)
		return false, err
	}

	g.logger.Info("PR successfully moved to status", "status", args.Status, "PR_id", args.PullRequestId, "last_commit", user.Author.LastReviewedCommit)
	return true, nil
}

// AddCommentToPR handles adding a comment to a specific pull request.
func (g *VCSBitbucket) AddComment(args shared.VCSAddCommentToPRRequest) (bool, error) {
	g.logger.Debug("starting to add a comment to a PR", "args", args)

	if err := g.validateAddComment(&args); err != nil {
		g.logger.Error("validation failed for adding a comment to PR operation", "error", err)
		return false, err
	}

	authInfo := bitbucket.AuthInfo{
		Username: g.globalConfig.BitbucketPlugin.BitbucketUsername,
		Token:    g.globalConfig.BitbucketPlugin.BitbucketToken,
	}

	client, err := bitbucket.New(g.logger, args.VCSURL, authInfo, g.globalConfig)
	if err != nil {
		g.logger.Error("initialization Bitbucket client failed", "error", err)
		return false, err
	}

	prData, err := client.PullRequests.Get(args.Namespace, args.Repository, args.PullRequestId)
	if err != nil {
		g.logger.Error("Failed to retrieve information about the PR", "PRID", args.PullRequestId, "error", err)
		return false, err
	}
	g.logger.Info("Commenting on a particular PR", "PR URL", fmt.Sprintf("%v/%v/%v/%v", args.VCSURL, args.Namespace, args.Repository, args.PullRequestId))

	if _, err := prData.AddComment(args.Comment); err != nil {
		g.logger.Error("Failed to add comment to PR", "error", err)
		return false, err
	}

	g.logger.Info("Comment successfully added")
	return true, nil
}

func (g *VCSBitbucket) fetchPR(args *shared.VCSFetchRequest) (string, error) {
	g.logger.Info("handling PR changes fetching")

	authInfo := bitbucket.AuthInfo{
		Username: g.globalConfig.BitbucketPlugin.BitbucketUsername,
		Token:    g.globalConfig.BitbucketPlugin.BitbucketToken,
	}
	client, err := bitbucket.New(g.logger, args.RepoParam.VCSURL, authInfo, g.globalConfig)
	if err != nil {
		g.logger.Error("initialization Bitbucket client failed", "error", err)
		return "", err
	}

	// TODO change the RepoParam structure to int
	prId, _ := strconv.Atoi(args.RepoParam.PRID)
	prData, err := client.PullRequests.Get(args.RepoParam.Namespace, args.RepoParam.RepoName, prId)
	if err != nil {
		g.logger.Error("failed to retrieve information about the PR", "PRID", prId, "error", err)
		return "", err
	}

	changes, err := prData.GetChanges()
	if err != nil {
		g.logger.Error("failed to PR changes", "PRID", prId, "error", err)
		return "", err
	}

	g.logger.Debug("strating to fetch PR code")
	// Setting a branch from PR data for fetching
	args.Branch = prData.FromReference.DisplayID

	// TODO Fix a strange bug when it fetches only pr changes without all other files in case of PR fetch
	_, err = git.CloneRepository(g.logger, g.globalConfig, args)
	if err != nil {
		g.logger.Error("failed to clone repository", "error", err)
		return "", err
	}

	baseDestPath := shared.GetPRTempPath(g.logger, args.RepoParam.VCSURL, (args.RepoParam.Namespace), args.RepoParam.RepoName, prId)

	g.logger.Debug("copying files that have changed")
	for _, val := range *changes {
		if !shared.ContainsSubstring(val.Type, bitbucket.ChangeTypes) {
			g.logger.Debug("Skipping", "type", val.Type, "path", val.Path.ToString)
			continue
		}

		srcPath := filepath.Join(args.TargetFolder, val.Path.ToString)
		destPath := filepath.Join(baseDestPath, val.Path.ToString)
		err := shared.Copy(srcPath, destPath)
		if err != nil {
			g.logger.Error("error copying file", "error", err)
		}
	}

	if err := shared.CopyDotFiles(args.TargetFolder, baseDestPath, g.logger); err != nil {
		return "", err
	}

	g.logger.Info("files for PR scan are copied", "folder", baseDestPath)
	return baseDestPath, nil
}

func (g *VCSBitbucket) Fetch(args shared.VCSFetchRequest) (shared.VCSFetchResponse, error) {
	var result shared.VCSFetchResponse

	if err := g.validateFetch(&args); err != nil {
		g.logger.Error("validation failed for fetch operation", "error", err)
		return shared.VCSFetchResponse{}, err
	}

	switch args.Mode {
	case "PRscan":
		// Fetching for pull request scanning
		path, err := g.fetchPR(&args)
		if err != nil {
			g.logger.Error("failed to fetch pull request")
			return result, err
		}
		result.Path = path

	default:
		// Default fetching operation
		path, err := git.CloneRepository(g.logger, g.globalConfig, &args)
		if err != nil {
			g.logger.Error("failed to clone repository", "error", err)
			return result, err
		}
		result.Path = path
	}

	return result, nil
}

func (g *VCSBitbucket) Setup(configData config.Config) (bool, error) {
	g.globalConfig = &configData
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

	VCS := &VCSBitbucket{
		logger: logger,
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			shared.PluginTypeVCS: &shared.VCSPlugin{Impl: VCS},
		},
		Logger: logger,
	})
}
