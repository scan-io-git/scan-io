package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/bitbucket"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

type VCSBitbucket struct {
	logger       hclog.Logger
	globalConfig *config.Config
}

// toRepositoryParams converts a slice of internal Repository type to a slice of external RepositoryParams type.
func toRepositoryParams(repos *[]bitbucket.Repository) []shared.RepositoryParams {
	var repoParams []shared.RepositoryParams
	for _, repo := range *repos {
		httpLink, sshLink := bitbucket.ExtractCloneLinks(repo.Links.Clone)
		repoParams = append(repoParams, shared.RepositoryParams{
			Namespace: repo.Project.Name,
			RepoName:  repo.Name,
			HttpLink:  httpLink,
			SshLink:   sshLink,
		})
	}
	return repoParams
}

// convertToPRParams converts the internal PullRequest type to the external PRParams type.
func convertToPRParams(pr *bitbucket.PullRequest) shared.PRParams {
	return shared.PRParams{
		Id:          pr.ID,
		Title:       pr.Title,
		Description: pr.Description,
		State:       pr.State,
		Author:      shared.User{DisplayName: pr.Author.User.DisplayName, Email: pr.Author.User.EmailAddress},
		SelfLink:    pr.Links.Self[0].Href,
		Source: shared.Reference{
			ID:           pr.FromReference.ID,
			DisplayId:    pr.FromReference.DisplayID,
			LatestCommit: pr.FromReference.LatestCommit,
		},
		Destination: shared.Reference{
			ID:           pr.ToReference.ID,
			DisplayId:    pr.ToReference.DisplayID,
			LatestCommit: pr.ToReference.LatestCommit,
		},
		CreatedDate: pr.CreatedDate,
		UpdatedDate: pr.UpdatedDate,
	}
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

	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("Initialization failed during the listing function", "error", err)
		return nil, err
	}

	authInfo := bitbucket.AuthInfo{
		Username: variables.Username,
		Token:    variables.Token,
	}

	client, err := bitbucket.New(g.logger, args.VCSURL, authInfo, g.globalConfig)
	if err != nil {
		g.logger.Error("initialization bitbucket client failed", "error", err)
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

	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("An init stage of an all repositories listing function is failed", "error", err)
		return shared.PRParams{}, err
	}

	authInfo := bitbucket.AuthInfo{
		Username: variables.Username,
		Token:    variables.Token,
	}
	client, err := bitbucket.New(g.logger, args.VCSURL, authInfo, g.globalConfig)
	if err != nil {
		g.logger.Error("initialization bitbucket client failed", "error", err)
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

	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("An init stage of an all repositories listing function is failed", "error", err)
		return false, err
	}

	authInfo := bitbucket.AuthInfo{
		Username: variables.Username,
		Token:    variables.Token,
	}

	client, err := bitbucket.New(g.logger, args.VCSURL, authInfo, g.globalConfig)
	if err != nil {
		g.logger.Error("initialization bitbucket client failed", "error", err)
		return false, err
	}

	prData, err := client.PullRequests.Get(args.Namespace, args.Repository, args.PullRequestId)
	if err != nil {
		g.logger.Error("Failed to retrieve information about the PR", "PRID", args.PullRequestId, "error", err)
		return false, err
	}

	if _, err := prData.AddRole(args.Login, args.Role); err != nil {
		g.logger.Error("Failed to add role to PR", "error", err)
		return false, err
	}

	g.logger.Info("User successfully added to the PR", "user", args.Login, "role", args.Role)
	return true, nil
}

// SetStatusOfPR handles setting a status of PR based on the provided VCSSetStatusOfPRRequest.
func (g *VCSBitbucket) SetStatusOfPR(args shared.VCSSetStatusOfPRRequest) (bool, error) {
	g.logger.Debug("Starting changing a status of PR", "args", args)

	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("Failed to initialize for changing the PR status", "error", err)
		return false, err
	}

	authInfo := bitbucket.AuthInfo{
		Username: variables.Username,
		Token:    variables.Token,
	}

	client, err := bitbucket.New(g.logger, args.VCSURL, authInfo, g.globalConfig)
	if err != nil {
		g.logger.Error("initialization bitbucket client failed", "error", err)
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
	g.logger.Debug("Starting to add a comment to a PR", "args", args)

	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("Initialization failed for adding a comment to a PR", "error", err)
		return false, err
	}

	authInfo := bitbucket.AuthInfo{
		Username: variables.Username,
		Token:    variables.Token,
	}

	client, err := bitbucket.New(g.logger, args.VCSURL, authInfo, g.globalConfig)
	if err != nil {
		g.logger.Error("initialization bitbucket client failed", "error", err)
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

func (g *VCSBitbucket) fetchPRChanges(args *shared.VCSFetchRequest, variables *shared.EvnVariables) ([]*Change, error) {
	g.logger.Info("Fetching PR changes")
	var prData interface{}
	changes := []*Change{}

	client, err := utils.NewHTTPClient(false, "")
	if err != nil {
		g.logger.Error("Creating HTTP client finished unsuccessfuly", "error", err)
		return nil, err
	}

	start := "0"
	baseUrl := fmt.Sprintf("https://%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%s", args.RepoParam.VCSURL, args.RepoParam.Namespace, args.RepoParam.RepoName, args.RepoParam.PRID)
	urlReqDiff := fmt.Sprintf("%s/changes", baseUrl)
	authValue := variables.Username + ":" + variables.Token
	authHeader := base64.StdEncoding.EncodeToString([]byte(authValue))
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Basic " + authHeader},
	}
	response, responseBody, err := client.DoRequest("GET", baseUrl, headers, nil)
	if err != nil {
		g.logger.Error("A PR changes function is failed by doing a request", "error", err)
		return nil, err
	}

	if response.StatusCode >= http.StatusBadRequest {
		g.logger.Error("HTTP request failed with status", "code", response.StatusCode, "error", err)
		return nil, fmt.Errorf("Response body %v", string(responseBody))
	}

	if err = json.Unmarshal(responseBody, &prData); err != nil {
		g.logger.Error("A PR changes function is failed by parsing json", "error", err)
		return nil, err
	}

	pr := prData.(map[string]interface{})
	fromRef, _ := pr["fromRef"].(map[string]interface{})
	branch, _ := fromRef["id"].(string)
	args.Branch = branch

	g.logger.Debug("Starting extracting PR changes")
	for {
		changesReponse := new(Changes)

		params := url.Values{
			"withComments": []string{"false"},
			"start":        []string{start},
			"limit":        []string{"100"},
		}
		fullURL := fmt.Sprintf("%s?%s", urlReqDiff, params.Encode())
		response, responseBody, err := client.DoRequest("GET", fullURL, headers, nil)
		if err != nil {
			g.logger.Error("A PR changes function is failed by doing a request", "error", err)
			return nil, err
		}

		if response.StatusCode >= http.StatusBadRequest {
			g.logger.Error("HTTP request failed with status", "code", response.StatusCode, "responseBody", responseBody)
			return nil, fmt.Errorf("Response body %v", string(responseBody))
		}
		if err = json.Unmarshal(responseBody, &changesReponse); err != nil {
			g.logger.Error("A PR changes function is failed by parsing json", "error", err)
			return nil, err
		}

		changes = append(changes, changesReponse.Values...)
		if changesReponse.IsLastPage || changesReponse.NextPageStart == nil {
			break
		}
		start = strconv.Itoa(*changesReponse.NextPageStart)
	}

	return changes, nil
}

func (g *VCSBitbucket) fetchPR(args *shared.VCSFetchRequest, variables *shared.EvnVariables) (string, error) {
	g.logger.Info("Handling PR changes fetching")

	_, err := g.init("list", "")
	if err != nil {
		g.logger.Error("An init stage of retriving information about a PR is failed", "error", err)
		return "", err
	}

	changes, err := g.fetchPRChanges(args, variables)
	if err != nil {
		return "", err
	}

	g.logger.Info("Strating fetching PR code")
	//TODO there is a strange bug when it fetchs only pr changes without all other files in case of PR fetch ???
	_, err = shared.GitClone(*args, *variables, g.logger)
	if err != nil && err.Error() != "already up-to-date" {
		g.logger.Error("The fetching PR function is failed", "error", err)
		return "", err
	}

	// Create a folder for scanning changed files from the PR
	// TODO move to core
	rawStartTime := time.Now().UTC()
	startTime := rawStartTime.Format(time.RFC3339)
	targetPRFolder := filepath.Join(shared.GetScanioHome(), "tmp", strings.ToLower(args.RepoParam.VCSURL), strings.ToLower(args.RepoParam.Namespace),
		strings.ToLower(args.RepoParam.RepoName), "scanio-pr-tmp", args.RepoParam.PRID, startTime)
	if err := os.MkdirAll(targetPRFolder, os.ModePerm); err != nil {
		g.logger.Error("Creating a folde for PR function is failed", "error", err)
		return "", err
	}

	g.logger.Debug("Copy files which are changed")
	for _, val := range changes {
		if !shared.ContainsSubstring(val.Type, changeTypes) {
			g.logger.Debug("Skipping", "type", val.Type, "path", val.Path.ToString)
			continue
		}

		srcPath := filepath.Join(args.TargetFolder, val.Path.ToString)
		destPath := filepath.Join(targetPRFolder, val.Path.ToString)
		err := shared.Copy(srcPath, destPath)
		if err != nil {
			g.logger.Error("Error copying file", "error", err)
		}
	}

	g.logger.Debug("Copy usefull files which are started with dot")
	files, err := os.ReadDir(args.TargetFolder)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if file.IsDir() && file.Name()[0] == '.' {
			sourceFolder := filepath.Join(args.TargetFolder, file.Name())
			targetFolderNext := filepath.Join(targetPRFolder, file.Name())
			if err := shared.Copy(sourceFolder, targetFolderNext); err != nil {
				fmt.Printf("Error copying directory: %v\n", err)
			}
		} else if file.Name()[0] == '.' {
			sourceFolder := filepath.Join(args.TargetFolder, file.Name())
			err := shared.Copy(sourceFolder, targetPRFolder)
			if err != nil {
				fmt.Printf("Error copying file: %v\n", err)
			}
		}
	}

	g.logger.Info("Files for PR scan are copied", "folder", targetPRFolder)
	return targetPRFolder, nil
}

func (g *VCSBitbucket) Fetch(args shared.VCSFetchRequest) (shared.VCSFetchResponse, error) {
	var (
		result shared.VCSFetchResponse
		path   string
	)

	variables, err := g.init("fetch", args.AuthType)
	if err != nil {
		g.logger.Error("An init function for a fetching function is failed", "error", err)
		return result, err
	}
	if args.Mode == "PRscan" {
		path, err = g.fetchPR(&args, &variables)
		if err != nil {
			return result, err
		}
		result.Path = path
	} else {
		path, err = shared.GitClone(args, variables, g.logger)
		if err != nil {
			g.logger.Error("The fetching function is failed", "error", err)
			return result, err
		}
		result.Path = path
	}

	return result, nil
}

func (g *VCSBitbucket) Setup(configData []byte) (bool, error) {
	var cfg config.Config
	if err := json.Unmarshal(configData, &cfg); err != nil {
		return false, err
	}
	g.globalConfig = &cfg
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
	})
}
