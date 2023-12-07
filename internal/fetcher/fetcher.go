package fetcher

import (
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"

	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
)

type Fetcher struct {
	authType      string
	sshKey        string
	jobs          int
	branch        string
	vcsPluginName string
	rmExts        []string
	logger        hclog.Logger
}

func New(authType string, sshKey string, jobs int, branch string, vcsPluginName string, rmExts []string, logger hclog.Logger) Fetcher {
	return Fetcher{
		authType:      authType,
		sshKey:        sshKey,
		jobs:          jobs,
		branch:        branch,
		vcsPluginName: vcsPluginName,
		rmExts:        rmExts,
		logger:        logger,
	}
}

func (f Fetcher) PrepFetchArgs(repos []shared.RepositoryParams) ([]shared.VCSFetchRequest, error) {
	var (
		fetchArgs []shared.VCSFetchRequest
	)
	mode := "basic"

	for _, repo := range repos {

		cloneURL := repo.SshLink
		if f.authType == "http" {
			cloneURL = repo.HttpLink
		}

		domain, err := utils.GetDomain(cloneURL)
		if err != nil {
			return nil, err
		}
		repo.VCSURL = domain

		if repo.PRID != "" {
			mode = "PRscan"
		}
		targetFolder := shared.GetRepoPath(strings.ToLower(domain), filepath.Join(strings.ToLower(repo.Namespace), strings.ToLower(repo.RepoName)))

		fetchArgs = append(fetchArgs, shared.VCSFetchRequest{
			CloneURL:     cloneURL,
			Branch:       f.branch,
			AuthType:     f.authType,
			SSHKey:       f.sshKey,
			TargetFolder: targetFolder,
			Mode:         mode,
			RepoParam:    repo,
		})

	}
	return fetchArgs, nil
}

func (f Fetcher) fetchRepo(fetchArgs shared.VCSFetchRequest) error {
	err := shared.WithPlugin("plugin-vcs", shared.PluginTypeVCS, f.vcsPluginName, func(raw interface{}) error {
		vcsName := raw.(shared.VCS)
		_, err := vcsName.Fetch(fetchArgs)
		if err != nil {
			f.logger.Error("vcs plugin failed on fetch", "err", err)
			return err
		}

		utils.FindByExtAndRemove(fetchArgs.TargetFolder, f.rmExts)
		return nil
	})

	return err
}

func (f Fetcher) FetchRepos(fetchArgs []shared.VCSFetchRequest) shared.GenericLaunchesResult {
	f.logger.Info("Fetching starting", "total", len(fetchArgs), "goroutines", f.jobs)

	var results shared.GenericLaunchesResult
	resultsChannel := make(chan shared.GenericResult, len(fetchArgs))
	values := make([]interface{}, len(fetchArgs))
	for i := range fetchArgs {
		values[i] = fetchArgs[i]
	}

	shared.ForEveryStringWithBoundedGoroutines(f.jobs, values, func(i int, value interface{}) {
		var message string
		fetchArgs := value.(shared.VCSFetchRequest)
		f.logger.Info("Goroutine started", "#", i+1, "args", fetchArgs)

		err := f.fetchRepo(fetchArgs)
		if err != nil {
			message = err.Error()
		}

		if err != nil && err.Error() != "already up-to-date" {
			f.logger.Error("VCS plugin failed on fetch", "err", err)
			resultFetch := shared.GenericResult{Args: fetchArgs, Result: "", Status: "FAILED", Message: message}
			resultsChannel <- resultFetch
		} else {
			resultFetch := shared.GenericResult{Args: fetchArgs, Result: "", Status: "OK", Message: message}
			resultsChannel <- resultFetch
		}
	})

	close(resultsChannel)
	for result := range resultsChannel {
		results.Launches = append(results.Launches, result)
	}
	return results
}
