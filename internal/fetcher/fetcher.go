package fetcher

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/files"

	utils "github.com/scan-io-git/scan-io/internal/utils"
)

const (
	BasicMode  = "basic"
	PRScanMode = "fetchPR"
)

// Fetcher represents the configuration and behavior of a fetcher.
type Fetcher struct {
	pluginName     string       // Name of the VCS plugin to use
	authType       string       // Authentication type (e.g., "http", "ssh")
	sshKey         string       // Path to the SSH key
	branch         string       // Branch to fetch
	rmListExts     []string     // List of extensions to remove after fetching
	concurrentJobs int          // Number of concurrent jobs to run
	logger         hclog.Logger // Logger for logging messages and errors
}

// New creates a new Fetcher instance with the provided configuration.
func New(pluginName, authType, sshKey, branch string, rmListExts []string, jobs int, logger hclog.Logger) *Fetcher {
	return &Fetcher{
		pluginName:     pluginName,
		authType:       authType,
		sshKey:         sshKey,
		branch:         branch,
		rmListExts:     rmListExts,
		concurrentJobs: jobs,
		logger:         logger,
	}
}

// PrepFetchReqList prepares fetch arguments for the repositories.
func (f *Fetcher) PrepFetchReqList(cfg *config.Config, repos []shared.RepositoryParams) ([]shared.VCSFetchRequest, error) {
	var fetchReqList []shared.VCSFetchRequest

	for _, repo := range repos {
		cloneURL := f.getCloneURL(repo)
		domain, err := utils.GetDomain(cloneURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get domain for URL %s: %w", cloneURL, err)
		}

		repo.Domain = domain
		fetchMode := getFetchMode(repo)
		if f.pluginName == "bitbucket" && strings.HasPrefix(repo.Namespace, "~") {
			repo.Namespace = strings.TrimPrefix(repo.Namespace, "~") // in the case of user repos we should put results into the same folder for ssh and http links
		}
		targetFolder := config.GetRepositoryPath(cfg, domain, filepath.Join(repo.Namespace, repo.Repository))

		fetchReqList = append(fetchReqList, f.createFetchRequest(repo, cloneURL, targetFolder, fetchMode))
	}
	return fetchReqList, nil
}

// getCloneURL returns the appropriate clone URL based on the auth type.
func (f *Fetcher) getCloneURL(repo shared.RepositoryParams) string {
	if f.authType == "http" {
		return repo.HTTPLink
	}
	return repo.SSHLink
}

// getFetchMode determines the mode for the fetch request.
func getFetchMode(repo shared.RepositoryParams) string {
	if repo.PullRequestId != "" {
		return PRScanMode
	}
	return BasicMode
}

// createFetchRequest creates a VCSFetchRequest with the specified parameters.
func (f *Fetcher) createFetchRequest(repo shared.RepositoryParams, cloneURL, targetFolder, fetchMode string) shared.VCSFetchRequest {
	return shared.VCSFetchRequest{
		CloneURL:     cloneURL,
		Branch:       f.branch,
		AuthType:     f.authType,
		SSHKey:       f.sshKey,
		TargetFolder: targetFolder,
		Mode:         fetchMode,
		RepoParam:    repo,
	}
}

// fetchRepo fetches a single repository using the configured VCS plugin.
func (f *Fetcher) fetchRepo(cfg *config.Config, fetchArgs shared.VCSFetchRequest) (shared.VCSFetchResponse, error) {
	var result shared.VCSFetchResponse

	err := shared.WithPlugin(cfg, "plugin-vcs", shared.PluginTypeVCS, f.pluginName, func(raw interface{}) error {
		vcsPlugin, ok := raw.(shared.VCS)
		if !ok {
			return fmt.Errorf("invalid plugin type")
		}
		var err error
		result, err = vcsPlugin.Fetch(fetchArgs)
		if err != nil {
			f.logger.Error("VCS plugin fetch failed", "fetchArgs", fetchArgs, "error", err)
			return fmt.Errorf("VCS plugin fetch failed. Fetch arguments: %v. Error: %w", fetchArgs, err)
		}

		files.FindByExtAndRemove(fetchArgs.TargetFolder, f.rmListExts)
		return nil
	})

	return result, err
}

// FetchRepos fetches repositories concurrently.
func (f *Fetcher) FetchRepos(cfg *config.Config, fetchReqList []shared.VCSFetchRequest) (shared.GenericLaunchesResult, error) {
	f.logger.Info("fetch starting", "total", len(fetchReqList), "goroutines", f.concurrentJobs)

	var results shared.GenericLaunchesResult
	resultsChannel := make(chan shared.GenericResult, len(fetchReqList))
	errorChannel := make(chan error, len(fetchReqList))
	values := make([]interface{}, len(fetchReqList))
	for i := range fetchReqList {
		values[i] = fetchReqList[i]
	}

	shared.ForEveryStringWithBoundedGoroutines(f.concurrentJobs, values, func(i int, value interface{}) {
		fetchArgs, ok := value.(shared.VCSFetchRequest)
		if !ok {
			err := fmt.Errorf("invalid fetch argument type at index %d", i)
			f.logger.Error(err.Error())
			errorChannel <- err
			return
		}
		f.logger.Info("goroutine started", "index", i+1, "args", fetchArgs)

		var message string
		result, err := f.fetchRepo(cfg, fetchArgs)
		if err != nil {
			message = err.Error()
		}

		if err != nil && err.Error() != "already up-to-date" {
			resultsChannel <- shared.GenericResult{Args: fetchArgs, Result: result, Status: "FAILED", Message: err.Error()}
			errorChannel <- err
		} else {
			resultsChannel <- shared.GenericResult{Args: fetchArgs, Result: result, Status: "OK", Message: message}
		}
	})

	close(resultsChannel)
	close(errorChannel)

	for result := range resultsChannel {
		results.Launches = append(results.Launches, result)
	}
	var errs []error
	for err := range errorChannel {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		f.logger.Debug("fetch execution errors", "errors", errs)
		return results, fmt.Errorf("one or more fetch attempts failed. Check the results file")
	}

	return results, nil
}
