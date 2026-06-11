package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hashicorp/go-hclog"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

func TestAddedLines(t *testing.T) {
	repoDir, baseHash, headHash := setupDiffRepo(t)

	client := newTestGitClient()

	got, err := AddedLines(client, repoDir, baseHash, headHash, nil)
	if err != nil {
		t.Fatalf("AddedLines returned error: %v", err)
	}

	wantData := map[int]string{
		2: "beta2",
		4: "delta",
	}
	if diff := compareLineMaps(wantData, got["data.txt"]); diff != "" {
		t.Fatalf("unexpected additions for data.txt:\n%s", diff)
	}

	wantNew := map[int]string{
		1: "onlyline",
	}
	if diff := compareLineMaps(wantNew, got["new.txt"]); diff != "" {
		t.Fatalf("unexpected additions for new.txt:\n%s", diff)
	}

	wantPlain := map[int]string{
		1: "noline",
	}
	if diff := compareLineMaps(wantPlain, got["plain.txt"]); diff != "" {
		t.Fatalf("unexpected additions for plain.txt:\n%s", diff)
	}

	filtered, err := AddedLines(client, repoDir, baseHash, headHash, []string{"new.txt"})
	if err != nil {
		t.Fatalf("AddedLines with filters returned error: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 file after filtering, got %d", len(filtered))
	}
	if diff := compareLineMaps(wantNew, filtered["new.txt"]); diff != "" {
		t.Fatalf("unexpected filtered additions:\n%s", diff)
	}
}

func TestMaterializeDiff(t *testing.T) {
	repoDir, baseHash, headHash := setupDiffRepo(t)
	diffRoot := filepath.Join(t.TempDir(), "diff")

	client := newTestGitClient()

	if err := MaterializeDiff(client, repoDir, diffRoot, baseHash, headHash, nil); err != nil {
		t.Fatalf("MaterializeDiff returned error: %v", err)
	}

	dataPath := filepath.Join(diffRoot, "data.txt")
	if b, err := os.ReadFile(dataPath); err != nil {
		t.Fatalf("reading data diff: %v", err)
	} else {
		want := "\nbeta2\n\ndelta\n"
		if string(b) != want {
			t.Fatalf("unexpected data diff contents:\nwant %q\n got %q", want, string(b))
		}
	}

	newPath := filepath.Join(diffRoot, "new.txt")
	if b, err := os.ReadFile(newPath); err != nil {
		t.Fatalf("reading new diff: %v", err)
	} else {
		want := "onlyline\n"
		if string(b) != want {
			t.Fatalf("unexpected new diff contents:\nwant %q\n got %q", want, string(b))
		}
	}

	plainPath := filepath.Join(diffRoot, "plain.txt")
	if b, err := os.ReadFile(plainPath); err != nil {
		t.Fatalf("reading plain diff: %v", err)
	} else {
		want := "noline"
		if string(b) != want {
			t.Fatalf("unexpected plain diff contents:\nwant %q\n got %q", want, string(b))
		}
	}
}

// func TestAddedLinesFetchesMissingCommit(t *testing.T) {
// 	repoDir, baseHash, headHash := setupDiffRepoWithRemote(t)

// 	repo, err := git.PlainOpen(repoDir)
// 	if err != nil {
// 		t.Fatalf("PlainOpen clone: %v", err)
// 	}

// 	if _, err := repo.CommitObject(plumbing.NewHash(baseHash)); err == nil {
// 		t.Fatalf("expected base commit to be absent before AddedLines fetch")
// 	}

// 	client := newTestGitClient()
// 	got, err := AddedLines(client, repoDir, baseHash, headHash, nil)
// 	if err != nil {
// 		t.Fatalf("AddedLines returned error: %v", err)
// 	}
// 	if len(got) == 0 {
// 		t.Fatalf("expected diff results after fetching missing commit")
// 	}

// 	if _, err := repo.CommitObject(plumbing.NewHash(baseHash)); err != nil {
// 		t.Fatalf("base commit still missing after fetch: %v", err)
// 	}
// }

// setupDiffRepo initialises a temporary repository with two commits and returns
// the repo path along with base and head commit hashes.
func setupDiffRepo(t *testing.T) (string, string, string) {
	t.Helper()

	repoDir := t.TempDir()
	repo, err := git.PlainInit(repoDir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}

	baseFiles := map[string]string{
		"data.txt": "alpha\nbeta\ngamma\n",
	}
	baseHash := commitFiles(t, wt, baseFiles, "base commit")

	headFiles := map[string]string{
		"data.txt":  "alpha\nbeta2\ngamma\ndelta\n",
		"new.txt":   "onlyline\n",
		"plain.txt": "noline",
	}
	headHash := commitFiles(t, wt, headFiles, "head commit")

	return repoDir, baseHash.String(), headHash.String()
}

func newTestGitClient() *Client {
	return &Client{
		logger:       hclog.NewNullLogger(),
		timeout:      time.Minute,
		globalConfig: &config.Config{},
	}
}

// func setupDiffRepoWithRemote(t *testing.T) (string, string, string) {
// 	t.Helper()

// 	originDir := filepath.Join(t.TempDir(), "origin")
// 	if _, err := git.PlainInit(originDir, true); err != nil {
// 		t.Fatalf("PlainInit origin: %v", err)
// 	}

// 	repoDir := filepath.Join(t.TempDir(), "seed")
// 	repo, err := git.PlainInit(repoDir, false)
// 	if err != nil {
// 		t.Fatalf("PlainInit seed: %v", err)
// 	}

// 	_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{originDir}})
// 	if err != nil {
// 		t.Fatalf("CreateRemote: %v", err)
// 	}

// 	wt, err := repo.Worktree()
// 	if err != nil {
// 		t.Fatalf("Worktree: %v", err)
// 	}

// 	baseFiles := map[string]string{
// 		"data.txt": "alpha\nbeta\ngamma\n",
// 	}
// 	baseHash := commitFiles(t, wt, baseFiles, "base commit")
// 	if err := repo.Push(&git.PushOptions{RemoteName: "origin"}); err != nil {
// 		t.Fatalf("push base: %v", err)
// 	}

// 	headFiles := map[string]string{
// 		"data.txt":  "alpha\nbeta2\ngamma\ndelta\n",
// 		"new.txt":   "onlyline\n",
// 		"plain.txt": "noline",
// 	}
// 	headHash := commitFiles(t, wt, headFiles, "head commit")
// 	if err := repo.Push(&git.PushOptions{RemoteName: "origin"}); err != nil {
// 		t.Fatalf("push head: %v", err)
// 	}

// 	cloneDir := filepath.Join(t.TempDir(), "clone")
// 	if _, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
// 		URL:           originDir,
// 		Depth:         1,
// 		SingleBranch:  true,
// 		ReferenceName: plumbing.NewBranchReferenceName("master"),
// 		Tags:          git.NoTags,
// 	}); err != nil {
// 		t.Fatalf("PlainClone: %v", err)
// 	}

// 	return cloneDir, baseHash.String(), headHash.String()
// }

func commitFiles(t *testing.T, wt *git.Worktree, files map[string]string, message string) plumbing.Hash {
	t.Helper()

	for path, content := range files {
		abs := filepath.Join(wt.Filesystem.Root(), path)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", abs, err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", abs, err)
		}
		if _, err := wt.Add(path); err != nil {
			t.Fatalf("add %s: %v", path, err)
		}
	}

	hash, err := wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{Name: "tester", Email: "tester@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	return hash
}

func compareLineMaps(want, got map[int]string) string {
	if len(want) != len(got) {
		return fmt.Sprintf("different map lengths: want %d got %d", len(want), len(got))
	}
	for k, v := range want {
		if got[k] != v {
			return fmt.Sprintf("mismatch for line %d: want %q got %q", k, v, got[k])
		}
	}
	return ""
}

// setupMergeBaseRepo creates a local bare origin with a master branch (C0-C3) and a
// feature branch that diverges at C1 (adds commits A and B), then shallow-clones the
// feature branch. It also simulates EnsureCommitPresent by fetching the master tip into
// a tmp ref that is immediately removed — leaving masterTipSHA in .git/shallow without
// any named ref, matching the production state before MergeBaseSHA is called.
//
// Returns: cloneDir, forkPointSHA (C1), featureTipSHA (B), masterTipSHA (C3).
func setupMergeBaseRepo(t *testing.T) (cloneDir, forkPointSHA, featureTipSHA, masterTipSHA string) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not in PATH; skipping merge-base CLI test")
	}

	tmp := t.TempDir()
	originDir := filepath.Join(tmp, "origin")
	seedDir := filepath.Join(tmp, "seed")
	cloneDir = filepath.Join(tmp, "clone")

	run := func(dir string, args ...string) string {
		cmd := exec.Command("git", args...)
		if dir != "" {
			cmd.Dir = dir
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v (dir=%q): %v\n%s", args, dir, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	if err := os.MkdirAll(originDir, 0o755); err != nil {
		t.Fatalf("mkdir origin: %v", err)
	}
	run(originDir, "init", "--bare")

	if err := os.MkdirAll(seedDir, 0o755); err != nil {
		t.Fatalf("mkdir seed: %v", err)
	}
	run(seedDir, "init")
	run(seedDir, "symbolic-ref", "HEAD", "refs/heads/master")
	run(seedDir, "config", "user.email", "test@test.com")
	run(seedDir, "config", "user.name", "Test")
	run(seedDir, "remote", "add", "origin", "file://"+originDir)

	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(seedDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	write("file.txt", "c0\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "C0")

	write("file.txt", "c1\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "C1")
	forkPointSHA = run(seedDir, "rev-parse", "HEAD")

	// Feature branch diverges from C1.
	run(seedDir, "checkout", "-b", "feature")
	write("feature.txt", "A\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "A")
	write("feature.txt", "B\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "B")
	run(seedDir, "push", "origin", "feature")
	featureTipSHA = run(seedDir, "rev-parse", "HEAD")

	// Advance master past C1.
	run(seedDir, "checkout", "master")
	write("file.txt", "c2\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "C2")
	write("file.txt", "c3\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "C3")
	run(seedDir, "push", "origin", "master")
	masterTipSHA = run(seedDir, "rev-parse", "HEAD")

	if err := os.MkdirAll(cloneDir, 0o755); err != nil {
		t.Fatalf("mkdir clone: %v", err)
	}
	run("", "clone",
		"--depth=1", "--branch=feature", "--single-branch", "--no-tags",
		"file://"+originDir, cloneDir)
	run(cloneDir, "config", "user.email", "test@test.com")
	run(cloneDir, "config", "user.name", "Test")

	// Simulate EnsureCommitPresent: fetch master tip by SHA into a tmp ref then remove it,
	// leaving the commit in .git/shallow with no named ref (matching production state).
	masterTmpRef := "refs/scanio/tmp/" + masterTipSHA
	run(cloneDir, "fetch", "--depth=1", "origin", masterTipSHA+":"+masterTmpRef)
	run(cloneDir, "update-ref", "-d", masterTmpRef)

	return cloneDir, forkPointSHA, featureTipSHA, masterTipSHA
}

func TestMergeBaseSHA_shallow(t *testing.T) {
	cloneDir, wantFork, headSHA, baseSHA := setupMergeBaseRepo(t)

	client := newTestGitClient()
	got, err := client.MergeBaseSHA(cloneDir, headSHA, "master", baseSHA)
	if err != nil {
		t.Fatalf("MergeBaseSHA returned unexpected error: %v", err)
	}
	if got != wantFork {
		t.Errorf("merge base = %q, want %q", got, wantFork)
	}
}

func TestMergeBaseSHA_noBranch(t *testing.T) {
	repoDir, _, _ := setupDiffRepo(t)
	client := newTestGitClient()

	got, err := client.MergeBaseSHA(repoDir, "", "", "")
	if err != nil {
		t.Fatalf("MergeBaseSHA with empty inputs returned error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty result for no-branch skip, got %q", got)
	}
}

// setupCommitCloneRepo builds the same origin as setupMergeBaseRepo but mimics
// the cloneCommit + EnsureCommitPresent flow used for fork PRs: the clone is
// initialised without any branch tracking (no configured fetch refspec), and
// both the feature tip and the master tip are fetched by SHA into temporary
// refs that are immediately deleted — leaving both commits in .git/shallow with
// no named ref. This is the exact state that triggers the merge-base bug.
//
// Returns: cloneDir, forkPointSHA (C1 — expected merge-base),
// featureTipSHA, masterTipSHA, tipMinusOneSHA (the wrong "tip-1" answer).
func setupCommitCloneRepo(t *testing.T) (cloneDir, forkPointSHA, featureTipSHA, masterTipSHA, tipMinusOneSHA string) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not in PATH; skipping merge-base CLI test")
	}

	tmp := t.TempDir()
	originDir := filepath.Join(tmp, "origin")
	seedDir := filepath.Join(tmp, "seed")
	cloneDir = filepath.Join(tmp, "clone")

	run := func(dir string, args ...string) string {
		cmd := exec.Command("git", args...)
		if dir != "" {
			cmd.Dir = dir
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v (dir=%q): %v\n%s", args, dir, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	if err := os.MkdirAll(originDir, 0o755); err != nil {
		t.Fatalf("mkdir origin: %v", err)
	}
	run(originDir, "init", "--bare")

	if err := os.MkdirAll(seedDir, 0o755); err != nil {
		t.Fatalf("mkdir seed: %v", err)
	}
	run(seedDir, "init")
	run(seedDir, "symbolic-ref", "HEAD", "refs/heads/master")
	run(seedDir, "config", "user.email", "test@test.com")
	run(seedDir, "config", "user.name", "Test")
	run(seedDir, "remote", "add", "origin", "file://"+originDir)

	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(seedDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	write("file.txt", "c0\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "C0")

	write("file.txt", "c1\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "C1")
	forkPointSHA = run(seedDir, "rev-parse", "HEAD")

	// Feature branch diverges from C1.
	run(seedDir, "checkout", "-b", "feature")
	write("feature.txt", "A\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "A")
	tipMinusOneSHA = run(seedDir, "rev-parse", "HEAD") // A is the direct parent of the PR tip
	write("feature.txt", "B\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "B")
	featureTipSHA = run(seedDir, "rev-parse", "HEAD")
	run(seedDir, "push", "origin", "feature")

	// Advance master past C1.
	run(seedDir, "checkout", "master")
	write("file.txt", "c2\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "C2")
	write("file.txt", "c3\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "C3")
	run(seedDir, "push", "origin", "master")
	masterTipSHA = run(seedDir, "rev-parse", "HEAD")

	// Simulate cloneCommit (go-git CreateRemote): init + remote add, then immediately
	// unset the default fetch refspec that "git remote add" injects. go-git's
	// CreateRemote only stores a URL — no Fetch entries — so the local config has
	// no remote.origin.fetch. Without a fetch refspec, "git fetch --shallow-exclude"
	// has nothing to update and fails, reproducing the production fork-PR bug.
	if err := os.MkdirAll(cloneDir, 0o755); err != nil {
		t.Fatalf("mkdir clone: %v", err)
	}
	run(cloneDir, "init")
	run(cloneDir, "config", "user.email", "test@test.com")
	run(cloneDir, "config", "user.name", "Test")
	run(cloneDir, "remote", "add", "origin", "file://"+originDir)
	run(cloneDir, "config", "--unset", "remote.origin.fetch") // mirror go-git CreateRemote

	// cloneCommit fetches the PR tip into a tmp ref, checks out, then removes the ref.
	featureTmpRef := "refs/scanio/tmp/" + featureTipSHA
	run(cloneDir, "fetch", "--depth=1", "origin", featureTipSHA+":"+featureTmpRef)
	run(cloneDir, "checkout", "--detach", featureTipSHA)
	run(cloneDir, "update-ref", "-d", featureTmpRef)

	// EnsureCommitPresent fetches the base (master tip) into a tmp ref then removes it.
	masterTmpRef := "refs/scanio/tmp/" + masterTipSHA
	run(cloneDir, "fetch", "--depth=1", "origin", masterTipSHA+":"+masterTmpRef)
	run(cloneDir, "update-ref", "-d", masterTmpRef)

	return cloneDir, forkPointSHA, featureTipSHA, masterTipSHA, tipMinusOneSHA
}

// TestMergeBaseSHA_shallowCommitClone is a regression test for the fork-PR merge-base
// bug: when the repo was cloned by commit SHA (cloneCommit path, no fetch refspec),
// MergeBaseSHA must return the true fork point, not the immediate parent of the PR tip.
// Returning tip-1 would silently shrink the diff window and cause missed findings.
func TestMergeBaseSHA_shallowCommitClone(t *testing.T) {
	cloneDir, wantFork, headSHA, baseSHA, tipMinusOne := setupCommitCloneRepo(t)

	client := newTestGitClient()
	got, err := client.MergeBaseSHA(cloneDir, headSHA, "master", baseSHA)
	if err != nil {
		t.Fatalf("MergeBaseSHA returned unexpected error: %v", err)
	}
	// Guard against the specific tip-1 regression: a wrong baseline silently causes
	// missed findings in diff-aware scanning.
	if got == tipMinusOne {
		t.Errorf("merge base = tip-1 (%q): this causes missed findings; want fork point %q", got, wantFork)
	}
	if got != wantFork {
		t.Errorf("merge base = %q, want fork point %q", got, wantFork)
	}
}

// TestEnsureMergeBaseReachable verifies the go-git-based materialization path:
// given a commit-based shallow clone (no fetch refspec, both commits at depth=1
// with tmp refs removed), EnsureMergeBaseReachable must make the fork-point SHA
// reachable from HEAD so that change-aware scanners can use "git diff --merge-base <sha>".
func TestEnsureMergeBaseReachable(t *testing.T) {
	cloneDir, forkPointSHA, headSHA, _, _ := setupCommitCloneRepo(t)

	client := newTestGitClient()

	if err := EnsureMergeBaseReachable(client, cloneDir, headSHA, forkPointSHA); err != nil {
		t.Fatalf("EnsureMergeBaseReachable returned error: %v", err)
	}

	// Verify via the git CLI that a change-aware scanner's diff command would work:
	// git merge-base headSHA forkPointSHA must return forkPointSHA.
	cmd := exec.Command("git", "merge-base", headSHA, forkPointSHA)
	cmd.Dir = cloneDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git merge-base failed after EnsureMergeBaseReachable: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != forkPointSHA {
		t.Errorf("git merge-base = %q, want %q", got, forkPointSHA)
	}
}
