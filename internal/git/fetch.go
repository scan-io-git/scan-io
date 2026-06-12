package git

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"

	log "github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// EnsureCommitPresent fetches the given commit SHA into repoPath if it is not
// already present in the local object store. It is safe to call when the commit
// is already available (no-op). Returns an error if the SHA is empty or the
// fetch fails.
func EnsureCommitPresent(gitClient *Client, repoPath, sha string) error {
	if sha == "" {
		return fmt.Errorf("commit SHA is required")
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository %q: %w", repoPath, err)
	}

	return ensureCommitPresent(gitClient, repo, plumbing.NewHash(sha))
}

// ensureCommitPresent verifies that the given commit hash exists locally, using
// the supplied gitClient to fetch it from the remote when required.
func ensureCommitPresent(gitClient *Client, repo *git.Repository, hash plumbing.Hash) error {
	if _, err := repo.CommitObject(hash); err != nil {
		gitClient.logger.Debug("commit missing locally, attempting fetch", "hash", hash.String())
		if err := fetchCommit(gitClient, repo, hash); err != nil {
			return err
		}
	}
	return nil
}

// fetchCommit synchronises the provided commit hash into the local repository
// using the gitClient's authentication, TLS, and timeout settings.
func fetchCommit(gitClient *Client, repo *git.Repository, hash plumbing.Hash) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitClient.timeout)
	defer cancel()

	gitLog := log.GetLoggerOutput(gitClient.logger)
	output := io.MultiWriter(
		gitLog,
		os.Stderr,
	)

	insecure := InsecureFromCfg(gitClient.globalConfig)

	remoteName := origin
	if _, err := repo.Remote(remoteName); err != nil {
		remotes, rErr := repo.Remotes()
		if rErr != nil || len(remotes) == 0 {
			return fmt.Errorf("no remotes available to fetch commit %s", hash.String())
		}
		remoteName = remotes[0].Config().Name
	}

	tmpRef := plumbing.ReferenceName(fmt.Sprintf(tmpRefPrefix+"%s", hash.String()))
	refspec := config.RefSpec(fmt.Sprintf("+%s:%s", hash.String(), tmpRef.String()))

	gitClient.logger.Debug("fetching commit", "remote", remoteName, "hash", hash.String())

	fetchErr := repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName:      remoteName,
		Auth:            gitClient.auth,
		InsecureSkipTLS: insecure,
		Progress:        output,
		Depth:           1,
		RefSpecs:        []config.RefSpec{refspec},
		Tags:            git.NoTags,
	})

	if fetchErr != nil && fetchErr != git.NoErrAlreadyUpToDate {
		gitClient.logger.Warn("fetch commit failed", "hash", hash.String(), "error", fetchErr)
		return fetchErr
	}

	defer func() {
		_ = repo.Storer.RemoveReference(tmpRef)
	}()

	if _, err := repo.CommitObject(hash); err != nil {
		return err
	}
	gitClient.logger.Debug("commit fetched", "hash", hash.String())
	return nil
}

// resolveRemoteName returns the name of the preferred remote: try "origin"
// first, fall back to the first configured remote.
func resolveRemoteName(repo *git.Repository) (string, error) {
	if _, err := repo.Remote(origin); err == nil {
		return origin, nil
	}
	remotes, err := repo.Remotes()
	if err != nil || len(remotes) == 0 {
		return "", fmt.Errorf("no remotes configured in repository")
	}
	return remotes[0].Config().Name, nil
}
