package fetcher

import (
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"

	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
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

func (f Fetcher) PrepFetchArgs(logger hclog.Logger, repos []shared.RepositoryParams) ([]shared.VCSFetchRequest, error) {
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

		targetFolder := shared.GetRepoPath(logger, strings.ToLower(domain), filepath.Join(strings.ToLower(repo.Namespace), strings.ToLower(repo.RepoName)))

		fetchArgs = append(fetchArgs, shared.VCSFetchRequest{
			CloneURL:     cloneURL,
			Branch:       f.branch,
			AuthType:     f.authType,
			SSHKey:       f.sshKey,
			TargetFolder: targetFolder,
		})

	}
	return fetchArgs, nil
}

func (f Fetcher) fetchRepo(cfg *config.Config, fetchArg shared.VCSFetchRequest) error {

	shared.WithPlugin(cfg, "plugin-vcs", shared.PluginTypeVCS, f.vcsPluginName, func(raw interface{}) {
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

func (f Fetcher) FetchRepos(cfg *config.Config, fetchArgs []shared.VCSFetchRequest) error {

	f.logger.Info("Fetching starting", "total", len(fetchArgs), "goroutines", f.jobs)

	values := make([]interface{}, len(fetchArgs))
	for i := range fetchArgs {
		values[i] = fetchArgs[i]
	}

	shared.ForEveryStringWithBoundedGoroutines(f.jobs, values, func(i int, value interface{}) {
		fetchArgs := value.(shared.VCSFetchRequest)
		f.logger.Info("Goroutine started", "#", i+1, "args", fetchArgs)

		err := f.fetchRepo(cfg, fetchArgs)
		if err != nil {
			f.logger.Error("fetcher's fetchRepo() failed", "err", err)
		}
	})

	return nil
}
