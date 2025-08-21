package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/go-github/v47/github"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"golang.org/x/oauth2"

	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
	"github.com/scan-io-git/scan-io/pkg/shared/httpclient"
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

// initializeGithubClient creates and initializes a new Github client.
func (g *VCSGithub) initializeGithubClient() (*github.Client, error) {
	var client *github.Client

	restyClient, err := httpclient.New(g.logger, g.globalConfig)
	if err != nil {
		g.logger.Error("failed to initialize HTTP client", "error", err)
		return nil, err
	}
	httpClient := restyClient.RestyClient.GetClient()

	// Support custom headers for Resty
	transport := &httpclient.CustomRoundTripper{
		BaseTransport: httpClient.Transport,
		Headers:       g.globalConfig.HTTPClient.CustomHeaders,
	}
	httpClient.Transport = transport

	if g.globalConfig.GithubPlugin.Token == "" {
		client = github.NewClient(httpClient)
	} else {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: g.globalConfig.GithubPlugin.Token},
		)
		oauthTransport := &oauth2.Transport{
			Source: ts,
			Base:   httpClient.Transport,
		}
		httpClient.Transport = oauthTransport
		client = github.NewClient(httpClient)
	}

	return client, nil
}

// listRepositoriesForProject fetches repositories for a given project.
func (g *VCSGithub) listRepositoriesForProject(client *github.Client, projectKey string) ([]shared.RepositoryParams, error) {
	// TODO: expand for a personal user namespace
	repositories, _, err := client.Repositories.ListByOrg(context.Background(), projectKey, &github.RepositoryListByOrgOptions{})
	if err != nil {
		g.logger.Error("failed to retrieve repositories for the project", "project", projectKey, "error", err)
		return nil, err
	}
	return toRepositoryParams(repositories), nil
}

// listRepositoriesForAllProjects fetches repositories for all projects.
func (g *VCSGithub) listRepositoriesForAllProjects(client *github.Client) ([]shared.RepositoryParams, error) {
	var result []shared.RepositoryParams

	// Retrieve the list of organizations for the authenticated user
	orgs, _, err := client.Organizations.List(context.Background(), "", nil)
	if err != nil {
		g.logger.Error("failed to list organizations", "error", err)
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	// Fetch repositories for each organization
	for _, org := range orgs {
		if org.Login == nil {
			g.logger.Warn("skipping organization with missing name")
			continue
		}

		orgName := *org.Login
		g.logger.Debug("fetching repositories for organization", "organization", orgName)

		repos, err := g.listRepositoriesForProject(client, orgName)
		if err != nil {
			g.logger.Error("failed to list repositories for organization", "organization", orgName, "error", err)
			continue
		}

		result = append(result, repos...)
	}

	// If no organizations were found, fallback to listing repositories for the authenticated user
	if len(orgs) == 0 {
		g.logger.Warn("no organizations found; searching for repositories of the current user")
		userRepos, _, err := client.Repositories.List(context.Background(), "", nil)
		if err != nil {
			g.logger.Error("failed to retrieve repositories for the current user", "error", err)
			return nil, fmt.Errorf("failed to retrieve repositories for the current user: %w", err)
		}

		result = append(result, toRepositoryParams(userRepos)...)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no repositories found for organizations or the current user")
	}
	return result, nil
}

// ListRepos handles listing repositories based on the provided VCSListReposRequest.
func (g *VCSGithub) ListRepositories(args shared.VCSListRepositoriesRequest) ([]shared.RepositoryParams, error) {
	g.logger.Debug("starting execution of list repositories function", "args", args)

	if err := g.validateList(&args); err != nil {
		g.logger.Error("validation failed for listing repositories operation", "error", err)
		return nil, err
	}

	client, err := g.initializeGithubClient()
	if err != nil {
		return nil, err
	}

	if len(args.RepoParam.Namespace) > 0 {
		return g.listRepositoriesForProject(client, args.RepoParam.Namespace)
	}

	return g.listRepositoriesForAllProjects(client)
}

// RetrievePRInformation handles retrieving PR information based on the provided VCSRetrievePRInformationRequest.
func (g *VCSGithub) RetrievePRInformation(args shared.VCSRetrievePRInformationRequest) (shared.PRParams, error) {
	g.logger.Debug("starting to retrieve information about a PR", "args", args)

	if err := g.validateRetrievePRInformation(&args); err != nil {
		g.logger.Error("validation failed for retrieving pull request information operation", "error", err)
		return shared.PRParams{}, err
	}

	client, err := g.initializeGithubClient()
	if err != nil {
		return shared.PRParams{}, err
	}
	prID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	prData, _, err := client.PullRequests.Get(context.Background(), args.RepoParam.Namespace, args.RepoParam.Repository, prID)
	if err != nil {
		g.logger.Error("failed to retrieve information about the PR", "PRID", prID, "error", err)
		return shared.PRParams{}, fmt.Errorf("failed to retrieve PR: %w", err)
	}

	return convertToPRParams(prData), nil
}

// AddRoleToPR handles adding a specified role to a PR based on the provided VCSAddRoleToPRRequest.
func (g *VCSGithub) AddRoleToPR(args shared.VCSAddRoleToPRRequest) (bool, error) {
	g.logger.Debug("starting to add a user to a PR", "args", args)

	if err := g.validateAddRoleToPR(&args); err != nil {
		g.logger.Error("validation failed for adding a user to PR operation", "error", err)
		return false, err
	}

	client, err := g.initializeGithubClient()
	if err != nil {
		return false, err
	}
	prID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	if _, _, err := client.PullRequests.Get(context.Background(), args.RepoParam.Namespace, args.RepoParam.Repository, prID); err != nil {
		g.logger.Error("failed to retrieve information about the PR", "PRID", prID, "error", err)
		return false, fmt.Errorf("failed to retrieve PR: %w", err)
	}

	switch strings.ToLower(args.Role) {
	case "assignee":
		assignees := []string{args.Login}
		if _, _, err := client.Issues.AddAssignees(context.Background(), args.RepoParam.Namespace, args.RepoParam.Repository, prID, assignees); err != nil {
			g.logger.Error("failed to add assignee to PR", "login", args.Login, "role", args.Role, "error", err)
			return false, fmt.Errorf("failed to add assignee to PR: %w", err)
		}

	case "reviewer":
		reviewers := []string{args.Login}
		req := &github.ReviewersRequest{
			Reviewers: reviewers,
		}

		if _, _, err := client.PullRequests.RequestReviewers(context.Background(), args.RepoParam.Namespace, args.RepoParam.Repository, prID, *req); err != nil {
			g.logger.Error("failed to add reviewer to PR", "error", err)
			return false, fmt.Errorf("failed to add reviewer to PR: %w", err)
		}
	default:
		g.logger.Error("unsupported role for PR operation", "role", args.Role)
		return false, fmt.Errorf("unsupported role: %s", args.Role)
	}

	g.logger.Info("user successfully added to the PR", "login", args.Login, "role", args.Role)
	return true, nil
}

// SetStatusOfPR handles setting the status of a PR based on the provided VCSSetStatusOfPRRequest.
func (g *VCSGithub) SetStatusOfPR(args shared.VCSSetStatusOfPRRequest) (bool, error) {
	g.logger.Debug("starting to change the status of a PR", "args", args)

	if err := g.validateSetStatusOfPR(&args); err != nil {
		g.logger.Error("validation failed for setting a status to PR operation", "error", err)
		return false, err
	}

	client, err := g.initializeGithubClient()
	if err != nil {
		return false, err
	}
	prID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	prData, _, err := client.PullRequests.Get(context.Background(), args.RepoParam.Namespace, args.RepoParam.Repository, prID)
	if err != nil {
		g.logger.Error("failed to retrieve information about the PR", "PRID", prID, "error", err)
		return false, fmt.Errorf("failed to retrieve PR: %w", err)
	}

	g.logger.Info("changing status of a particular PR", "PR", fmt.Sprintf("%v/%v/%v/%v", args.RepoParam.Domain, args.RepoParam.Namespace, args.RepoParam.Repository, prID))
	review := &github.PullRequestReviewRequest{
		Body:  github.String(args.Comment),
		Event: github.String(strings.ToUpper(args.Status)),
	}

	if _, _, err := client.PullRequests.CreateReview(context.Background(), args.RepoParam.Namespace, args.RepoParam.Repository, prID, review); err != nil {
		g.logger.Error("failed to set the status of the PR", "error", err)
		return false, fmt.Errorf("failed to set the status of the PR: %w", err)
	}

	g.logger.Info("PR successfully moved to status", "status", args.Status, "PRID", prID, "last_commit", prData.Head.GetSHA())
	return true, nil
}

// AddCommentToPR handles adding a comment to a specific pull request.
func (g *VCSGithub) AddCommentToPR(args shared.VCSAddCommentToPRRequest) (bool, error) {
	g.logger.Debug("starting to add a comment to a PR", "args", args)
	if err := g.validateAddCommentToPR(&args); err != nil {
		g.logger.Error("validation failed for adding a comment to PR operation", "error", err)
		return false, err
	}

	client, err := g.initializeGithubClient()
	if err != nil {
		return false, err
	}
	prID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	if _, _, err := client.PullRequests.Get(context.Background(), args.RepoParam.Namespace, args.RepoParam.Repository, prID); err != nil {

		g.logger.Error("failed to retrieve information about the PR", "PRID", prID, "error", err)
		return false, fmt.Errorf("failed to retrieve PR: %w", err)
	}
	g.logger.Info("commenting on a particular PR", "PR URL", fmt.Sprintf("%v/%v/%v/%v", args.RepoParam.Domain, args.RepoParam.Namespace, args.RepoParam.Repository, prID))
	comment := &github.IssueComment{
		Body: github.String(args.Comment),
	}
	if _, _, err = client.Issues.CreateComment(context.Background(), args.RepoParam.Namespace, args.RepoParam.Repository, prID, comment); err != nil {
		g.logger.Error("failed to add comment to PR", "error", err)
		return false, err
	}

	g.logger.Info("successfully added a comment to PR", "PR", prID)
	return true, nil
}

// fetchPR handles fetching pull request changes.
func (g *VCSGithub) fetchPR(args *shared.VCSFetchRequest) (string, error) {
	g.logger.Info("handling PR changes fetching")

	client, err := g.initializeGithubClient()
	if err != nil {
		return "", err
	}

	prID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	prData, _, err := client.PullRequests.Get(context.Background(), args.RepoParam.Namespace, args.RepoParam.Repository, prID)
	if err != nil {
		g.logger.Error("failed to retrieve information about the PR", "PRID", prID, "error", err)
		return "", fmt.Errorf("failed to retrieve PR: %w", err)
	}

	if len(args.Branch) == 0 {
		args.Branch = prData.Head.GetRef()

		if prData.Head.Repo.GetFork() {
			args.Branch = prData.Head.GetSHA()
			g.logger.Warn("found merging from a fork", "fromRefLink", prData.Head.Repo.GetHTMLURL())
			g.logger.Warn("changes will be taken from the default branch and latest commit", "latest_commit", prData.Head.GetSHA())
		}
	}

	changes, _, err := client.PullRequests.ListFiles(context.Background(), args.RepoParam.Namespace, args.RepoParam.Repository, prID, nil)
	if err != nil {
		g.logger.Error("failed to retrieve PR changes", "PRID", prID, "error", err)
		return "", err
	}
	g.logger.Debug("PR Data", prData)

	args.CloneURL = prData.Head.Repo.GetSSHURL()
	g.logger.Debug("starting to fetch PR code")

	pluginConfigMap, err := shared.StructToMap(g.globalConfig.GithubPlugin)
	if err != nil {
		g.logger.Error("error converting struct to map", "error", err)
		return "", err
	}

	clientGit, err := git.New(g.logger, g.globalConfig, pluginConfigMap, args)
	if err != nil {
		g.logger.Error("failed to initialize Git client", "error", err)
		return "", err
	}

	_, err = clientGit.CloneRepository(args)
	if err != nil {
		g.logger.Error("failed to clone repository", "error", err)
		return "", err
	}

	baseDestPath := config.GetPRTempPath(g.globalConfig, args.RepoParam.Domain, args.RepoParam.Namespace, args.RepoParam.Repository, prID)

	g.logger.Debug("copying files that have changed")
	for _, val := range changes {
		if !shared.ContainsSubstring(val.GetStatus(), []string{"added", "modified", "copied", "changed"}) {
			g.logger.Debug("skipping", "type", val.GetStatus(), "path", val.GetFilename())
			continue
		}

		srcPath := filepath.Join(args.TargetFolder, val.GetFilename())
		destPath := filepath.Join(baseDestPath, val.GetFilename())
		if err := files.Copy(srcPath, destPath); err != nil {
			g.logger.Error("error copying file", "error", err)
		}
	}

	if err := files.CopyDotFiles(args.TargetFolder, baseDestPath, g.logger); err != nil {
		return "", err
	}

	g.logger.Info("files for PR scan are copied", "folder", baseDestPath)
	return baseDestPath, nil
}

// Fetch retrieves code based on the provided VCSFetchRequest.
func (g *VCSGithub) Fetch(args shared.VCSFetchRequest) (shared.VCSFetchResponse, error) {
	var result shared.VCSFetchResponse

	if err := g.validateFetch(&args); err != nil {
		g.logger.Error("validation failed for fetch operation", "error", err)
		return shared.VCSFetchResponse{}, err
	}

	switch args.Mode {
	case "fetchPR":
		path, err := g.fetchPR(&args)
		if err != nil {
			g.logger.Error("failed to fetch pull request")
			return result, err
		}
		result.Path = path

	default:
		pluginConfigMap, err := shared.StructToMap(g.globalConfig.GithubPlugin)
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

// CreateIssue creates a new GitHub issue using the provided request.
//
// Parameters:
//
//	args - VCSIssueCreationRequest containing repository details and issue content
//
// Examples:
//   - Create an issue:
//     req := shared.VCSIssueCreationRequest{
//     RepoParam: shared.RepositoryParams{
//     Namespace: "octocat",
//     Repository: "hello-world",
//     },
//     Title: "New Feature Request",
//     Body: "Please add support for...",
//     }
//     issueNumber, err := githubClient.CreateIssue(req)
//
// Returns:
//   - The number of the created issue
//   - An error if the issue creation fails
func (g *VCSGithub) CreateIssue(args shared.VCSIssueCreationRequest) (int, error) {
	client, err := g.initializeGithubClient()
	if err != nil {
		return 0, fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	issue := &github.IssueRequest{
		Title: github.String(args.Title),
		Body:  github.String(args.Body),
	}

	ctx := context.Background()
	createdIssue, _, err := client.Issues.Create(ctx, args.RepoParam.Namespace, args.RepoParam.Repository, issue)
	if err != nil {
		return 0, fmt.Errorf("failed to create GitHub issue: %w", err)
	}

	return createdIssue.GetNumber(), nil
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
