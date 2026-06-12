package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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

// setupDeepCommitCloneRepo is like setupCommitCloneRepo but the feature branch
// has 15 commits from the fork point, so the merge-base lies beyond depth=10.
// Returns: cloneDir, forkPointSHA, featureTipSHA.
func setupDeepCommitCloneRepo(t *testing.T) (cloneDir, forkPointSHA, featureTipSHA string) {
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

	write("file.txt", "base\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "C0")

	write("file.txt", "c1\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "C1")
	forkPointSHA = run(seedDir, "rev-parse", "HEAD")

	// Feature branch with 15 commits beyond the fork point (merge-base at depth 16).
	run(seedDir, "checkout", "-b", "feature")
	for i := 1; i <= 15; i++ {
		write("feat.txt", fmt.Sprintf("feature commit %d\n", i))
		run(seedDir, "add", ".")
		run(seedDir, "commit", "-m", fmt.Sprintf("F%d", i))
	}
	featureTipSHA = run(seedDir, "rev-parse", "HEAD")
	run(seedDir, "push", "origin", "feature")

	run(seedDir, "checkout", "master")
	write("file.txt", "c2\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "C2")
	run(seedDir, "push", "origin", "master")

	// Simulate cloneCommit: no fetch refspec, featureTip at depth=1.
	if err := os.MkdirAll(cloneDir, 0o755); err != nil {
		t.Fatalf("mkdir clone: %v", err)
	}
	run(cloneDir, "init")
	run(cloneDir, "config", "user.email", "test@test.com")
	run(cloneDir, "config", "user.name", "Test")
	run(cloneDir, "remote", "add", "origin", "file://"+originDir)
	run(cloneDir, "config", "--unset", "remote.origin.fetch")

	featureTmpRef := "refs/scanio/tmp/" + featureTipSHA
	run(cloneDir, "fetch", "--depth=1", "origin", featureTipSHA+":"+featureTmpRef)
	run(cloneDir, "checkout", "--detach", featureTipSHA)
	run(cloneDir, "update-ref", "-d", featureTmpRef)

	// Simulate EnsureCommitPresent(forkPointSHA): pre-fetch the fork point at depth=1
	// and remove its tmp ref. This mirrors what the Bitbucket plugin does when
	// baseSHA == apiMergeBase (PR branched from current target tip): the merge-base is
	// in the object store as a depth-1 shallow root before EnsureMergeBaseReachable runs.
	// Without this step, isMergeBaseReachable returns (false, ErrObjectNotFound) at
	// depth=10 (object missing) rather than (false, nil) (object present but unreachable
	// via shallow walk) — so the pre-fetch is needed to hit the true bug path.
	forkTmpRef := "refs/scanio/tmp/" + forkPointSHA
	run(cloneDir, "fetch", "--depth=1", "origin", forkPointSHA+":"+forkTmpRef)
	run(cloneDir, "update-ref", "-d", forkTmpRef)

	return cloneDir, forkPointSHA, featureTipSHA
}

// TestEnsureMergeBaseReachable_deepBranch is the regression test for the
// shallow-boundary premature-termination bug: when the merge-base is more
// than 10 commits from HEAD, go-git's IsAncestor returns (false, nil) at
// the depth-10 shallow boundary. The deepening loop must not treat this as
// "definitively not an ancestor" — it must continue to depth=50.
func TestEnsureMergeBaseReachable_deepBranch(t *testing.T) {
	cloneDir, forkPointSHA, headSHA := setupDeepCommitCloneRepo(t)

	client := newTestGitClient()
	if err := EnsureMergeBaseReachable(client, cloneDir, headSHA, forkPointSHA); err != nil {
		t.Fatalf("EnsureMergeBaseReachable returned error for 15-commit branch: %v", err)
	}

	// Confirm via git CLI — this is what change-aware scanners use.
	cmd := exec.Command("git", "merge-base", headSHA, forkPointSHA)
	cmd.Dir = cloneDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git merge-base failed after EnsureMergeBaseReachable: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != forkPointSHA {
		t.Errorf("git merge-base = %q, want %q", got, forkPointSHA)
	}
}

// setupDirectParentCloneRepo reproduces the topology where the PR head sits
// directly on top of the target tip: head's parent IS the merge-base. Both
// commits are fetched at depth=1 (cloneCommit + EnsureCommitPresent), so both
// commit objects exist in the store while .git/shallow lists both as parentless
// roots. go-git's IsAncestor ignores .git/shallow and sees head→parent→mb, but
// the git CLI respects .git/shallow and reports "no merge base found".
// Returns: cloneDir, mergeBaseSHA (= head's parent = target tip), headSHA.
func setupDirectParentCloneRepo(t *testing.T) (cloneDir, mergeBaseSHA, headSHA string) {
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

	// Target tip = merge-base: the PR branches from the current tip.
	write("file.txt", "c1\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "C1")
	mergeBaseSHA = run(seedDir, "rev-parse", "HEAD")
	run(seedDir, "push", "origin", "master")

	// Single PR commit directly on top of the target tip.
	run(seedDir, "checkout", "-b", "feature")
	write("feature.txt", "F1\n")
	run(seedDir, "add", ".")
	run(seedDir, "commit", "-m", "F1")
	headSHA = run(seedDir, "rev-parse", "HEAD")
	run(seedDir, "push", "origin", "feature")

	// Simulate cloneCommit: no fetch refspec, head at depth=1.
	if err := os.MkdirAll(cloneDir, 0o755); err != nil {
		t.Fatalf("mkdir clone: %v", err)
	}
	run(cloneDir, "init")
	run(cloneDir, "config", "user.email", "test@test.com")
	run(cloneDir, "config", "user.name", "Test")
	run(cloneDir, "remote", "add", "origin", "file://"+originDir)
	run(cloneDir, "config", "--unset", "remote.origin.fetch")

	headTmpRef := "refs/scanio/tmp/" + headSHA
	run(cloneDir, "fetch", "--depth=1", "origin", headSHA+":"+headTmpRef)
	run(cloneDir, "checkout", "--detach", headSHA)
	run(cloneDir, "update-ref", "-d", headTmpRef)

	// Simulate EnsureCommitPresent(baseSHA): fetch the merge-base at depth=1.
	// Now both commit OBJECTS are in the store, but .git/shallow lists both
	// as parentless roots — the exact lying-short-circuit precondition.
	mbTmpRef := "refs/scanio/tmp/" + mergeBaseSHA
	run(cloneDir, "fetch", "--depth=1", "origin", mergeBaseSHA+":"+mbTmpRef)
	run(cloneDir, "update-ref", "-d", mbTmpRef)

	return cloneDir, mergeBaseSHA, headSHA
}

// TestEnsureMergeBaseReachable_directParent is the regression test for the
// lying-short-circuit bug: when the merge-base is the PR head's direct parent
// and both objects are present from depth-1 fetches, go-git's IsAncestor
// (which ignores .git/shallow) reports reachable and EnsureMergeBaseReachable
// returns success — but the git CLI (which respects .git/shallow) still fails
// with "no merge base found" because head is listed as a parentless root.
// EnsureMergeBaseReachable must leave the repo in a state where the git CLI
// agrees, since that is what change-aware scanners actually invoke.
func TestEnsureMergeBaseReachable_directParent(t *testing.T) {
	cloneDir, mergeBaseSHA, headSHA := setupDirectParentCloneRepo(t)

	client := newTestGitClient()
	if err := EnsureMergeBaseReachable(client, cloneDir, headSHA, mergeBaseSHA); err != nil {
		t.Fatalf("EnsureMergeBaseReachable returned error: %v", err)
	}

	// The verdict comes from the git CLI, not go-git.
	cmd := exec.Command("git", "merge-base", headSHA, mergeBaseSHA)
	cmd.Dir = cloneDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git merge-base failed after EnsureMergeBaseReachable reported success: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != mergeBaseSHA {
		t.Errorf("git merge-base = %q, want %q", got, mergeBaseSHA)
	}
}

// TestResolveMergeBase_apiPath: API returns the known fork point → result == apiMB
// and "git merge-base head mb" succeeds in the clone (materialized).
func TestResolveMergeBase_apiPath(t *testing.T) {
	cloneDir, forkPoint, headSHA, baseSHA, _ := setupCommitCloneRepo(t)
	client := newTestGitClient()
	api := func() (string, error) { return forkPoint, nil }
	got := client.ResolveMergeBase(cloneDir, headSHA, "master", baseSHA, api)
	if got != forkPoint {
		t.Fatalf("ResolveMergeBase = %q, want %q", got, forkPoint)
	}
	out, err := exec.Command("git", "-C", cloneDir, "merge-base", headSHA, forkPoint).Output()
	if err != nil || strings.TrimSpace(string(out)) != forkPoint {
		t.Errorf("git merge-base after ResolveMergeBase: out=%q err=%v", out, err)
	}
}

// TestResolveMergeBase_apiFails: API errors → falls back to git computation,
// still returns the fork point.
func TestResolveMergeBase_apiFails(t *testing.T) {
	cloneDir, forkPoint, headSHA, baseSHA, _ := setupCommitCloneRepo(t)
	client := newTestGitClient()
	api := func() (string, error) { return "", fmt.Errorf("api down") }
	if got := client.ResolveMergeBase(cloneDir, headSHA, "master", baseSHA, api); got != forkPoint {
		t.Errorf("ResolveMergeBase fallback = %q, want %q", got, forkPoint)
	}
}
