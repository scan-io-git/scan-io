package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sourcegraph/go-diff/diff"

	sharedfiles "github.com/scan-io-git/scan-io/pkg/shared/files"
	log "github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// AddedLines returns, for every file touched between baseHash and headHash, a map
// of new-file line numbers to the textual content that was added. Returned line
// numbers are 1-based and only include additions; deletions and context lines are
// ignored. Paths that are deleted or outside the optional filter list are skipped.
// The provided gitClient is used to ensure both commits are available locally.
func AddedLines(gitClient *Client, repoPath, baseHash, headHash string, filters []string) (map[string]map[int]string, error) {
	if baseHash == "" {
		return nil, fmt.Errorf("base hash is required to compute diff")
	}
	if headHash == "" {
		return nil, fmt.Errorf("head hash is required to compute diff")
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository %q: %w", repoPath, err)
	}

	baseHashObj := plumbing.NewHash(baseHash)
	headHashObj := plumbing.NewHash(headHash)

	if err := ensureCommitPresent(gitClient, repo, baseHashObj); err != nil {
		return nil, fmt.Errorf("failed to resolve base commit %q: %w", baseHash, err)
	}
	if err := ensureCommitPresent(gitClient, repo, headHashObj); err != nil {
		return nil, fmt.Errorf("failed to resolve head commit %q: %w", headHash, err)
	}

	baseCommit, err := repo.CommitObject(baseHashObj)
	if err != nil {
		return nil, err
	}
	headCommit, err := repo.CommitObject(headHashObj)
	if err != nil {
		return nil, err
	}

	baseTree, err := baseCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to load base tree: %w", err)
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to load head tree: %w", err)
	}

	changes, err := baseTree.Diff(headTree)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	var patchBuf bytes.Buffer
	for _, c := range changes {
		p, pErr := safePatch(c)
		if pErr != nil {
			gitClient.logger.Warn("skipping file in diff (too large for in-memory diff)", "err", pErr)
			continue
		}
		patchBuf.WriteString(p.String())
	}

	parsed, err := diff.ParseMultiFileDiff(patchBuf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	allowed := buildFilterSet(filters)
	result := make(map[string]map[int]string)

	for _, fd := range parsed {
		// checking deleted files and with no changes
		if fd == nil || fd.NewName == "/dev/null" || len(fd.Hunks) == 0 {
			continue
		}

		path := strings.TrimPrefix(fd.NewName, "b/")
		if len(allowed) > 0 && !allowed[path] {
			continue
		}

		added := make(map[int]string)

		for _, h := range fd.Hunks {
			if h == nil {
				continue
			}
			lineNo := int(h.NewStartLine)
			if lineNo <= 0 {
				lineNo = 1
			}
			for _, bodyLine := range bytes.Split(h.Body, []byte("\n")) {
				if len(bodyLine) == 0 {
					continue
				}

				switch bodyLine[0] {
				case '+':
					added[lineNo] = string(bodyLine[1:])
					lineNo++
				case '-':
					// deletion; do not advance new file line counter
					continue
				default:
					lineNo++
				}
			}
		}

		if len(added) > 0 {
			result[path] = added
		}
	}

	return result, nil
}

// MaterializeDiff writes diff-focused copies of the provided files into diffRoot.
// Every output file mirrors the repository structure but contains only the newly
// added lines (other positions remain blank), allowing scanners to operate on diff
// hunks without re-running git diff. When no additions are detected the function
// exits early without writing anything. The gitClient is used for commit lookups
// and logging.
func MaterializeDiff(gitClient *Client, repoRoot, diffRoot, baseSHA, headSHA string, files []string) error {
	if err := sharedfiles.CreateFolderIfNotExists(diffRoot); err != nil {
		return fmt.Errorf("prepare diff folder: %w", err)
	}

	paths := uniqueNonEmpty(files)
	addedLines, err := AddedLines(gitClient, repoRoot, baseSHA, headSHA, paths)
	if err != nil {
		return err
	}

	if len(addedLines) == 0 {
		gitClient.logger.Info("no additions detected between commits", "base", baseSHA, "head", headSHA)
		return nil
	}

	if len(paths) == 0 {
		paths = sortedKeys(addedLines)
	}

	for _, relPath := range paths {
		relPath = strings.TrimSpace(relPath)
		if relPath == "" {
			continue
		}

		lines := addedLines[relPath]
		if len(lines) == 0 {
			gitClient.logger.Debug("skipping file with no additions", "path", relPath)
			continue
		}

		if err := writeSparseFile(repoRoot, diffRoot, relPath, lines); err != nil {
			gitClient.logger.Warn("skipping file writing due to error", "err", err)
			continue
		}
	}

	return nil
}

// writeSparseFile writes a copy of relPath into diffRoot, keeping only the line
// numbers present in the supplied map and leaving other positions empty. The file
// retains trailing newline semantics from the source to minimise surprises for
// downstream tools.
func writeSparseFile(repoRoot, diffRoot, relPath string, lines map[int]string) error {
	srcPath := filepath.Join(repoRoot, relPath)
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read %q for diff materialisation: %w", relPath, err)
	}

	content := string(data)
	headLines := strings.Split(content, "\n")
	diffLines := make([]string, len(headLines))

	for lineNumber, value := range lines {
		if lineNumber <= 0 {
			continue
		}
		index := lineNumber - 1
		if index >= len(diffLines) {
			diffLines = append(diffLines, make([]string, index-len(diffLines)+1)...)
		}
		diffLines[index] = value
	}

	output := strings.Join(diffLines, "\n")
	if strings.HasSuffix(content, "\n") && !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	dstPath := filepath.Join(diffRoot, relPath)
	if err := sharedfiles.CreateFolderIfNotExists(filepath.Dir(dstPath)); err != nil {
		return fmt.Errorf("failed to prepare folder for %q: %w", relPath, err)
	}

	if err := os.WriteFile(dstPath, []byte(output), 0600); err != nil {
		return fmt.Errorf("failed to write diff file %q: %w", dstPath, err)
	}

	return nil
}

// safePatch computes the patch for a single file change, recovering from panics
// that sergi/go-diff can produce when a file has more unique lines than Unicode
// rune space (~1.1M). On panic the error is returned and the caller skips the file.
func safePatch(c *object.Change) (p *object.Patch, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("diff panic: %v", r)
		}
	}()
	p, err = c.Patch()
	return
}

// buildFilterSet returns an O(1) lookup table for the provided filter slice.
// Nil is returned when no filters are supplied to avoid extra map checks downstream.
func buildFilterSet(filters []string) map[string]bool {
	if len(filters) == 0 {
		return nil
	}
	set := make(map[string]bool, len(filters))
	for _, f := range filters {
		set[f] = true
	}
	return set
}

// uniqueNonEmpty strips empty entries and duplicates from the path list while
// preserving the original order. Paths are kept verbatim (no trimming) so values
// containing leading/trailing spaces remain intact.
func uniqueNonEmpty(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, exists := set[item]; exists {
			continue
		}
		set[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

// sortedKeys returns the map keys in ascending order to provide deterministic
// iteration when no explicit filter list was provided.
func sortedKeys(m map[string]map[int]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

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
		if fetchErr != nil {
			if fetchErr == git.NoErrAlreadyUpToDate {
				gitClient.logger.Debug("commit already available", "hash", hash.String())
			} else {
				gitClient.logger.Warn("fetch commit failed", "hash", hash.String(), "error", fetchErr)
				return fetchErr
			}
		}
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

// EnsureMergeBaseReachable progressively deepens headSHA's ancestry via go-git
// until mergeBaseSHA is reachable from headSHA. This is required so that callers
// so that change-aware scanners can run "git diff --merge-base <sha>" against the shallow
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
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}

	mbHash := plumbing.NewHash(mergeBaseSHA)
	headHash := plumbing.NewHash(headSHA)

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

	headTmpRef := plumbing.ReferenceName(tmpRefPrefix + "mr-head-" + headSHA[:12])
	headSpec := config.RefSpec(fmt.Sprintf("+%s:%s", headSHA, headTmpRef))
	defer func() { _ = repo.Storer.RemoveReference(headTmpRef) }()

	insecure := InsecureFromCfg(gitClient.globalConfig)

	// Steps 3-4: iteratively deepen headSHA and check reachability.
	for _, depth := range []int{10, 50, 200} {
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
		if err == nil {
			if reachable {
				return nil
			}
			// err==nil && !reachable: mergeBaseSHA is definitively not an ancestor.
			return fmt.Errorf("merge-base %s is not an ancestor of head %s", mergeBaseSHA, headSHA)
		}
		// err != nil: shallow boundary reached before finding mergeBaseSHA; try deeper.
		gitClient.logger.Debug("merge-base not yet reachable, deepening further",
			"depth", depth, "error", err)
	}

	// Step 5: git binary fallback — deepen and verify via CLI.
	return ensureMergeBaseReachableViaCLI(gitClient, ctx, repoPath, headSHA, mergeBaseSHA)
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

// resolveRemoteName returns the name of the preferred remote, mirroring the
// logic in fetchCommit: try "origin" first, fall back to the first configured.
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

// ensureMergeBaseReachableViaCLI is the git binary fallback for
// EnsureMergeBaseReachable. It deepens headSHA via explicit SHA fetches and
// verifies that mergeBaseSHA becomes reachable using git merge-base.
func ensureMergeBaseReachableViaCLI(c *Client, ctx context.Context, repoPath, headSHA, mergeBaseSHA string) error {
	env, err := c.gitCLIEnv()
	if err != nil {
		return fmt.Errorf("git binary unavailable: %w", err)
	}

	headTmpRef := fmt.Sprintf("refs/scanio/tmp/mr-head-%s", headSHA[:12])
	defer c.runGit(ctx, repoPath, env, "update-ref", "-d", headTmpRef) //nolint:errcheck

	for _, depth := range []int{10, 50, 200} {
		if _, err := c.runGit(ctx, repoPath, env,
			"fetch", fmt.Sprintf("--depth=%d", depth), origin,
			fmt.Sprintf("+%s:%s", headSHA, headTmpRef)); err != nil {
			c.logger.Warn("git binary deepen failed", "depth", depth, "error", err)
			return fmt.Errorf("merge-base %s not reachable: git fetch failed: %w", mergeBaseSHA, err)
		}
		if out, err := c.runGit(ctx, repoPath, env, "merge-base", headSHA, mergeBaseSHA); err == nil && out == mergeBaseSHA {
			return nil
		}
	}
	return fmt.Errorf("merge-base %s not reachable from %s within depth budget", mergeBaseSHA, headSHA)
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
	if headSHA == "" || baseSHA == "" {
		c.logger.Warn("merge-base skipped: headSHA and baseSHA required")
		return "", nil
	}

	// Resolve the remote name the same way fetchCommit does: prefer "origin", fall
	// back to the first configured remote. Hardcoding "origin" would silently fail
	// for repos whose remote has a different name.
	remoteName := origin
	if out, err := c.runGit(ctx, repoPath, env, "remote", "get-url", origin); err != nil || out == "" {
		if list, lErr := c.runGit(ctx, repoPath, env, "remote"); lErr == nil && list != "" {
			remoteName = strings.Fields(list)[0]
		} else {
			c.logger.Warn("merge-base skipped: no remote configured")
			return "", nil
		}
	}

	headRef := tmpRefPrefix + "mb-head-" + headSHA[:12]
	baseRef := tmpRefPrefix + "mb-base-" + baseSHA[:12]
	headSpec := fmt.Sprintf("+%s:%s", headSHA, headRef)
	baseSpec := fmt.Sprintf("+%s:%s", baseSHA, baseRef)

	// Use a background context for cleanup so that expired/cancelled fetch contexts
	// do not leave tmp refs dangling in the repo.
	defer func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanCancel()
		_, _ = c.runGit(cleanCtx, repoPath, env, "update-ref", "-d", headRef)
		_, _ = c.runGit(cleanCtx, repoPath, env, "update-ref", "-d", baseRef)
	}()

	for _, depth := range []int{1, 10, 50, 200} {
		if _, err := c.runGit(ctx, repoPath, env,
			"fetch", fmt.Sprintf("--depth=%d", depth), remoteName,
			headSpec, baseSpec); err != nil {
			c.logger.Warn("merge-base: fetch failed", "depth", depth, "error", err)
			return "", nil
		}
		if mb, err := c.runGit(ctx, repoPath, env, "merge-base", headSHA, baseSHA); err == nil && mb != "" {
			return mb, nil
		}
	}

	c.logger.Warn("merge-base skipped: common ancestor not found within depth budget (260 commits); falling back to full scan")
	return "", nil
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
