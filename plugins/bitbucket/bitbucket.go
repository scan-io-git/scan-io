package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"github.com/scan-io-git/scan-io/internal/bitbucket"
	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"

	ftutils "github.com/scan-io-git/scan-io/internal/fetcherutils"
	utils "github.com/scan-io-git/scan-io/internal/utils"
)

const PluginName = "bitbucket"

// TODO: Wrap it in a custom error handler to add to the stack trace.
// Metadata of the plugin
var (
	Version       = "unknown"
	GolangVersion = "unknown"
	BuildTime     = "unknown"
)

// VCSBitbucket implements VCS operations for Bitbucket.
type VCSBitbucket struct {
	logger       hclog.Logger
	globalConfig *config.Config
	name         string
}

// newVCSBitbucket creates a new instance of VCSBitbucket.
func newVCSBitbucket(logger hclog.Logger) *VCSBitbucket {
	return &VCSBitbucket{
		logger: logger,
		name:   PluginName,
	}
}

// setGlobalConfig sets the global configuration for the VCSBitbucket instance.
func (g *VCSBitbucket) setGlobalConfig(globalConfig *config.Config) {
	g.globalConfig = globalConfig
}

// initializeBitbucketClient creates and initializes a new Bitbucket client.
func (g *VCSBitbucket) initializeBitbucketClient(domain string) (*bitbucket.Client, error) {
	authInfo := bitbucket.AuthInfo{
		Username: g.globalConfig.BitbucketPlugin.Username,
		Token:    g.globalConfig.BitbucketPlugin.Token,
	}

	client, err := bitbucket.New(g.globalConfig, g.logger, domain, authInfo)
	if err != nil {
		g.logger.Error("initialization of Bitbucket client failed", "error", err)
		return nil, err
	}
	return client, nil
}

// listRepositoriesForProject fetches repositories for a given project or user.
func (g *VCSBitbucket) listRepositoriesForProject(client *bitbucket.Client, projectKey string) ([]shared.RepositoryParams, error) {
	var repositories *[]bitbucket.Repository
	var err error

	switch {
	case strings.HasPrefix(projectKey, "users/"):
		repositories, err = client.Repositories.ListUserRepos(strings.TrimPrefix(projectKey, "users/"))
	default:
		repositories, err = client.Repositories.List(projectKey)
	}

	if err != nil {
		g.logger.Error("failed to retrieve repositories", "projectKey", projectKey, "error", err)
		return nil, fmt.Errorf("failed to retrieve repositories for %q: %w", projectKey, err)
	}

	return toRepositoryParams(repositories), nil
}

// listRepositoriesForAllProjects fetches repositories for all projects.
func (g *VCSBitbucket) listRepositoriesForAllProjects(client *bitbucket.Client) ([]shared.NamespaceParams, int, error) {
	var gRepoCount int
	projects, err := client.Projects.List()
	if err != nil {
		g.logger.Error("failed to list all projects", "error", err)
		return nil, gRepoCount, err
	}

	if projects == nil {
		return nil, gRepoCount, fmt.Errorf("no projects found")
	}

	var result []shared.NamespaceParams
	for _, project := range *projects {
		repos, err := g.listRepositoriesForProject(client, project.Key)
		if err != nil {
			g.logger.Error("failed to list repositories for project, continuing...", "project", project.Key, "error", err)
			continue
		}
		repoCount := len(repos)
		gRepoCount += repoCount
		result = append(result, shared.NamespaceParams{Namespace: project.Key, RepositoryCount: repoCount, Repositories: repos})
	}

	if len(result) == 0 {
		return nil, gRepoCount, fmt.Errorf("list of repositories is empty")
	}
	return result, gRepoCount, nil
}

// ListRepos handles listing repositories based on the provided VCSListReposRequest.
func (g *VCSBitbucket) ListRepositories(args shared.VCSListRepositoriesRequest) ([]shared.VCSParams, error) {
	var result []shared.VCSParams
	g.logger.Debug("starting execution of list repositories function", "args", args)

	if err := g.validateList(&args); err != nil {
		g.logger.Error("validation failed for listing repositories operation", "error", err)
		return nil, err
	}

	client, err := g.initializeBitbucketClient(args.RepoParam.Domain)
	if err != nil {
		return nil, err
	}

	if len(args.RepoParam.Namespace) > 0 {
		repos, err := g.listRepositoriesForProject(client, args.RepoParam.Namespace)
		if err != nil {
			return nil, err
		}
		result = append(result, shared.VCSParams{
			Domain:          args.VCSDomain,
			NamespaceCount:  1,
			RepositoryCount: len(repos),
			Namespaces: []shared.NamespaceParams{
				{
					Namespace:       args.RepoParam.Namespace,
					RepositoryCount: len(repos),
					Repositories:    repos,
				},
			}})
		return result, nil
	}

	projects, repoCount, err := g.listRepositoriesForAllProjects(client)
	if err != nil {
		return nil, err
	}
	result = append(result, shared.VCSParams{
		Domain:          args.VCSDomain,
		NamespaceCount:  len(projects),
		RepositoryCount: repoCount,
		Namespaces:      projects,
	})
	return result, nil
}

// RetrievePRInformation handles retrieving PR information based on the provided VCSRetrievePRInformationRequest.
func (g *VCSBitbucket) RetrievePRInformation(args shared.VCSRetrievePRInformationRequest) (shared.PRParams, error) {
	g.logger.Debug("starting to retrieve information about a PR", "args", args)

	if err := g.validateRetrievePRInformation(&args); err != nil {
		g.logger.Error("validation failed for retrieving pull request information operation", "error", err)
		return shared.PRParams{}, err
	}

	client, err := g.initializeBitbucketClient(args.RepoParam.Domain)
	if err != nil {
		return shared.PRParams{}, err
	}

	prID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	prData, err := client.PullRequests.Get(args.RepoParam.Namespace, args.RepoParam.Repository, prID)
	if err != nil {
		g.logger.Error("failed to retrieve information about the PR", "PRID", prID, "error", err)
		return shared.PRParams{}, err
	}

	return convertToPRParams(prData), nil
}

// AddRoleToPR handles adding a specified role to a PR based on the provided VCSAddRoleToPRRequest.
func (g *VCSBitbucket) AddRoleToPR(args shared.VCSAddRoleToPRRequest) (bool, error) {
	g.logger.Debug("starting to add a user to a PR", "args", args)

	if err := g.validateAddRoleToPR(&args); err != nil {
		g.logger.Error("validation failed for adding a user to PR operation", "error", err)
		return false, err
	}

	client, err := g.initializeBitbucketClient(args.RepoParam.Domain)
	if err != nil {
		return false, err
	}

	prID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	prData, err := client.PullRequests.Get(args.RepoParam.Namespace, args.RepoParam.Repository, prID)
	if err != nil {
		g.logger.Error("failed to retrieve information about the PR", "PRID", prID, "error", err)
		return false, err
	}

	if _, err := prData.AddRole(args.Role, args.Login); err != nil {
		g.logger.Error("failed to add role to PR", "login", args.Login, "role", args.Role, "error", err)
		return false, err
	}

	g.logger.Info("user successfully added to the PR", "login", args.Login, "role", args.Role)
	return true, nil
}

// SetStatusOfPR handles setting the status of a PR based on the provided VCSSetStatusOfPRRequest.
func (g *VCSBitbucket) SetStatusOfPR(args shared.VCSSetStatusOfPRRequest) (bool, error) {
	g.logger.Debug("starting to change the status of a PR", "args", args)

	if err := g.validateSetStatusOfPR(&args); err != nil {
		g.logger.Error("validation failed for setting a status to PR operation", "error", err)
		return false, err
	}

	client, err := g.initializeBitbucketClient(args.RepoParam.Domain)
	if err != nil {
		return false, err
	}

	prID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	prData, err := client.PullRequests.Get(args.RepoParam.Namespace, args.RepoParam.Repository, prID)
	if err != nil {
		g.logger.Error("failed to retrieve information about the PR", "PRID", prID, "error", err)
		return false, err
	}
	g.logger.Info("changing status of a particular PR", "PR", fmt.Sprintf("%v/%v/%v/%v", args.RepoParam.Domain, args.RepoParam.Namespace, args.RepoParam.Repository, prID))

	_, err = prData.SetStatus(args.Status, args.Login)
	if err != nil {
		g.logger.Error("failed to set the status of the PR", "error", err)
		return false, err
	}

	g.logger.Info("PR successfully moved to status", "status", args.Status, "PRID", prID, "last_commit", prData.Author.LastReviewedCommit)
	return true, nil
}

// AddCommentToPR handles adding a comment to a specific pull request.
func (g *VCSBitbucket) AddCommentToPR(args shared.VCSAddCommentToPRRequest) (bool, error) {
	g.logger.Debug("starting to add a comment to a PR", "args", args)

	if err := g.validateAddCommentToPR(&args); err != nil {
		g.logger.Error("validation failed for adding a comment to PR operation", "error", err)
		return false, err
	}

	client, err := g.initializeBitbucketClient(args.RepoParam.Domain)
	if err != nil {
		return false, err
	}

	prID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	prData, err := client.PullRequests.Get(args.RepoParam.Namespace, args.RepoParam.Repository, prID)
	if err != nil {
		g.logger.Error("failed to retrieve information about the PR", "PRID", prID, "error", err)
		return false, err
	}
	g.logger.Info("commenting on a particular PR", "PR URL", fmt.Sprintf("%v/%v/%v/%v", args.RepoParam.Domain, args.RepoParam.Namespace, args.RepoParam.Repository, prID))

	if _, err := prData.AddComment(args.Comment.Body, args.Comment.FilePaths); err != nil {
		g.logger.Error("failed to add comment to PR", "error", err)
		return false, err
	}

	g.logger.Info("successfully added a comment to PR", "PR", prID)
	return true, nil
}

// CreateIssue is not implemented for Bitbucket yet. Added to satisfy the VCS interface.
func (g *VCSBitbucket) CreateIssue(args shared.VCSIssueCreationRequest) (int, error) {
	g.logger.Error("CreateIssue not implemented for Bitbucket", "repo", fmt.Sprintf("%s/%s", args.RepoParam.Namespace, args.RepoParam.Repository))
	return 0, fmt.Errorf("CreateIssue not implemented for Bitbucket")
}

// ListIssues is not implemented for Bitbucket yet. Added to satisfy the VCS interface.
func (g *VCSBitbucket) ListIssues(args shared.VCSListIssuesRequest) ([]shared.IssueParams, error) {
	g.logger.Error("ListIssues not implemented for Bitbucket", "repo", fmt.Sprintf("%s/%s", args.RepoParam.Namespace, args.RepoParam.Repository))
	return nil, fmt.Errorf("ListIssues not implemented for Bitbucket")
}

// UpdateIssue is not implemented for Bitbucket yet. Added to satisfy the VCS interface.
func (g *VCSBitbucket) UpdateIssue(args shared.VCSIssueUpdateRequest) (bool, error) {
	g.logger.Error("UpdateIssue not implemented for Bitbucket", "repo", fmt.Sprintf("%s/%s", args.RepoParam.Namespace, args.RepoParam.Repository), "number", args.Number)
	return false, fmt.Errorf("UpdateIssue not implemented for Bitbucket")
}

// CreateIssueComment is not implemented for Bitbucket yet. Added to satisfy the VCS interface.
func (g *VCSBitbucket) CreateIssueComment(args shared.VCSCreateIssueCommentRequest) (bool, error) {
	g.logger.Error("CreateIssueComment not implemented for Bitbucket", "repo", fmt.Sprintf("%s/%s", args.RepoParam.Namespace, args.RepoParam.Repository), "number", args.Number)
	return false, fmt.Errorf("CreateIssueComment not implemented for Bitbucket")
}

func (g *VCSBitbucket) AddInLineCommentsList(req shared.VCSAddInLineCommentsListRequest) (bool, error) {
	g.logger.Debug("adding sarif comments to PR", "args", req)

	if err := g.validateAddSarifComments(&req); err != nil {
		g.logger.Error("sarif comment validation failed", "error", err)
		return false, err
	}

	client, err := g.initializeBitbucketClient(req.RepoParam.Domain)
	if err != nil {
		return false, err
	}

	prID, _ := strconv.Atoi(req.RepoParam.PullRequestID)
	pr, err := client.PullRequests.Get(req.RepoParam.Namespace, req.RepoParam.Repository, prID)
	if err != nil {
		g.logger.Error("failed to retrieve PR", "PRID", prID, "error", err)
		return false, err
	}

	for idx, comment := range req.Comments {
		if err := g.postInlineComment(pr, idx, comment); err != nil {
			return false, err
		}
	}

	if summary := strings.TrimSpace(req.Summary); summary != "" {
		if _, err := pr.AddComment(summary, nil); err != nil {
			g.logger.Error("failed to add sarif summary comment", "error", err)
			return false, err
		}
	}

	return true, nil
}

func (g *VCSBitbucket) postInlineComment(pr *bitbucket.PullRequest, idx int, comment shared.Comment) error {
	text := strings.TrimSpace(comment.Body)
	if text == "" {
		return nil
	}

	var addErr error
	if comment.Path != "" && comment.Line > 0 {
		_, addErr = pr.AddInlineComment(
			text,
			comment.Path,
			comment.Line,
			bitbucket.InlineCommentOptions{
				LineType:    bitbucket.LineTypeAdded,
				FileType:    bitbucket.FileTypeTo,
				DiffType:    bitbucket.DiffTypeEffective,
				Attachments: comment.FilePaths,
			},
		)
	} else {
		_, addErr = pr.AddComment(text, comment.FilePaths)
	}

	if addErr != nil {
		g.logger.Error("failed to add comment",
			"index", idx,
			"path", comment.Path,
			"line", comment.Line,
			"error", addErr,
		)
		return addErr
	}

	g.logger.Debug("sarif comment posted", "index", idx, "path", comment.Path, "line", comment.Line)
	return nil
}

// fetchPR handles fetching pull request changes.
func (g *VCSBitbucket) fetchPR(args *shared.VCSFetchRequest) (string, error) {
	g.logger.Info("handling PR changes fetching")

	domain, err := utils.GetDomain(args.CloneURL)
	if err != nil {
		return "", err
	}

	client, err := g.initializeBitbucketClient(domain)
	if err != nil {
		return "", err
	}

	prID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	prData, err := client.PullRequests.Get(args.RepoParam.Namespace, args.RepoParam.Repository, prID)
	if err != nil {
		g.logger.Error("failed to retrieve information about the PR", "PRID", prID, "error", err)
		return "", err
	}

	fromRefLink := prData.FromReference.Repository.Links.Self[0].Href
	u, err := url.ParseRequestURI(fromRefLink)
	if err != nil {
		return "", err
	}

	args.Branch = prData.FromReference.ID

	pathDirs := vcsurl.GetPathDirs(u.Path)
	if pathDirs[0] == "users" {
		args.Branch = prData.FromReference.LatestCommit
		g.logger.Warn("found merging from user personal repository",
			"fromRefLink", fromRefLink,
		)
		g.logger.Warn("pr will be fetched as a detached latest commit",
			"latest_commit", prData.FromReference.LatestCommit,
		)
	} else if args.FetchMode == ftutils.PRCommitMode {
		args.Branch = prData.FromReference.LatestCommit
		g.logger.Info("fetching pull request by commit",
			"latest_commit", prData.FromReference.LatestCommit,
		)
	}

	changes, err := prData.GetChanges()
	if err != nil {
		g.logger.Error("failed to retrieve PR changes", "PRID", prID, "error", err)
		return "", err
	}
	g.logger.Debug("PR", "data", prData)

	newCloneURL := findCloneURL(prData, g.logger)
	if newCloneURL != "" {
		args.CloneURL = newCloneURL
	} else {
		g.logger.Warn("no valid clone URL found in both FromReference and ToReference repositories")
	}

	g.logger.Debug("starting to fetch PR code")

	pluginConfigMap, err := shared.StructToMap(g.globalConfig.BitbucketPlugin)
	if err != nil {
		g.logger.Error("error converting struct to map", "error", err)
		return "", err
	}

	clientGit, err := git.New(g.logger, g.globalConfig, pluginConfigMap, args, PluginName)
	if err != nil {
		g.logger.Error("failed to initialize Git client", "error", err)
		return "", err
	}

	// TODO: Fix a strange bug when it fetches only PR changes without all other files in case of PR fetch
	_, err = clientGit.CloneRepository(args)
	if err != nil {
		g.logger.Error("failed to clone repository", "error", err)
		return "", err
	}

	baseDestPath := config.GetPRTempPath(g.globalConfig, args.RepoParam.Domain, args.RepoParam.Namespace, args.RepoParam.Repository, prID)

	g.logger.Debug("copying files that have changed")
	for _, val := range *changes {
		if !shared.ContainsSubstring(val.Type, bitbucket.ChangeTypes) {
			g.logger.Debug("skipping", "type", val.Type, "path", val.Path.ToString)
			continue
		}

		srcPath := filepath.Join(args.TargetFolder, val.Path.ToString)
		destPath := filepath.Join(baseDestPath, val.Path.ToString)
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

// findCloneURL searches for a valid clone URL, first in the fromRef, then in the toRef.
func findCloneURL(prData *bitbucket.PullRequest, logger hclog.Logger) string {
	var checkCloneURL func(repo bitbucket.Repository) (string, bool)
	checkCloneURL = func(repo bitbucket.Repository) (string, bool) {
		// Check if the main repository has a valid clone URL
		if len(repo.Links.Clone) > 0 && repo.Links.Clone[0].Href != "" {
			logger.Debug("found repository clone URL from API response")
			return repo.Links.Clone[0].Href, true
		}

		// Recursively check nested origins, if they exist
		if repo.Origin != nil {
			logger.Debug("searching origin clone URL from API response")
			return checkCloneURL(*repo.Origin)
		}

		// No valid clone URL found in this repository or its origins
		return "", false
	}

	// Check the FromReference.Repository (main repository)
	mainRepo := prData.FromReference.Repository
	if cloneURL, found := checkCloneURL(mainRepo); found {
		logger.Info("found URL from the FromReference API response", "CloneURL", cloneURL)
		return cloneURL
	}

	// If not found, check the ToReference.Repository
	toRepo := prData.ToReference.Repository
	if cloneURL, found := checkCloneURL(toRepo); found {
		logger.Info("using clone URL from the ToReference API response", "CloneURL", cloneURL)
		return cloneURL
	}

	// If no clone URL is found in either, return an error
	return ""
}

// Fetch retrieves code based on the provided VCSFetchRequest.
func (g *VCSBitbucket) Fetch(args shared.VCSFetchRequest) (shared.VCSFetchResponse, error) {
	var result shared.VCSFetchResponse

	if err := g.validateFetch(&args); err != nil {
		g.logger.Error("validation failed for fetch operation", "error", err)
		return shared.VCSFetchResponse{}, err
	}

	switch args.FetchMode {
	case ftutils.PRBranchMode, ftutils.PRRefMode, ftutils.PRCommitMode:
		path, err := g.fetchPR(&args)
		if err != nil {
			g.logger.Error("failed to fetch pull request")
			return result, err
		}
		result.Path = path

	default:
		pluginConfigMap, err := shared.StructToMap(g.globalConfig.BitbucketPlugin)
		if err != nil {
			g.logger.Error("error converting struct to map", "error", err)
			return result, err
		}

		clientGit, err := git.New(g.logger, g.globalConfig, pluginConfigMap, &args, PluginName)
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

// Setup initializes the global configuration for the VCSBitbucket instance.
func (g *VCSBitbucket) Setup(configData config.Config) (bool, error) {
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

	bitbucketInstance := newVCSBitbucket(logger)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			shared.PluginTypeVCS: &shared.VCSPlugin{Impl: bitbucketInstance},
		},
		Logger: logger,
	})
}
