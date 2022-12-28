package fetcher

import (
	"path/filepath"

	"github.com/hashicorp/go-hclog"

	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
)

type Fetcher struct {
	authType      string
	sshKey        string
	jobs          int
	vcsPluginName string
	rmExts        []string
	logger        hclog.Logger
}

func New(authType string, sshKey string, jobs int, vcsPluginName string, rmExts []string, logger hclog.Logger) Fetcher {
	return Fetcher{
		authType:      authType,
		sshKey:        sshKey,
		jobs:          jobs,
		vcsPluginName: vcsPluginName,
		rmExts:        rmExts,
		logger:        logger,
	}
}

func (f Fetcher) PrepFetchArgs(repos []shared.RepositoryParams) ([]shared.VCSFetchRequest, error) {
	var fetchArgs []shared.VCSFetchRequest

	for _, repo := range repos {

		cloneURL := repo.SshLink
		if f.authType == "http" {
			cloneURL = repo.HttpLink
		}

		domain, err := utils.GetDomain(cloneURL)
		if err != nil {
			return nil, err
		}

		targetFolder := shared.GetRepoPath(domain, filepath.Join(repo.Namespace, repo.RepoName))

		fetchArgs = append(fetchArgs, shared.VCSFetchRequest{
			CloneURL:     cloneURL,
			AuthType:     f.authType,
			SSHKey:       f.sshKey,
			TargetFolder: targetFolder,
		})

	}
	return fetchArgs, nil
}

func (f Fetcher) fetchRepo(fetchArg shared.VCSFetchRequest) error {

	shared.WithPlugin("plugin-vcs", shared.PluginTypeVCS, f.vcsPluginName, func(raw interface{}) {
		vcsName := raw.(shared.VCS)
		err := vcsName.Fetch(fetchArg)
		if err != nil {
			f.logger.Error("vcs plugin failed on fetch", "err", err)
		} else {
			utils.FindByExtAndRemove(fetchArg.TargetFolder, f.rmExts)
		}
	})

	return nil
}

func (f Fetcher) FetchRepos(fetchArgs []shared.VCSFetchRequest) error {

	f.logger.Info("Fetching starting", "total", len(fetchArgs), "goroutines", f.jobs)

	values := make([]interface{}, len(fetchArgs))
	for i := range fetchArgs {
		values[i] = fetchArgs[i]
	}

	shared.ForEveryStringWithBoundedGoroutines(f.jobs, values, func(i int, value interface{}) {
		fetchArgs := value.(shared.VCSFetchRequest)
		f.logger.Info("Goroutine started", "#", i+1, "args", fetchArgs)

		err := f.fetchRepo(fetchArgs)
		if err != nil {
			f.logger.Error("fetcher's fetchRepo() failed", "err", err)
		}
	})

	return nil
}
