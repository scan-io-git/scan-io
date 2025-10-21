package git

import (
	"fmt"
	"os"
	"path/filepath"
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
