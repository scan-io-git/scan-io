package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// cloneMethod selects how the local clone in buildMergeBaseRepo is produced.
type cloneMethod int

const (
	// cloneGitClone uses "git clone --depth=1 --branch=feature" (has a fetch refspec).
	cloneGitClone cloneMethod = iota
	// cloneInitDetach mimics cloneCommit: init + remote add with the fetch refspec
	// unset (go-git CreateRemote stores only a URL), then a depth-1 SHA fetch and
	// a detached checkout. Reproduces the fork-PR clone state.
	cloneInitDetach
)

// prefetch selects which commit is fetched at depth=1 into a temporary ref that
// is immediately deleted, mimicking EnsureCommitPresent: the commit object ends
// up in .git/shallow as a parentless root with no named ref.
type prefetch int

const (
	prefetchMasterTip prefetch = iota
	prefetchForkPoint
)

// mergeBaseRepoOpts parameterizes the repository topology built by buildMergeBaseRepo.
type mergeBaseRepoOpts struct {
	featureCommits int         // commits on feature after the fork point (>= 1)
	masterAdvance  int         // commits on master after the fork point (0 = head's parent IS the master tip)
	clone          cloneMethod // how the local clone is produced
	prefetch       []prefetch  // commits fetched at depth=1 then left ref-less in .git/shallow
	// headIsMerge makes the clone target a merge commit whose first parent is the
	// master tip (the base) and whose second parent is the feature tip.
	headIsMerge bool
}

// mergeBaseRepo holds the SHAs of the topology built by buildMergeBaseRepo.
type mergeBaseRepo struct {
	cloneDir      string
	forkPointSHA  string   // C1; == masterTipSHA when masterAdvance == 0
	featureTipSHA string   // last feature commit (the PR head)
	masterTipSHA  string   // master tip after masterAdvance commits (pre-merge; the merge's first parent)
	mergeHeadSHA  string   // merge commit head (set only when headIsMerge); parents [masterTip, featureTip]
	featureSHAs   []string // feature commits in order; featureSHAs[len-2] is tip-1
}

// buildMergeBaseRepo creates a bare origin with a master branch (C0, C1) and a
// feature branch diverging at C1, then produces a shallow local clone according
// to opts. All merge-base test topologies are expressed through this builder.
func buildMergeBaseRepo(t *testing.T, opts mergeBaseRepoOpts) mergeBaseRepo {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not in PATH; skipping merge-base CLI test")
	}

	tmp := t.TempDir()
	originDir := filepath.Join(tmp, "origin")
	seedDir := filepath.Join(tmp, "seed")
	cloneDir := filepath.Join(tmp, "clone")

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

	for _, dir := range []string{originDir, seedDir, cloneDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	run(originDir, "init", "--bare")

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
	commit := func(name, content, message string) string {
		write(name, content)
		run(seedDir, "add", ".")
		run(seedDir, "commit", "-m", message)
		return run(seedDir, "rev-parse", "HEAD")
	}

	commit("file.txt", "c0\n", "C0")
	repo := mergeBaseRepo{cloneDir: cloneDir}
	repo.forkPointSHA = commit("file.txt", "c1\n", "C1")

	// Feature branch diverges at the fork point.
	run(seedDir, "checkout", "-b", "feature")
	for i := 1; i <= opts.featureCommits; i++ {
		sha := commit("feature.txt", fmt.Sprintf("feature %d\n", i), fmt.Sprintf("F%d", i))
		repo.featureSHAs = append(repo.featureSHAs, sha)
	}
	repo.featureTipSHA = repo.featureSHAs[len(repo.featureSHAs)-1]
	run(seedDir, "push", "origin", "feature")

	// Master optionally advances past the fork point.
	run(seedDir, "checkout", "master")
	repo.masterTipSHA = repo.forkPointSHA
	for i := 1; i <= opts.masterAdvance; i++ {
		repo.masterTipSHA = commit("file.txt", fmt.Sprintf("c%d\n", i+1), fmt.Sprintf("C%d", i+1))
	}
	run(seedDir, "push", "origin", "master")

	// Optionally make the head a merge commit: merge feature into master so the
	// merge's first parent is the master tip (the base) and its second parent is
	// the feature tip. masterTipSHA is left pointing at the pre-merge tip (the
	// base / first parent) so prefetchMasterTip still fetches the base.
	cloneTarget := repo.featureTipSHA
	if opts.headIsMerge {
		run(seedDir, "merge", "--no-ff", "--no-edit", "feature")
		repo.mergeHeadSHA = run(seedDir, "rev-parse", "HEAD")
		run(seedDir, "push", "origin", "master")
		cloneTarget = repo.mergeHeadSHA
	}

	// Produce the local clone.
	switch opts.clone {
	case cloneGitClone:
		run("", "clone",
			"--depth=1", "--branch=feature", "--single-branch", "--no-tags",
			"file://"+originDir, cloneDir)
		run(cloneDir, "config", "user.email", "test@test.com")
		run(cloneDir, "config", "user.name", "Test")
	case cloneInitDetach:
		run(cloneDir, "init")
		run(cloneDir, "config", "user.email", "test@test.com")
		run(cloneDir, "config", "user.name", "Test")
		run(cloneDir, "remote", "add", "origin", "file://"+originDir)
		run(cloneDir, "config", "--unset", "remote.origin.fetch") // mirror go-git CreateRemote
		tipRef := "refs/scanio/tmp/" + cloneTarget
		run(cloneDir, "fetch", "--depth=1", "origin", cloneTarget+":"+tipRef)
		run(cloneDir, "checkout", "--detach", cloneTarget)
		run(cloneDir, "update-ref", "-d", tipRef)
	}

	// Simulate EnsureCommitPresent for the requested commits: fetch at depth=1
	// into a tmp ref then delete it, leaving the commit in .git/shallow with no
	// named ref (the production state before merge-base resolution runs).
	for _, p := range opts.prefetch {
		sha := repo.masterTipSHA
		if p == prefetchForkPoint {
			sha = repo.forkPointSHA
		}
		ref := "refs/scanio/tmp/" + sha
		run(cloneDir, "fetch", "--depth=1", "origin", sha+":"+ref)
		run(cloneDir, "update-ref", "-d", ref)
	}

	return repo
}

// setupMergeBaseRepo: branch-based shallow clone (git clone) with the master tip
// pre-fetched — the non-fork PR topology.
func setupMergeBaseRepo(t *testing.T) (cloneDir, forkPointSHA, featureTipSHA, masterTipSHA string) {
	r := buildMergeBaseRepo(t, mergeBaseRepoOpts{
		featureCommits: 2,
		masterAdvance:  2,
		clone:          cloneGitClone,
		prefetch:       []prefetch{prefetchMasterTip},
	})
	return r.cloneDir, r.forkPointSHA, r.featureTipSHA, r.masterTipSHA
}

// setupCommitCloneRepo: commit-based shallow clone (no fetch refspec) with the
// master tip pre-fetched — the fork-PR topology that triggered the original bug.
func setupCommitCloneRepo(t *testing.T) (cloneDir, forkPointSHA, featureTipSHA, masterTipSHA, tipMinusOneSHA string) {
	r := buildMergeBaseRepo(t, mergeBaseRepoOpts{
		featureCommits: 2,
		masterAdvance:  2,
		clone:          cloneInitDetach,
		prefetch:       []prefetch{prefetchMasterTip},
	})
	return r.cloneDir, r.forkPointSHA, r.featureTipSHA, r.masterTipSHA, r.featureSHAs[len(r.featureSHAs)-2]
}

// setupDeepCommitCloneRepo: the feature branch has 15 commits so the merge-base
// lies beyond the first deepening step (depth=10), with the fork point itself
// pre-fetched at depth=1.
func setupDeepCommitCloneRepo(t *testing.T) (cloneDir, forkPointSHA, featureTipSHA string) {
	r := buildMergeBaseRepo(t, mergeBaseRepoOpts{
		featureCommits: 15,
		masterAdvance:  1,
		clone:          cloneInitDetach,
		prefetch:       []prefetch{prefetchForkPoint},
	})
	return r.cloneDir, r.forkPointSHA, r.featureTipSHA
}

// setupDirectParentCloneRepo: a single PR commit directly on top of the target
// tip, so head's parent IS the merge-base. Both commits end up as parentless
// roots in .git/shallow — the lying-short-circuit topology.
func setupDirectParentCloneRepo(t *testing.T) (cloneDir, mergeBaseSHA, headSHA string) {
	r := buildMergeBaseRepo(t, mergeBaseRepoOpts{
		featureCommits: 1,
		masterAdvance:  0,
		clone:          cloneInitDetach,
		prefetch:       []prefetch{prefetchMasterTip},
	})
	return r.cloneDir, r.forkPointSHA, r.featureTipSHA
}

// setupMergeHeadCloneRepo: the PR head is a merge commit whose first parent is
// the target tip (the base) and whose second parent is the feature tip. Both the
// merge head and the base end up as parentless roots in .git/shallow, but the
// head's second parent is absent — the topology where go-git's IsAncestor lies
// (it finds the base via the present first parent) while the git CLI fails.
func setupMergeHeadCloneRepo(t *testing.T) (cloneDir, baseSHA, headSHA string) {
	r := buildMergeBaseRepo(t, mergeBaseRepoOpts{
		featureCommits: 2,
		masterAdvance:  1,
		clone:          cloneInitDetach,
		prefetch:       []prefetch{prefetchMasterTip},
		headIsMerge:    true,
	})
	return r.cloneDir, r.masterTipSHA, r.mergeHeadSHA
}

func TestMergeBaseSHA_shallow(t *testing.T) {
	cloneDir, wantFork, headSHA, baseSHA := setupMergeBaseRepo(t)

	client := newTestGitClient()
	got, err := client.mergeBaseSHA(cloneDir, headSHA, "master", baseSHA)
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

	got, err := client.mergeBaseSHA(repoDir, "", "", "")
	if err != nil {
		t.Fatalf("MergeBaseSHA with empty inputs returned error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty result for no-branch skip, got %q", got)
	}
}

// TestMergeBaseSHA_shallowCommitClone is a regression test for the fork-PR merge-base
// bug: when the repo was cloned by commit SHA (cloneCommit path, no fetch refspec),
// mergeBaseSHA must return the true fork point, not the immediate parent of the PR tip.
// Returning tip-1 would silently shrink the diff window and cause missed findings.
func TestMergeBaseSHA_shallowCommitClone(t *testing.T) {
	cloneDir, wantFork, headSHA, baseSHA, tipMinusOne := setupCommitCloneRepo(t)

	client := newTestGitClient()
	got, err := client.mergeBaseSHA(cloneDir, headSHA, "master", baseSHA)
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
// with tmp refs removed), ensureMergeBaseReachable must make the fork-point SHA
// reachable from HEAD so that change-aware scanners can use "git diff --merge-base <sha>".
func TestEnsureMergeBaseReachable(t *testing.T) {
	cloneDir, forkPointSHA, headSHA, _, _ := setupCommitCloneRepo(t)

	client := newTestGitClient()

	if err := client.ensureMergeBaseReachable(cloneDir, headSHA, forkPointSHA); err != nil {
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

// TestEnsureMergeBaseReachable_deepBranch is the regression test for the
// shallow-boundary premature-termination bug: when the merge-base is more
// than 10 commits from HEAD, go-git's IsAncestor returns (false, nil) at
// the depth-10 shallow boundary. The deepening loop must not treat this as
// "definitively not an ancestor" — it must continue to depth=50.
func TestEnsureMergeBaseReachable_deepBranch(t *testing.T) {
	cloneDir, forkPointSHA, headSHA := setupDeepCommitCloneRepo(t)

	client := newTestGitClient()
	if err := client.ensureMergeBaseReachable(cloneDir, headSHA, forkPointSHA); err != nil {
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

// TestEnsureMergeBaseReachable_directParent is the regression test for the
// lying-short-circuit bug: when the merge-base is the PR head's direct parent
// and both objects are present from depth-1 fetches, go-git's IsAncestor
// (which ignores .git/shallow) reports reachable and ensureMergeBaseReachable
// returns success — but the git CLI (which respects .git/shallow) still fails
// with "no merge base found" because head is listed as a parentless root.
// ensureMergeBaseReachable must leave the repo in a state where the git CLI
// agrees, since that is what change-aware scanners actually invoke.
func TestEnsureMergeBaseReachable_directParent(t *testing.T) {
	cloneDir, mergeBaseSHA, headSHA := setupDirectParentCloneRepo(t)

	client := newTestGitClient()
	if err := client.ensureMergeBaseReachable(cloneDir, headSHA, mergeBaseSHA); err != nil {
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

// TestEnsureMergeBaseReachable_mergeCommitHead is the regression test for the
// merge-commit-head false positive: when the PR head is a merge commit whose
// first parent is the base (present) but whose second parent is absent,
// cleanShallowEntries cannot unshallow the head, so go-git's IsAncestor reports
// reachable via the present first parent while the git CLI still fails with
// "no merge base found". ensureMergeBaseReachable must deepen until the git CLI
// — the oracle change-aware scanners use — agrees.
func TestEnsureMergeBaseReachable_mergeCommitHead(t *testing.T) {
	cloneDir, baseSHA, headSHA := setupMergeHeadCloneRepo(t)

	client := newTestGitClient()
	if err := client.ensureMergeBaseReachable(cloneDir, headSHA, baseSHA); err != nil {
		t.Fatalf("ensureMergeBaseReachable returned error: %v", err)
	}

	// The verdict comes from the git CLI, not go-git.
	cmd := exec.Command("git", "merge-base", headSHA, baseSHA)
	cmd.Dir = cloneDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git merge-base failed after ensureMergeBaseReachable reported success: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != baseSHA {
		t.Errorf("git merge-base = %q, want %q", got, baseSHA)
	}
}

// TestResolveMergeBase_mergeCommitHead: with a merge-commit head, the API returns
// the base SHA; ResolveMergeBase must return it only after making it CLI-usable.
func TestResolveMergeBase_mergeCommitHead(t *testing.T) {
	cloneDir, baseSHA, headSHA := setupMergeHeadCloneRepo(t)

	client := newTestGitClient()
	api := func() (string, error) { return baseSHA, nil }
	got := client.ResolveMergeBase(cloneDir, headSHA, "master", baseSHA, api)
	if got != baseSHA {
		t.Fatalf("ResolveMergeBase = %q, want %q", got, baseSHA)
	}
	out, err := exec.Command("git", "-C", cloneDir, "merge-base", headSHA, baseSHA).Output()
	if err != nil || strings.TrimSpace(string(out)) != baseSHA {
		t.Errorf("git merge-base after ResolveMergeBase: out=%q err=%v", out, err)
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

// TestResolveMergeBase_emptyTreeAPI: when the provider returns git's empty-tree
// hash (no merge base — initial commit / unrelated histories), ResolveMergeBase
// returns "" immediately, skipping materialization and the git fallback. The
// clone has a real fork point, so a non-empty result would mean the short-circuit
// failed and the git fallback computed one anyway.
func TestResolveMergeBase_emptyTreeAPI(t *testing.T) {
	cloneDir, _, headSHA, baseSHA, _ := setupCommitCloneRepo(t)
	client := newTestGitClient()
	api := func() (string, error) { return emptyTreeSHA, nil }
	if got := client.ResolveMergeBase(cloneDir, headSHA, "master", baseSHA, api); got != "" {
		t.Errorf("ResolveMergeBase with empty-tree API = %q, want \"\" (no baseline)", got)
	}
}
