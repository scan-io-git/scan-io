package git

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

const (
	// minSHALength is the minimum accepted SHA length; tmp ref names embed sha[:minSHALength].
	minSHALength = 12
	// tmpRefCleanupTimeout bounds the background-context cleanup of temporary refs.
	tmpRefCleanupTimeout = 10 * time.Second
)

var (
	// deepenLadder deepens an already-present head commit's ancestry.
	deepenLadder = []int{10, 50, 200}
	// initialDeepenLadder starts at depth 1 because the SHAs may be absent entirely.
	initialDeepenLadder = []int{1, 10, 50, 200}
)

// shaLongEnough validates that every SHA is at least minSHALength chars.
func shaLongEnough(shas ...string) error {
	for _, s := range shas {
		if len(s) < minSHALength {
			return fmt.Errorf("invalid SHA %q: must be at least %d chars", s, minSHALength)
		}
	}
	return nil
}

// mergeBaseTmpRef returns the temporary ref name used for merge-base fetches of sha.
func mergeBaseTmpRef(sha string) string {
	return tmpRefPrefix + "mb-" + sha[:minSHALength]
}

// removeTmpRefs deletes refs via the git CLI using a background context so an
// expired caller context cannot leave refs dangling.
func (c *Client) removeTmpRefs(repoPath string, env []string, refs ...string) {
	ctx, cancel := context.WithTimeout(context.Background(), tmpRefCleanupTimeout)
	defer cancel()
	for _, ref := range refs {
		_, _ = c.runGit(ctx, repoPath, env, "update-ref", "-d", ref)
	}
}

// EnsureMergeBaseReachable progressively deepens headSHA's ancestry via go-git
// until mergeBaseSHA is reachable from headSHA. This is required so that
// change-aware scanners can run "git diff --merge-base <sha>" against the shallow
// clone.
//
// Strategy:
//  1. Short-circuit: check reachability immediately (full clone or prior fetch).
//  2. Re-fetch headSHA at increasing absolute depth (10→50→200). go-git's
//     FetchContext with Depth>1 deepens the existing shallow clone — getWants
//     adds the commit to the request even if it already exists, because
//     shallow=true fires when depth != 1 and .git/shallow is non-empty. As
//     headSHA's ancestry grows, mergeBaseSHA appears in the object store as part
//     of the ancestry chain; no separate SHA fetch is needed.
//  3. After each deepen, call cleanShallowEntries to remove stale .git/shallow
//     entries. go-git's updateShallow only adds new entries; without this step
//     the git CLI would still treat headSHA as a root with no parents (the core
//     of go-git issue #1443).
//  4. Check reachability with go-git's Commit.IsAncestor. Returns (false, err)
//     when the walk hits the shallow boundary before reaching mergeBaseSHA —
//     treated as "need deeper" rather than "not an ancestor".
//  5. If go-git deepening fails, fall back to the git binary.
//
// Returns nil when mergeBaseSHA is confirmed reachable from headSHA.
// Returns an error when the commit cannot be made reachable within the depth
// budget; callers should fall back to MergeBaseSHA for compute+materialize.
func EnsureMergeBaseReachable(gitClient *Client, repoPath, headSHA, mergeBaseSHA string) error {
	if err := shaLongEnough(headSHA, mergeBaseSHA); err != nil {
		return err
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}

	mbHash := plumbing.NewHash(mergeBaseSHA)
	headHash := plumbing.NewHash(headSHA)

	// Sync .git/shallow with the object store BEFORE any reachability check.
	// go-git's IsAncestor walks raw parent links and ignores .git/shallow, while
	// the git CLI (what change-aware scanners invoke) respects it. After separate
	// depth-1 fetches of head and base, both commit objects can be present while
	// .git/shallow still lists head as a parentless root — IsAncestor then reports
	// reachable (e.g. when the merge-base is head's direct parent) although
	// "git merge-base" fails. Pruning stale shallow entries makes both views agree.
	if err := cleanShallowEntries(repo); err != nil {
		gitClient.logger.Warn("cleanShallowEntries failed before reachability check", "error", err)
	}

	// Step 1: short-circuit — already reachable (full clone or prior deep fetch).
	if reachable, _ := isMergeBaseReachable(repo, headHash, mbHash); reachable {
		return nil
	}

	remoteName, err := resolveRemoteName(repo)
	if err != nil {
		return fmt.Errorf("no remote configured for deepening: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitClient.timeout)
	defer cancel()

	headTmpRef := plumbing.ReferenceName(mergeBaseTmpRef(headSHA))
	headSpec := config.RefSpec(fmt.Sprintf("+%s:%s", headSHA, headTmpRef))
	defer func() { _ = repo.Storer.RemoveReference(headTmpRef) }()

	insecure := InsecureFromCfg(gitClient.globalConfig)

	// Steps 3-4: iteratively deepen headSHA and check reachability.
	for _, depth := range deepenLadder {
		fetchErr := repo.FetchContext(ctx, &git.FetchOptions{
			RemoteName:      remoteName,
			Auth:            gitClient.auth,
			InsecureSkipTLS: insecure,
			Depth:           depth,
			RefSpecs:        []config.RefSpec{headSpec},
			Tags:            git.NoTags,
			Force:           true,
		})
		if fetchErr != nil && fetchErr != git.NoErrAlreadyUpToDate {
			gitClient.logger.Warn("go-git deepening failed, trying git binary fallback",
				"depth", depth, "error", fetchErr)
			break
		}

		// Prune stale .git/shallow entries that go-git's updateShallow leaves
		// behind (it adds new boundaries but never removes old ones).
		if err := cleanShallowEntries(repo); err != nil {
			gitClient.logger.Warn("cleanShallowEntries failed", "error", err)
			break
		}

		reachable, err := isMergeBaseReachable(repo, headHash, mbHash)
		if err == nil && reachable {
			return nil
		}
		// Continue deepening regardless of whether err is nil or non-nil:
		// - err != nil: IsAncestor hit a shallow boundary (missing parent object); try deeper.
		// - err == nil && !reachable: go-git stopped cleanly at the shallow boundary without
		//   finding mergeBaseSHA — this is NOT a definitive "not an ancestor" since M may
		//   simply lie beyond the current depth. Never conclude "not an ancestor" until the
		//   full depth budget is exhausted and the CLI fallback also fails.
		gitClient.logger.Debug("merge-base not yet reachable, deepening further",
			"depth", depth, "reachable", reachable, "err", err)
	}

	// Step 5: git binary fallback — deepen and verify via CLI.
	// Give the fallback its own timeout: the go-git loop may have consumed most of
	// the original ctx budget, and we don't want three depth iterations to starve
	// the CLI path of all remaining time.
	cliCtx, cliCancel := context.WithTimeout(context.Background(), gitClient.timeout)
	defer cliCancel()
	return ensureMergeBaseReachableViaCLI(gitClient, cliCtx, repoPath, headSHA, mergeBaseSHA)
}

// isMergeBaseReachable checks whether mbHash is an ancestor of headHash using
// the local go-git object graph. Returns (true, nil) when reachable,
// (false, nil) when definitively not an ancestor, and (false, err) when the
// walk hits a shallow boundary before reaching a conclusion.
func isMergeBaseReachable(repo *git.Repository, headHash, mbHash plumbing.Hash) (bool, error) {
	headCommit, err := repo.CommitObject(headHash)
	if err != nil {
		return false, err
	}
	mbCommit, err := repo.CommitObject(mbHash)
	if err != nil {
		return false, err
	}
	return mbCommit.IsAncestor(headCommit)
}

// cleanShallowEntries removes entries from .git/shallow whose parents are now
// present in the local object store. go-git's updateShallow only adds new
// shallow boundaries; this function removes ones that are no longer valid so
// the git CLI sees the correct (deeper) ancestry.
func cleanShallowEntries(repo *git.Repository) error {
	shallows, err := repo.Storer.Shallow()
	if err != nil || len(shallows) == 0 {
		return err
	}
	var keep []plumbing.Hash
	for _, sha := range shallows {
		commit, err := repo.CommitObject(sha)
		if err != nil {
			keep = append(keep, sha) // can't inspect; leave it
			continue
		}
		// Root commits (no parents) are never shallow.
		stillShallow := false
		for _, p := range commit.ParentHashes {
			if _, err := repo.CommitObject(p); err != nil {
				stillShallow = true
				break
			}
		}
		if stillShallow {
			keep = append(keep, sha)
		}
	}
	return repo.Storer.SetShallow(keep)
}

// deepenToMergeBase progressively re-fetches fetchSHAs at each depth (a single
// "git fetch --depth=N --no-tags <remote> +sha:tmpRef..." per depth), then runs
// "git merge-base headSHA otherSHA". A result is accepted when it equals wantSHA,
// or is any non-empty SHA when wantSHA is "". Returns (mb, nil) on success,
// ("", nil) when the depth budget is exhausted, ("", err) when a fetch fails.
func (c *Client) deepenToMergeBase(ctx context.Context, repoPath string, env []string,
	remoteName string, depths []int, fetchSHAs []string, headSHA, otherSHA, wantSHA string) (string, error) {

	refspecs := make([]string, 0, len(fetchSHAs))
	tmpRefs := make([]string, 0, len(fetchSHAs))
	for _, sha := range fetchSHAs {
		ref := mergeBaseTmpRef(sha)
		tmpRefs = append(tmpRefs, ref)
		refspecs = append(refspecs, fmt.Sprintf("+%s:%s", sha, ref))
	}
	defer c.removeTmpRefs(repoPath, env, tmpRefs...)

	for _, depth := range depths {
		args := append([]string{"fetch", fmt.Sprintf("--depth=%d", depth), "--no-tags", remoteName}, refspecs...)
		if _, err := c.runGit(ctx, repoPath, env, args...); err != nil {
			return "", fmt.Errorf("deepen fetch at depth %d: %w", depth, err)
		}
		out, err := c.runGit(ctx, repoPath, env, "merge-base", headSHA, otherSHA)
		if err == nil && out != "" && (wantSHA == "" || out == wantSHA) {
			return out, nil
		}
	}
	return "", nil
}

// ensureMergeBaseReachableViaCLI is the git binary fallback for
// EnsureMergeBaseReachable. It deepens headSHA via explicit SHA fetches and
// verifies that mergeBaseSHA becomes reachable using git merge-base.
func ensureMergeBaseReachableViaCLI(c *Client, ctx context.Context, repoPath, headSHA, mergeBaseSHA string) error {
	if err := shaLongEnough(headSHA); err != nil {
		return err
	}

	env, err := c.gitCLIEnv()
	if err != nil {
		return fmt.Errorf("git binary unavailable: %w", err)
	}

	remoteName, err := c.resolveRemoteNameCLI(ctx, repoPath, env)
	if err != nil {
		return fmt.Errorf("no remote configured for deepening: %w", err)
	}

	mb, err := c.deepenToMergeBase(ctx, repoPath, env, remoteName, deepenLadder,
		[]string{headSHA}, headSHA, mergeBaseSHA, mergeBaseSHA)
	if err != nil {
		return fmt.Errorf("merge-base %s not reachable: git fetch failed: %w", mergeBaseSHA, err)
	}
	if mb == "" {
		return fmt.Errorf("merge-base %s not reachable from %s within depth budget", mergeBaseSHA, headSHA)
	}
	return nil
}

// MergeBaseSHA computes the merge-base (fork point) SHA between the PR head and the
// target branch at repoPath, fetching the minimum history via the git CLI.
// Best-effort: returns ("", nil) when the fork point cannot be determined (passphrase
// SSH key, git binary absent, server error) so callers proceed without it.
func (c *Client) MergeBaseSHA(repoPath, headSHA, baseBranch, baseSHA string) (string, error) {
	if headSHA == "" && baseSHA == "" {
		c.logger.Warn("merge-base computation skipped: neither headSHA nor baseSHA available")
		return "", nil
	}

	env, err := c.gitCLIEnv()
	if err != nil {
		c.logger.Warn("merge-base computation skipped: CLI auth not supported", "reason", err)
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	isShallow, _ := c.runGit(ctx, repoPath, env, "rev-parse", "--is-shallow-repository")
	if isShallow == "true" {
		return c.mergeBaseShallow(ctx, repoPath, env, headSHA, baseSHA)
	}
	return c.mergeBaseFull(ctx, repoPath, env, headSHA, baseSHA)
}

// mergeBaseShallow computes the merge-base for a shallow clone.
//
// The repo was produced by cloneCommit (fork PRs) or cloneAtRef (branch PRs). In both
// cases the origin remote may have no configured fetch refspec, so git operations that
// require one (--shallow-exclude, --deepen without explicit refs) fail silently or with
// "no commits selected for shallow requests". Using those would either return an empty
// result or, worse, tip-1 — a wrong baseline that silently shrinks the diff window and
// causes missed findings in diff-aware scanning.
//
// Instead, re-fetch both headSHA and baseSHA with progressively deeper history using the
// explicit +<sha>:<tmpRef> refspec that works without a configured fetch refspec. Once
// their histories overlap, git merge-base returns the true common ancestor. If the
// histories do not meet within the depth budget the function returns ("", nil), which
// causes the caller to fall back to a full-tree scan — no missed findings.
func (c *Client) mergeBaseShallow(ctx context.Context, repoPath string, env []string, headSHA, baseSHA string) (string, error) {
	if err := shaLongEnough(headSHA, baseSHA); err != nil {
		c.logger.Warn("merge-base skipped", "reason", err)
		return "", nil
	}

	remoteName, err := c.resolveRemoteNameCLI(ctx, repoPath, env)
	if err != nil {
		c.logger.Warn("merge-base skipped: no remote configured")
		return "", nil
	}

	mb, err := c.deepenToMergeBase(ctx, repoPath, env, remoteName, initialDeepenLadder,
		[]string{headSHA, baseSHA}, headSHA, baseSHA, "")
	if err != nil {
		c.logger.Warn("merge-base: fetch failed", "error", err)
		return "", nil
	}
	if mb == "" {
		c.logger.Warn("merge-base skipped: common ancestor not found within depth budget (260 commits); falling back to full scan")
	}
	return mb, nil
}

// mergeBaseFull computes the merge-base for a full (non-shallow) clone.
// Both commits are already present in the local store, so git merge-base suffices.
func (c *Client) mergeBaseFull(ctx context.Context, repoPath string, env []string, headSHA, baseSHA string) (string, error) {
	if headSHA == "" || baseSHA == "" {
		c.logger.Warn("merge-base skipped: headSHA or baseSHA empty for full-clone path")
		return "", nil
	}
	out, err := c.runGit(ctx, repoPath, env, "merge-base", headSHA, baseSHA)
	if err != nil {
		c.logger.Warn("merge-base skipped: merge-base command failed", "error", err)
		return "", nil
	}
	return out, nil
}
