package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"gitlab.com/gitlab-org/api/client-go"

	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
	"github.com/scan-io-git/scan-io/pkg/shared/httpclient"

	ftutils "github.com/scan-io-git/scan-io/internal/fetcherutils"
)

const PluginName = "gitlab"

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
	name         string
}

// newVCSGitlab creates a new instance of VCSGitlab.
func newVCSGitlab(logger hclog.Logger) *VCSGitlab {
	return &VCSGitlab{
		logger: logger,
		name:   PluginName,
	}
}

// setGlobalConfig sets the global configuration for the VCSGitlab instance.
func (g *VCSGitlab) setGlobalConfig(globalConfig *config.Config) {
	g.globalConfig = globalConfig
}

// initializeGitlabClient creates and initializes a new Gitlab client.
func (g *VCSGitlab) initializeGitlabClient(vcsBaseURL string) (*gitlab.Client, error) {
	baseURL := fmt.Sprintf("https://%s/api/v4", vcsBaseURL)

	restyClient, err := httpclient.New(g.logger, g.globalConfig)
	if err != nil {
		g.logger.Error("failed to initialize HTTP client", "error", err)
		return nil, fmt.Errorf("failed to initialize HTTP client: %w", err)
	}
	httpClient := restyClient.RestyClient.GetClient()

	// Support custom headers for Resty
	httpClient.Transport = &httpclient.CustomRoundTripper{
		BaseTransport: httpClient.Transport,
		Headers:       g.globalConfig.HTTPClient.CustomHeaders,
	}

	client, err := gitlab.NewClient(
		g.globalConfig.GitlabPlugin.Token,
		gitlab.WithBaseURL(baseURL),
		gitlab.WithHTTPClient(httpClient))
	if err != nil {
		g.logger.Error("initialization of GitLab client failed", "error", err)
		return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	g.logger.Debug("GitLab client initialized successfully", "baseURL", baseURL)
	return client, nil
}

// Helper function to handle paginated API calls.
func fetchPaginated[T any](fetchFunc func(*gitlab.ListOptions) ([]T, *gitlab.Response, error)) ([]T, error) {
	var results []T
	options := &gitlab.ListOptions{Page: 1, PerPage: 30}

	for {
		items, resp, err := fetchFunc(options)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch paginated results: %w", err)
		}
		results = append(results, items...)
		if resp.NextPage == 0 {
			break
		}
		options.Page = resp.NextPage
	}
	return results, nil
}

// isLanguagePresent checks if the specified language is present in the language map.
func isLanguagePresent(languages *gitlab.ProjectLanguages, language string) bool {
	for lang := range *languages {
		if strings.EqualFold(lang, language) {
			return true
		}
	}
	return false
}

// listProjectsForGroup fetches repositories for a given group and supports searching in subgorups
func (g *VCSGitlab) listProjectsForGroup(client *gitlab.Client, groupKey, language string) ([]shared.RepositoryParams, error) {
	g.logger.Debug("listing projects for group", "groupKey", groupKey, "language", language)

	var allProjects []*gitlab.Project
	var collectProjects func(groupKey string) error

	collectProjects = func(groupKey string) error {
		// Fetch all projects for the current group
		projects, err := fetchPaginated(func(opts *gitlab.ListOptions) ([]*gitlab.Project, *gitlab.Response, error) {
			return client.Groups.ListGroupProjects(groupKey, &gitlab.ListGroupProjectsOptions{ListOptions: *opts})
		})
		if err != nil {
			g.logger.Error("failed to retrieve projects for group", "group", groupKey, "error", err)
			return err
		}
		allProjects = append(allProjects, projects...)

		subgroups, err := fetchPaginated(func(opts *gitlab.ListOptions) ([]*gitlab.Group, *gitlab.Response, error) {
			return client.Groups.ListSubGroups(groupKey, &gitlab.ListSubGroupsOptions{ListOptions: *opts})
		})
		if err != nil {
			g.logger.Error("failed to retrieve subgroups for group", "group", groupKey, "error", err)
			return err
		}

		// Recursively process each subgroup
		for _, subgroup := range subgroups {
			if subgroup == nil {
				continue
			}
			err := collectProjects(subgroup.FullPath)
			if err != nil {
				g.logger.Warn("failed to collect projects for subgroup, continuing", "subgroup", subgroup.FullPath, "error", err)
			}
		}

		return nil
	}

	// Start collecting projects and subgroups from the top-level group
	if err := collectProjects(groupKey); err != nil {
		return nil, fmt.Errorf("failed to retrieve projects for group and subgroups: %w", err)
	}

	var filteredProjects []*gitlab.Project
	for _, project := range allProjects {
		if project == nil {
			continue
		}
		if len(language) > 0 {
			languages, _, err := client.Projects.GetProjectLanguages(project.ID)
			if err != nil {
				g.logger.Warn("failed to retrieve languages for project", "projectWithNamespace", project.PathWithNamespace, "error", err)
				continue
			}
			if !isLanguagePresent(languages, language) {
				continue
			}
		}
		filteredProjects = append(filteredProjects, project)
	}

	return toRepositoryParams(filteredProjects), nil
}

// listProjectsForAllGroups fetches projects for all groups.
func (g *VCSGitlab) listProjectsForAllGroups(client *gitlab.Client, language string) ([]shared.RepositoryParams, error) {
	g.logger.Debug("listing projects for all groups", "language", language)
	allGroups, err := fetchPaginated(func(opts *gitlab.ListOptions) ([]*gitlab.Group, *gitlab.Response, error) {
		return client.Groups.ListGroups(&gitlab.ListGroupsOptions{
			ListOptions:  *opts,
			OrderBy:      gitlab.Ptr("id"),
			Sort:         gitlab.Ptr("asc"),
			AllAvailable: gitlab.Ptr(true),
		})
	})
	if err != nil {
		g.logger.Error("failed to list groups", "error", err)
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	var allRepositories []shared.RepositoryParams
	for _, group := range allGroups {
		if group == nil || len(group.FullPath) == 0 {
			g.logger.Warn("skipping group with missing name")
			continue
		}
		g.logger.Debug("fetching projects for group", "group", group.FullPath)
		repos, err := g.listProjectsForGroup(client, group.FullPath, language)
		if err != nil {
			g.logger.Error("failed to list projects for group", "group", group.FullPath, "error", err)
			continue
		}
		allRepositories = append(allRepositories, repos...)
	}

	if len(allRepositories) == 0 {
		return nil, fmt.Errorf("no projects found for groups or the current user")
	}
	return allRepositories, nil
}

// ListRepos handles listing repositories based on the provided VCSListReposRequest.
func (g *VCSGitlab) ListRepositories(args shared.VCSListRepositoriesRequest) ([]shared.RepositoryParams, error) {
	g.logger.Debug("starting projects listing", "args", args)

	if err := g.validateList(&args); err != nil {
		g.logger.Error("validation failed", "error", err)
		return nil, err
	}

	client, err := g.initializeGitlabClient(args.RepoParam.Domain)
	if err != nil {
		g.logger.Error("failed to initialize GitLab client", "error", err)
		return nil, fmt.Errorf("GitLab client initialization failed: %w", err)
	}

	if len(args.RepoParam.Namespace) > 0 {
		return g.listProjectsForGroup(client, args.RepoParam.Namespace, args.Language)
	}
	return g.listProjectsForAllGroups(client, args.Language)
}

// RetrievePRInformation handles retrieving PR information based on the provided VCSRetrievePRInformationRequest.
func (g *VCSGitlab) RetrievePRInformation(args shared.VCSRetrievePRInformationRequest) (shared.PRParams, error) {
	g.logger.Debug("starting to retrieve information about a MR", "args", args)

	if err := g.validateRetrievePRInformation(&args); err != nil {
		g.logger.Error("validation failed", "error", err)
		return shared.PRParams{}, fmt.Errorf("validation failed: %w", err)
	}

	client, err := g.initializeGitlabClient(args.RepoParam.Domain)
	if err != nil {
		g.logger.Error("failed to initialize GitLab client", "error", err)
		return shared.PRParams{}, fmt.Errorf("GitLab client initialization failed: %w", err)
	}

	mrID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	// TODO: need to handle the values safely
	projectID := fmt.Sprintf("%s/%s", args.RepoParam.Namespace, args.RepoParam.Repository)

	mrData, _, err := client.MergeRequests.GetMergeRequest(projectID, mrID, nil)
	if err != nil {
		g.logger.Error("failed to retrieve MR", "projectID", projectID, "MRID", mrID, "error", err)
		return shared.PRParams{}, fmt.Errorf("failed to retrieve MR: %w", err)
	}

	g.logger.Debug("successfully retrieved MR information", "projectID", projectID, "MRID", mrID)
	return convertToPRParams(mrData), nil
}

// AddRoleToPR handles adding a specified role to a MR based on the provided VCSAddRoleToPRRequest.
func (g *VCSGitlab) AddRoleToPR(args shared.VCSAddRoleToPRRequest) (bool, error) {
	g.logger.Debug("starting to add a user to a MR", "args", args)

	if err := g.validateAddRoleToPR(&args); err != nil {
		g.logger.Error("validation failed", "error", err)
		return false, fmt.Errorf("validation failed: %w", err)
	}

	client, err := g.initializeGitlabClient(args.RepoParam.Domain)
	if err != nil {
		g.logger.Error("failed to initialize GitLab client", "error", err)
		return false, fmt.Errorf("GitLab client initialization failed: %w", err)
	}

	mrID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	// TODO: need to handle the values safely
	projectID := fmt.Sprintf("%s/%s", args.RepoParam.Namespace, args.RepoParam.Repository)
	mrData, _, err := client.MergeRequests.GetMergeRequest(projectID, mrID, nil)
	if err != nil {
		g.logger.Error("failed to retrieve MR", "projectID", projectID, "MRID", mrID, "error", err)
		return false, fmt.Errorf("failed to retrieve MR: %w", err)
	}

	userData, _, err := client.Users.ListUsers(&gitlab.ListUsersOptions{
		Username: &args.Login,
	})
	if err != nil || len(userData) == 0 {
		g.logger.Error("failed to get user information or user not found", "login", args.Login, "error", err)
		return false, fmt.Errorf("failed to get user information or user not found: %w", err)
	}

	options := &gitlab.UpdateMergeRequestOptions{}
	switch strings.ToLower(args.Role) {
	case "assignee":
		options.AssigneeID = &userData[0].ID
	case "reviewer":
		options.ReviewerIDs = &[]int{userData[0].ID}
	default:
		err := fmt.Errorf("unsupported role: %s", args.Role)
		g.logger.Error("unsupported role for MR operation", "role", args.Role, "error", err)
		return false, err
	}

	if _, _, err := client.MergeRequests.UpdateMergeRequest(mrData.ProjectID, mrData.IID, options); err != nil {
		g.logger.Error("failed to add user to MR", "login", args.Login, "role", args.Role, "error", err)
		return false, fmt.Errorf("failed to add user to MR: %w", err)
	}

	g.logger.Info("user successfully added to the MR", "login", args.Login, "role", args.Role, "MRID", mrID, "projectID", projectID)
	return true, nil
}

// SetStatusOfPR handles setting the status of a PR based on the provided VCSSetStatusOfPRRequest.
func (g *VCSGitlab) SetStatusOfPR(args shared.VCSSetStatusOfPRRequest) (bool, error) {
	g.logger.Debug("starting to change the status of a MR", "args", args)

	if err := g.validateSetStatusOfPR(&args); err != nil {
		g.logger.Error("validation failed", "error", err)
		return false, fmt.Errorf("validation failed: %w", err)
	}

	client, err := g.initializeGitlabClient(args.RepoParam.Domain)
	if err != nil {
		g.logger.Error("failed to initialize GitLab client", "error", err)
		return false, fmt.Errorf("GitLab client initialization failed: %w", err)
	}

	mrID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	// TODO: need to handle the values safely
	projectID := fmt.Sprintf("%s/%s", args.RepoParam.Namespace, args.RepoParam.Repository)
	mrData, _, err := client.MergeRequests.GetMergeRequest(projectID, mrID, nil)
	if err != nil {
		g.logger.Error("failed to retrieve MR", "projectID", projectID, "MRID", mrID, "error", err)
		return false, fmt.Errorf("failed to retrieve MR: %w", err)
	}

	g.logger.Info("changing status of a particular MR", "MR", fmt.Sprintf("%v/%v/%v/%v", args.RepoParam.Domain, args.RepoParam.Namespace, args.RepoParam.Repository, mrID))

	switch strings.ToLower(args.Status) {
	case "approve":
		if _, _, err := client.MergeRequestApprovals.ApproveMergeRequest(mrData.ProjectID, mrData.IID, nil); err != nil {
			g.logger.Error("failed to approve the MR", "error", err, "MRID", mrID, "projectID", projectID)
			return false, fmt.Errorf("failed to approve the MR: %w", err)
		}
	case "unapprove":
		if _, err := client.MergeRequestApprovals.UnapproveMergeRequest(mrData.ProjectID, mrData.IID); err != nil {
			g.logger.Error("failed to unapprove the MR", "error", err, "MRID", mrID, "projectID", projectID)
			return false, fmt.Errorf("failed to unapprove the MR: %w", err)
		}
	default:
		err := fmt.Errorf("unsupported status: %s", args.Status)
		g.logger.Error("unsupported status for MR operation", "status", args.Status, "MRID", mrID, "projectID", projectID, "error", err)
		return false, err
	}

	g.logger.Info("MR successfully moved to status", "status", args.Status, "MRID", mrID, "projectID", projectID, "last_commit", mrData.SHA)
	return true, nil
}

// AddCommentToPR handles adding a comment to a specific merge request.
func (g *VCSGitlab) AddCommentToPR(args shared.VCSAddCommentToPRRequest) (bool, error) {
	g.logger.Debug("starting to add a comment to a MR", "args", args)

	if err := g.validateAddCommentToPR(&args); err != nil {
		g.logger.Error("validation failed", "error", err)
		return false, fmt.Errorf("validation failed: %w", err)
	}

	client, err := g.initializeGitlabClient(args.RepoParam.Domain)
	if err != nil {
		g.logger.Error("failed to initialize GitLab client", "error", err)
		return false, fmt.Errorf("GitLab client initialization failed: %w", err)
	}

	mrID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	// TODO: need to handle the values safely
	projectID := fmt.Sprintf("%s/%s", args.RepoParam.Namespace, args.RepoParam.Repository)
	mrData, _, err := client.MergeRequests.GetMergeRequest(projectID, mrID, nil)
	if err != nil {
		g.logger.Error("failed to retrieve MR", "projectID", projectID, "MRID", mrID, "error", err)
		return false, fmt.Errorf("failed to retrieve MR: %w", err)
	}

	commentText, err := g.buildCommentWithAttachments(client, mrData.ProjectID, args.Comment, args.FilePaths)
	if err != nil {
		g.logger.Warn("some files failed during upload, continuing with other files", "error", err)
	}

	g.logger.Info("commenting on a particular MR", "MR URL", fmt.Sprintf("%v/%v/%v/%v", args.RepoParam.Domain, args.RepoParam.Namespace, args.RepoParam.Repository, mrID))
	options := &gitlab.CreateMergeRequestNoteOptions{
		Body: &commentText,
	}

	if _, _, err = client.Notes.CreateMergeRequestNote(mrData.ProjectID, mrData.IID, options); err != nil {
		g.logger.Error("failed to add comment to MR", "error", err, "MRID", mrID, "projectID", projectID)
		return false, fmt.Errorf("failed to add comment to MR: %w", err)
	}

	g.logger.Info("successfully added a comment to MR", "MR", mrID)
	return true, nil
}

// buildCommentWithAttachments constructs the full comment text with file attachments.
func (g *VCSGitlab) buildCommentWithAttachments(client *gitlab.Client, projectID int, comment string, filePaths []string) (string, error) {
	var attachmentsText strings.Builder

	if len(filePaths) > 0 {
		attachmentsText.WriteString("\n\n**Report(s):**")
		for _, path := range filePaths {
			fileName, err := files.GetValidatedFileName(path)
			if err != nil {
				g.logger.Error("failed to validate file path, skipping file", "file", path, "error", err)
				continue
			}

			g.logger.Info("uploading file", "file", path)
			file, err := os.Open(path)
			if err != nil {
				g.logger.Error("failed to open file, skipping file", "file", path, "error", err)
				continue
			}

			uploadedFile, _, err := client.Projects.UploadFile(projectID, file, fileName, nil)
			file.Close()
			if err != nil {
				g.logger.Error("failed to upload file, skipping file", "file", path, "error", err)
				continue
			}

			attachmentMarkdown := fmt.Sprintf("* %s", uploadedFile.Markdown)
			attachmentsText.WriteString("\n" + attachmentMarkdown)
		}
	}

	fullComment := strings.Builder{}
	fullComment.WriteString(comment)
	fullComment.WriteString(attachmentsText.String())
	return fullComment.String(), nil
}

// fetchPR handles fetching pull request changes.
func (g *VCSGitlab) fetchPR(args *shared.VCSFetchRequest) (string, error) {
	g.logger.Info("handling PR changes fetching", "MRID", args.RepoParam.PullRequestID)

	client, err := g.initializeGitlabClient(args.RepoParam.Domain)
	if err != nil {
		g.logger.Error("failed to initialize GitLab client", "error", err)
		return "", fmt.Errorf("GitLab client initialization failed: %w", err)
	}

	mrID, _ := strconv.Atoi(args.RepoParam.PullRequestID)
	// TODO: need to handle the values safely
	projectID := fmt.Sprintf("%s/%s", args.RepoParam.Namespace, args.RepoParam.Repository)
	mrData, _, err := client.MergeRequests.GetMergeRequest(projectID, mrID, nil)
	if err != nil {
		g.logger.Error("failed to retrieve MR", "projectID", projectID, "MRID", mrID, "error", err)
		return "", fmt.Errorf("failed to retrieve MR: %w", err)
	}
	g.logger.Debug("MR Data", mrData)

	args.Branch = mrData.SourceBranch
	if mrData.SourceProjectID != mrData.TargetProjectID {
		args.Branch = mrData.SHA
		g.logger.Warn("found merging from a fork", "fromUser", mrData.Author.Username)
		g.logger.Warn("pr will be fetched as a detached latest commit",
			"latest_commit", mrData.SHA,
		)
	} else if args.FetchMode == ftutils.PRCommitMode {
		args.Branch = mrData.SHA
		g.logger.Info("fetching pull request by commit",
			"latest_commit", mrData.SHA,
		)
	}

	diffs, err := fetchPaginated(func(opts *gitlab.ListOptions) ([]*gitlab.MergeRequestDiff, *gitlab.Response, error) {
		return client.MergeRequests.ListMergeRequestDiffs(mrData.ProjectID, mrData.IID, &gitlab.ListMergeRequestDiffsOptions{ListOptions: *opts})
	})
	if err != nil {
		g.logger.Error("failed to get MR changes", "MRID", mrData.IID, "error", err)
		return "", fmt.Errorf("failed to get MR changes: %w", err)
	}

	projectData, _, err := client.Projects.GetProject(mrData.ProjectID, nil)
	if err != nil {
		g.logger.Error("failed to retrieve project details", "projectID", mrData.ProjectID, "error", err)
		return "", fmt.Errorf("failed to retrieve project details: %w", err)
	}
	args.CloneURL = projectData.SSHURLToRepo

	g.logger.Debug("starting to fetch MR code")

	pluginConfigMap, err := shared.StructToMap(g.globalConfig.GitlabPlugin)
	if err != nil {
		g.logger.Error("error converting struct to map", "error", err)
		return "", fmt.Errorf("error converting struct to map: %w", err)
	}

	clientGit, err := git.New(g.logger, g.globalConfig, pluginConfigMap, args, PluginName)
	if err != nil {
		g.logger.Error("failed to initialize Git client", "error", err)
		return "", fmt.Errorf("failed to initialize Git client: %w", err)
	}

	_, err = clientGit.CloneRepository(args)
	if err != nil {
		g.logger.Error("failed to clone repository", "error", err)
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	baseDestPath := config.GetPRTempPath(g.globalConfig, args.RepoParam.Domain, args.RepoParam.Namespace, args.RepoParam.Repository, mrID)

	g.logger.Debug("copying files that have changed")
	for _, val := range diffs {
		if val == nil {
			continue
		}
		if val.DeletedFile {
			g.logger.Debug("skipping file due to type", "type", "deletedFile", "path", val.NewPath)
			continue
		}

		srcPath := filepath.Join(args.TargetFolder, val.NewPath)
		destPath := filepath.Join(baseDestPath, val.NewPath)
		if err := files.Copy(srcPath, destPath); err != nil {
			g.logger.Error("error copying file", "error", err)
		}
	}

	if err := files.CopyDotFiles(args.TargetFolder, baseDestPath, g.logger); err != nil {
		g.logger.Error("failed to copy dot files", "error", err)
		return "", fmt.Errorf("failed to copy dot files: %w", err)
	}

	g.logger.Info("files for MR scan are copied", "folder", baseDestPath)
	return baseDestPath, nil
}

// Fetch retrieves code based on the provided VCSFetchRequest.
func (g *VCSGitlab) Fetch(args shared.VCSFetchRequest) (shared.VCSFetchResponse, error) {
	var result shared.VCSFetchResponse

	if err := g.validateFetch(&args); err != nil {
		g.logger.Error("validation failed", "error", err)
		return shared.VCSFetchResponse{}, fmt.Errorf("validation failed: %w", err)
	}

	switch args.FetchMode {
	case ftutils.PRBranchMode, ftutils.PRRefMode, ftutils.PRCommitMode:
		path, err := g.fetchPR(&args)
		if err != nil {
			g.logger.Error("failed to fetch files from pull request", "error", err)
			return result, fmt.Errorf("failed to fetch files from pull request: %w", err)
		}
		result.Path = path
	default:
		pluginConfigMap, err := shared.StructToMap(g.globalConfig.BitbucketPlugin)
		if err != nil {
			g.logger.Error("error converting struct to map", "error", err)
			return result, fmt.Errorf("error converting struct to map: %w", err)
		}

		clientGit, err := git.New(g.logger, g.globalConfig, pluginConfigMap, &args, PluginName)
		if err != nil {
			g.logger.Error("failed to initialize Git client", "error", err)
			return result, fmt.Errorf("failed to initialize Git client: %w", err)
		}

		path, err := clientGit.CloneRepository(&args)
		if err != nil {
			g.logger.Error("failed to clone repository", "error", err)
			return result, fmt.Errorf("failed to clone repository: %w", err)
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
