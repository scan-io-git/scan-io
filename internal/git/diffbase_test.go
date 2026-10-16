package git

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

const (
	configA = "timeout: 30\nold_token: ghp_OLDTOKEN1234567890abcdefghijklmnop12\nregion: eu\n"
	newKey  = "api_key: abc123NEWSECRETfromDEVELOPER"
)

// divergedRepo is the topology A-B-C-D on main with E-F on feature forked at B.
type divergedRepo struct {
	dir        string
	mergeBase  string // B
	featureTip string // F
	mainTip    string // D
}

type testGit struct {
	t   *testing.T
	dir string
}

func (g testGit) run(args ...string) string {
	g.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		g.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (g testGit) commit(files map[string]string, msg string) string {
	g.t.Helper()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(g.dir, name), []byte(content), 0o644); err != nil {
			g.t.Fatalf("write %s: %v", name, err)
		}
	}
	g.run("add", "-A")
	g.run("commit", "-m", msg)
	return g.run("rev-parse", "HEAD")
}

func newDivergedRepo(t *testing.T) divergedRepo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not in PATH")
	}
	g := testGit{t: t, dir: t.TempDir()}
	g.run("init", "-b", "main")

	g.commit(map[string]string{"config.yml": configA}, "A")
	r := divergedRepo{dir: g.dir}
	r.mergeBase = g.commit(map[string]string{"README.md": "# demo\n"}, "B")

	g.run("checkout", "-b", "feature")
	g.commit(map[string]string{"config.yml": configA + newKey + "\n"}, "E")
	r.featureTip = g.commit(map[string]string{"app.py": "print(\"hello\")\n"}, "F")

	g.run("checkout", "main")
	g.commit(map[string]string{"config.yml": strings.Replace(configA, "timeout: 30", "timeout: 60", 1)}, "C")
	r.mainTip = g.commit(map[string]string{
		"config.yml": "timeout: 60\nregion: eu\nregion_backup: us\n",
	}, "D")

	g.run("checkout", "feature")
	return r
}

func readDiffFile(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

func TestDiffLinesBase_MergeBaseExcludesTargetBranchChanges(t *testing.T) {
	r := newDivergedRepo(t)
	client := newTestGitClient()

	mb := client.ResolveMergeBase(r.dir, r.featureTip, "main", r.mainTip, nil)
	if mb != r.mergeBase {
		t.Fatalf("ResolveMergeBase = %q, want %q", mb, r.mergeBase)
	}

	tests := []struct {
		name        string
		mergeBase   string
		wantBase    string
		wantPresent []string
		wantAbsent  []string
	}{
		{
			name:        "merge base",
			mergeBase:   mb,
			wantBase:    r.mergeBase,
			wantPresent: []string{newKey},
			wantAbsent:  []string{"timeout: 30", "old_token"},
		},
		{
			name:        "target tip contrast",
			mergeBase:   "",
			wantBase:    r.mainTip,
			wantPresent: []string{newKey, "timeout: 30", "old_token"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DiffLinesBase(client, r.dir, tt.mergeBase, r.mainTip)
			if got != tt.wantBase {
				t.Fatalf("DiffLinesBase = %q, want %q", got, tt.wantBase)
			}

			root := filepath.Join(t.TempDir(), "diff")
			if err := MaterializeDiff(client, r.dir, root, got, r.featureTip, []string{"config.yml", "app.py"}); err != nil {
				t.Fatalf("MaterializeDiff: %v", err)
			}
			cfg := readDiffFile(t, root, "config.yml")
			for _, s := range tt.wantPresent {
				if !strings.Contains(cfg, s) {
					t.Errorf("config.yml diff missing %q:\n%s", s, cfg)
				}
			}
			for _, s := range tt.wantAbsent {
				if strings.Contains(cfg, s) {
					t.Errorf("config.yml diff unexpectedly contains %q:\n%s", s, cfg)
				}
			}
			if !strings.Contains(readDiffFile(t, root, "app.py"), `print("hello")`) {
				t.Errorf("app.py diff missing added line")
			}
		})
	}
}

func TestDiffLinesBase_MergeBaseKeepsLineNumbers(t *testing.T) {
	r := newDivergedRepo(t)
	client := newTestGitClient()

	got, err := AddedLines(client, r.dir, r.mergeBase, r.featureTip, nil)
	if err != nil {
		t.Fatalf("AddedLines: %v", err)
	}
	want := map[int]string{4: newKey}
	if diff := compareLineMaps(want, got["config.yml"]); diff != "" {
		t.Fatalf("config.yml added lines mismatch: %s", diff)
	}
}

func TestDiffLinesBase_FallsBackToTargetTip(t *testing.T) {
	r := newDivergedRepo(t)

	t.Run("empty merge base", func(t *testing.T) {
		var buf bytes.Buffer
		client := newBufferedGitClient(&buf)
		if got := DiffLinesBase(client, r.dir, "", r.mainTip); got != r.mainTip {
			t.Fatalf("DiffLinesBase = %q, want target tip %q", got, r.mainTip)
		}
		if !strings.Contains(buf.String(), "no merge base") || !strings.Contains(buf.String(), r.mainTip) {
			t.Errorf("expected info log naming the target tip, got %q", buf.String())
		}
	})

	t.Run("merge base not present and not fetchable", func(t *testing.T) {
		var buf bytes.Buffer
		client := newBufferedGitClient(&buf)
		bogus := strings.Repeat("a", 40)
		if got := DiffLinesBase(client, r.dir, bogus, r.mainTip); got != r.mainTip {
			t.Fatalf("DiffLinesBase = %q, want target tip %q", got, r.mainTip)
		}
		if !strings.Contains(buf.String(), "[WARN]") || !strings.Contains(buf.String(), bogus) {
			t.Errorf("expected warn log naming the merge base, got %q", buf.String())
		}
	})
}

func TestDiffLinesBase_UnrelatedHistories(t *testing.T) {
	r := newDivergedRepo(t)
	g := testGit{t: t, dir: r.dir}
	g.run("checkout", "--orphan", "unrelated")
	g.run("rm", "-rf", ".")
	orphanTip := g.commit(map[string]string{"solo.txt": "only\n"}, "orphan")

	client := newTestGitClient()
	if mb := client.ResolveMergeBase(r.dir, orphanTip, "main", r.mainTip, nil); mb != "" {
		t.Fatalf("ResolveMergeBase = %q, want empty for unrelated histories", mb)
	}
	base := DiffLinesBase(client, r.dir, "", r.mainTip)
	if base != r.mainTip {
		t.Fatalf("DiffLinesBase = %q, want target tip %q", base, r.mainTip)
	}

	root := filepath.Join(t.TempDir(), "diff")
	if err := MaterializeDiff(client, r.dir, root, base, orphanTip, []string{"solo.txt"}); err != nil {
		t.Fatalf("MaterializeDiff: %v", err)
	}
	if got := readDiffFile(t, root, "solo.txt"); got != "only\n" {
		t.Fatalf("solo.txt diff = %q, want %q", got, "only\n")
	}
}

func newBufferedGitClient(buf *bytes.Buffer) *Client {
	return &Client{
		logger:       hclog.New(&hclog.LoggerOptions{Output: buf, Level: hclog.Debug}),
		timeout:      time.Minute,
		globalConfig: &config.Config{},
	}
}
