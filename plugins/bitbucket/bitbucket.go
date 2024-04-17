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

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	//bitbucketv2 "github.com/ktrysmt/go-bitbucket"

	"github.com/mitchellh/mapstructure"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

type VCSBitbucket struct {
	logger       hclog.Logger
	globalConfig *config.Config
}

// Limit for Bitbucket v1 API page response
var opts = map[string]interface{}{
	"limit": maxLimitElements,
	"start": startElement,
}

func getProjectsResponse(r *bitbucketv1.APIResponse) ([]bitbucketv1.Project, error) {
	var m []bitbucketv1.Project
	err := mapstructure.Decode(r.Values["values"], &m)
	return m, err
}

// Listing all project in Bitbucket v1 API
func (g *VCSBitbucket) listAllProjects(client *bitbucketv1.APIClient) ([]shared.ProjectParams, error) {
	g.logger.Info("Starting to list all projects..")
	response, err := client.DefaultApi.GetProjects(opts)
	if err != nil {
		g.logger.Error("A listing projects function is failed", "error", err)
		return nil, err
	}

	g.logger.Info("Projects is listed")
	res, err := getProjectsResponse(response)
	if err != nil {
		g.logger.Error("A response parsing function is failed", "error", err)
		return nil, err
	}

	var projectsList []shared.ProjectParams
	for _, bitbucketRepo := range res {
		projectsList = append(projectsList, shared.ProjectParams{Key: bitbucketRepo.Key, Name: bitbucketRepo.Name, Link: bitbucketRepo.Links.Self[0].Href})
	}

	g.logger.Debug("The list of projects is ready")

	resultJson, _ := json.MarshalIndent(projectsList, "", "    ")
	g.logger.Debug(string(resultJson))

	return projectsList, nil
}

// Resolving information about all repositories in one project from Bitbucket v1 API
func (g *VCSBitbucket) resolveOneProject(client *bitbucketv1.APIClient, project string) ([]shared.RepositoryParams, error) {
	g.logger.Info("Resolving a particular project", "project", project)
	response, err := client.DefaultApi.GetRepositoriesWithOptions(project, opts)
	if err != nil {
		g.logger.Error("Resolving the project is failed", "project", project, "error", err)
		return nil, err
	}

	g.logger.Debug("Project is resolved", "project", project)
	result, err := bitbucketv1.GetRepositoriesResponse(response)
	if err != nil {
		g.logger.Error("Response parsing is failed", "project", project, "error", err)
		return nil, err
	}

	var resultList []shared.RepositoryParams
	for _, repo := range result {
		var httpLink string
		var sshLink string

		for _, clone_links := range repo.Links.Clone {
			if clone_links.Name == "http" {
				httpLink = clone_links.Href
			} else if clone_links.Name == "ssh" {
				sshLink = clone_links.Href
			} else {
				continue
			}
		}

		resultList = append(resultList, shared.RepositoryParams{Namespace: project, RepoName: repo.Name, HttpLink: httpLink, SshLink: sshLink})
	}

	g.logger.Debug("The list of repositories is ready")

	resultJson, _ := json.MarshalIndent(resultList, "", "    ")
	g.logger.Debug(string(resultJson))

	return resultList, nil
}

func (g *VCSBitbucket) ListRepos(args shared.VCSListReposRequest) ([]shared.RepositoryParams, error) {
	g.logger.Debug("Starting execution of an all-repositories listing function", "args", args)
	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("An init stage of an all repositories listing function is failed", "error", err)
		return nil, err
	}

	client, cancel := BBClient(args.VCSURL, variables)
	defer cancel()

	var repositories []shared.RepositoryParams
	if len(args.Namespace) != 0 {
		oneProjectData, err := g.resolveOneProject(client, args.Namespace)
		if err != nil {
			g.logger.Error("The particular repository function is failed", "error", err)
			return nil, err
		}
		for _, repo := range oneProjectData {
			repositories = append(repositories, repo)
		}

	} else {
		projectsList, err := g.listAllProjects(client)
		if err != nil {
			g.logger.Error("The all-repositories listing function is failed", "error", err)
			return nil, err
		}

		for _, projectName := range projectsList {
			oneProjectData, err := g.resolveOneProject(client, projectName.Key)
			if err != nil {
				g.logger.Error("The listing of all repositories function is failed", "error", err)
				return nil, err
			}
			for _, repo := range oneProjectData {
				repositories = append(repositories, repo)
			}
		}

	}

	return repositories, nil
}

func (g *VCSBitbucket) RetrivePRInformation(args shared.VCSRetrivePRInformationRequest) (shared.PRParams, error) {
	g.logger.Debug("Starting retrive information about a PR", "args", args)

	var pr bitbucketv1.PullRequest
	var result shared.PRParams
	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("An init stage of retriving information about a PR is failed", "error", err)
		return result, err
	}

	client, cancel := BBClient(args.VCSURL, variables)
	defer cancel()

	g.logger.Info("Retriving information a particular PR", "PR", fmt.Sprintf("%v/%v/%v/%v", args.VCSURL, args.Namespace, args.Repository, args.PullRequestId))
	rawResponse, err := client.DefaultApi.GetPullRequest(args.Namespace, args.Repository, args.PullRequestId)
	if err != nil {
		g.logger.Error("Getting information about PR is failed", "error", err)
		return result, err
	}

	pr, err = bitbucketv1.GetPullRequestResponse(rawResponse)
	if err != nil {
		g.logger.Error("Parsing information about PR is failed", "error", err)
		return result, err
	}

	result = shared.PRParams{
		PullRequestId: pr.ID,
		Title:         pr.Title,
		Description:   pr.Description,
		State:         pr.State,
		AuthorEmail:   pr.Author.User.EmailAddress,
		AuthorName:    pr.Author.User.DisplayName,
		SelfLink:      pr.Links.Self[0].Href,
		CreatedDate:   pr.CreatedDate,
		UpdatedDate:   pr.UpdatedDate,
		FromRef: shared.RefPRInf{
			ID:           pr.FromRef.ID,
			DisplayId:    pr.FromRef.DisplayID,
			LatestCommit: pr.FromRef.LatestCommit,
		},
		ToRef: shared.RefPRInf{
			ID:           pr.ToRef.ID,
			DisplayId:    pr.FromRef.DisplayID,
			LatestCommit: pr.ToRef.LatestCommit,
		},
	}
	g.logger.Info("Information about particular PR is retrived", "PR", result.SelfLink)

	return result, nil
}

func (g *VCSBitbucket) AddRoleToPR(args shared.VCSAddRoleToPRRequest) (interface{}, error) {
	g.logger.Debug("Starting add a reviewer PR", "args", args)

	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("An init stage of adding a reviewer to a PR is failed", "error", err)
		return nil, err
	}

	client, err := utils.NewHTTPClient(false, "")
	if err != nil {
		g.logger.Error("Creating HTTP client finished unsuccessfuly", "error", err)
		return nil, err
	}

	urlReq := fmt.Sprintf("https://%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/participants/", args.VCSURL, args.Namespace, args.Repository, args.PullRequestId)
	authValue := variables.Username + ":" + variables.Token
	authHeader := base64.StdEncoding.EncodeToString([]byte(authValue))
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Basic " + authHeader},
	}
	postData := []byte(fmt.Sprintf(`{"user": {"name": "%s"}, "role": "%s", "approved": false}`, args.Login, args.Role))
	g.logger.Info("Sending a request to add a user to a PR", "user", args.Login, "role", args.Role, "PR_url", urlReq)

	response, responseBody, err := client.DoRequest("POST", urlReq, headers, postData)
	if err != nil {
		g.logger.Error("An send a request stage of adding a user to a PR is failed", "error", err)
		return nil, err
	}

	if response.StatusCode == http.StatusConflict {
		text := fmt.Sprintf("The request's returned with a 409 response code. User %s is an author of the PR.", args.Login)
		g.logger.Error(text)
		g.logger.Debug("Debug details", "Body response", string(responseBody))
		return nil, fmt.Errorf(text)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		text := fmt.Sprintf("The request's returned with not a 2xx response code. Response body: %s", string(responseBody))
		g.logger.Error(text)
		g.logger.Debug("Debug details", "Body response", string(responseBody))
		return nil, fmt.Errorf(text)
	}
	g.logger.Info("The user is successfuly added to the PR", "user", args.Login, "role", args.Role, "PR_url", urlReq)
	return nil, nil
}

func (g *VCSBitbucket) SetStatusOfPR(args shared.VCSSetStatusOfPRRequest) (bool, error) {
	g.logger.Debug("Starting changing a status of PR", "args", args)
	var approval bool

	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("An init stage of changing a status to a PR is failed", "error", err)
		return false, err
	}

	client, cancel := BBClient(args.VCSURL, variables)
	defer cancel()

	g.logger.Info("Changin status of a particular PR", "PR", fmt.Sprintf("%v/%v/%v/%v", args.VCSURL, args.Namespace, args.Repository, args.PullRequestId))

	if args.Status == "APPROVED" {
		approval = true
	}

	userBB := bitbucketv1.UserWithMetadata{User: bitbucketv1.UserWithLinks{
		Name: args.Login,
		Slug: args.Login,
	},
		Approved: approval,
		Status:   args.Status,
	}

	rawResponse, err := client.DefaultApi.UpdateStatus(args.Namespace, args.Repository, int64(args.PullRequestId), args.Login, userBB)
	if err != nil {
		g.logger.Error("Getting information about PR is failed", "error", err)
		return false, err
	}

	participant, err := bitbucketv1.GetUserWithMetadataResponse(rawResponse)
	if err != nil {
		g.logger.Error("Parsing information about PR is failed", "error", err)
		return false, err
	}
	g.logger.Info("PR sucessfully moved to status", "status", args.Status, "PR_id", args.PullRequestId, "last_commit", participant.LastReviewedCommit)

	return true, nil
}

func (g *VCSBitbucket) AddComment(args shared.VCSAddCommentToPRRequest) (bool, error) {
	g.logger.Debug("Starting changing a status of PR", "args", args)

	variables, err := g.init("list", "")
	if err != nil {
		g.logger.Error("An init stage of adding a comment to a PR is failed", "error", err)
		return false, err
	}

	client, cancel := BBClient(args.VCSURL, variables)
	defer cancel()

	g.logger.Info("Commenting a particular PR", "PR", fmt.Sprintf("%v/%v/%v/%v", args.VCSURL, args.Namespace, args.Repository, args.PullRequestId))

	comment := bitbucketv1.Comment{
		Text: args.Comment,
	}

	_, err = client.DefaultApi.CreatePullRequestComment(args.Namespace, args.Repository, args.PullRequestId, comment, []string{"application/json"})
	if err != nil {
		g.logger.Error("Commenting PR is failed", "error", err)
		return false, err
	}

	g.logger.Info("Comment is done")
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

	var pluginMap = map[string]plugin.Plugin{
		shared.PluginTypeVCS: &shared.VCSPlugin{Impl: VCS},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
