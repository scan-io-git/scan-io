package files

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestDetermineFileFullPath(t *testing.T) {
	type testCase struct {
		name         string
		inputPath    string
		nameTemplate string
		expectFile   string
		expectFolder string
		setup        func(t *testing.T) (inputPath, expectFile, expectFolder string)
	}

	tmpDir := t.TempDir()

	tests := []testCase{
		{
			name:         "Directory path with name template",
			inputPath:    tmpDir,
			nameTemplate: "output.json",
			expectFile:   filepath.Join(tmpDir, "output.json"),
			expectFolder: tmpDir,
		},
		{
			name:         "File path with extension",
			inputPath:    filepath.Join(tmpDir, "data.json"),
			nameTemplate: "ignored.txt",
			expectFile:   filepath.Join(tmpDir, "data.json"),
			expectFolder: tmpDir,
			setup: func(t *testing.T) (string, string, string) {
				f := filepath.Join(tmpDir, "data.json")
				_ = os.WriteFile(f, []byte("test"), 0644)
				return f, f, tmpDir
			},
		},
		{
			name:         "Path with no extension, treat as folder",
			inputPath:    filepath.Join(tmpDir, "output_folder"),
			nameTemplate: "report.log",
			expectFile:   filepath.Join(tmpDir, "output_folder", "report.log"),
			expectFolder: filepath.Join(tmpDir, "output_folder"),
		},
		{
			name:         "Non-existent file with extension",
			inputPath:    filepath.Join(tmpDir, "nonexistent.yaml"),
			nameTemplate: "ignored.txt",
			expectFile:   filepath.Join(tmpDir, "nonexistent.yaml"),
			expectFolder: tmpDir,
		},
		{
			name:         "Non-existent folder",
			inputPath:    filepath.Join(tmpDir, "missing_folder"),
			nameTemplate: "result.json",
			expectFile:   filepath.Join(tmpDir, "missing_folder", "result.json"),
			expectFolder: filepath.Join(tmpDir, "missing_folder"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualPath := tt.inputPath
			expectFile := tt.expectFile
			expectFolder := tt.expectFolder

			if tt.setup != nil {
				actualPath, expectFile, expectFolder = tt.setup(t)
			}

			filePath, folderPath, err := DetermineFileFullPath(actualPath, tt.nameTemplate)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if filePath != expectFile {
				t.Errorf("Expected file path %s, got %s", expectFile, filePath)
			}
			if folderPath != expectFolder {
				t.Errorf("Expected folder path %s, got %s", expectFolder, folderPath)
			}
		})
	}
}

func TestCopyDotFilesSymlinks(t *testing.T) {
	// "{outside}" in a link target is replaced with the path of a file that
	// exists next to the source tree, outside both the source and destination.
	tests := []struct {
		name        string
		links       map[string]string
		wantLinks   []string
		wantSkipped []string
	}{
		{
			name: "regular files and nested dirs without symlinks",
		},
		{
			name:      "relative target inside the tree is copied",
			links:     map[string]string{".agents/ref.md": "notes/real.md"},
			wantLinks: []string{".agents/ref.md"},
		},
		{
			name:        "absolute target outside the tree is skipped",
			links:       map[string]string{".agents/ref.md": "{outside}"},
			wantSkipped: []string{".agents/ref.md"},
		},
		{
			name:        "relative target escaping the tree is skipped",
			links:       map[string]string{".agents/ref.md": "../../outside.txt"},
			wantSkipped: []string{".agents/ref.md"},
		},
		{
			name:      "dangling target inside the tree is copied as a dangling link",
			links:     map[string]string{".agents/ref.md": "missing.md"},
			wantLinks: []string{".agents/ref.md"},
		},
		{
			name: "target escaping through another symlink is skipped",
			links: map[string]string{
				".agents/ref.md": "up/../..",
				".agents/up":     "..",
			},
			wantLinks:   []string{".agents/up"},
			wantSkipped: []string{".agents/ref.md"},
		},
		{
			name: "dangling target that could escape once a missing dir appears is skipped",
			links: map[string]string{
				".agents/ref.md": "missing/../up/../outside.txt",
				".agents/up":     "..",
			},
			wantLinks:   []string{".agents/up"},
			wantSkipped: []string{".agents/ref.md"},
		},
		{
			name:        "target stepping back up after a name is skipped",
			links:       map[string]string{".agents/ref.md": "notes/../z-after.txt"},
			wantSkipped: []string{".agents/ref.md"},
		},
		{
			name:      "leading parent steps inside the tree are copied",
			links:     map[string]string{".agents/notes/ref.md": "../z-after.txt"},
			wantLinks: []string{".agents/notes/ref.md"},
		},
		{
			name:      "dot-slash prefix is copied",
			links:     map[string]string{".agents/ref.md": "./notes/real.md"},
			wantLinks: []string{".agents/ref.md"},
		},
		{
			name:        "escaping symlink in a top-level dot entry is skipped",
			links:       map[string]string{".escape": "{outside}"},
			wantSkipped: []string{".escape"},
		},
	}

	regular := map[string]string{
		".agents/notes/real.md":        "real",
		".agents/z-after.txt":          "after",
		".config/nested/deep/file.txt": "deep",
		".env":                         "env",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			src := filepath.Join(base, "repo")
			dst := filepath.Join(base, "dst")
			outside := filepath.Join(base, "outside.txt")
			writeTestFile(t, outside, "outside")
			writeTestFile(t, filepath.Join(src, "main.go"), "package main")
			for rel, content := range regular {
				writeTestFile(t, filepath.Join(src, rel), content)
			}
			for link, target := range tt.links {
				target = strings.ReplaceAll(target, "{outside}", outside)
				tt.links[link] = target
				if err := os.Symlink(target, filepath.Join(src, link)); err != nil {
					t.Fatalf("Symlink(%q): %v", link, err)
				}
			}

			var logs bytes.Buffer
			logger := hclog.New(&hclog.LoggerOptions{Output: &logs, Level: hclog.Warn})
			if err := CopyDotFiles(src, dst, logger); err != nil {
				t.Fatalf("CopyDotFiles returned error: %v", err)
			}

			for rel, want := range regular {
				got, err := os.ReadFile(filepath.Join(dst, rel))
				if err != nil {
					t.Errorf("expected %q to be copied: %v", rel, err)
				} else if string(got) != want {
					t.Errorf("%q content = %q, want %q", rel, got, want)
				}
			}
			if _, err := os.Lstat(filepath.Join(dst, "main.go")); !os.IsNotExist(err) {
				t.Errorf("expected non-dot entry main.go to be left out, got err=%v", err)
			}
			for _, link := range tt.wantLinks {
				got, err := os.Readlink(filepath.Join(dst, link))
				if err != nil {
					t.Errorf("expected symlink %q to be copied: %v", link, err)
				} else if got != tt.links[link] {
					t.Errorf("symlink %q -> %q, want %q", link, got, tt.links[link])
				}
			}
			for _, link := range tt.wantSkipped {
				if _, err := os.Lstat(filepath.Join(dst, link)); !os.IsNotExist(err) {
					t.Errorf("expected symlink %q to be skipped, got err=%v", link, err)
				}
				if !strings.Contains(logs.String(), "link="+link) {
					t.Errorf("expected a warning for skipped symlink %q, logs:\n%s", link, logs.String())
				}
			}
			if got := strings.Count(logs.String(), "[WARN]"); got != len(tt.wantSkipped) {
				t.Errorf("got %d warnings, want %d, logs:\n%s", got, len(tt.wantSkipped), logs.String())
			}
		})
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func TestRemoveUnsafeSymlinks(t *testing.T) {
	// "{outside}" in a link target is replaced with the path of a file that
	// exists next to the tree, outside it.
	tests := []struct {
		name        string
		links       map[string]string
		wantKept    []string
		wantRemoved []string
	}{
		{
			name:     "relative target inside the tree is kept",
			links:    map[string]string{"ref.md": "notes/real.md"},
			wantKept: []string{"ref.md"},
		},
		{
			name:        "absolute target outside the tree is removed",
			links:       map[string]string{"ref.md": "{outside}"},
			wantRemoved: []string{"ref.md"},
		},
		{
			name:        "relative target escaping the tree is removed",
			links:       map[string]string{"ref.md": "../../outside.txt"},
			wantRemoved: []string{"ref.md"},
		},
		{
			name:        "target stepping up after a name is removed",
			links:       map[string]string{"ref.md": "notes/../real.md"},
			wantRemoved: []string{"ref.md"},
		},
		{
			name:        "target escaping through another symlink is removed",
			links:       map[string]string{"sub/ref.md": "up/../..", "sub/up": ".."},
			wantKept:    []string{"sub/up"},
			wantRemoved: []string{"sub/ref.md"},
		},
		{
			name:     "dangling target inside the tree is kept",
			links:    map[string]string{"ref.md": "missing.md"},
			wantKept: []string{"ref.md"},
		},
		{
			name:     "target inside .git is kept",
			links:    map[string]string{"notes.txt": ".git/config"},
			wantKept: []string{"notes.txt"},
		},
		{
			name:        "unsafe symlink inside .git is removed",
			links:       map[string]string{".git/evil": "{outside}"},
			wantRemoved: []string{".git/evil"},
		},
		{
			name: "directory symlink is kept and not descended into",
			links: map[string]string{
				"dirlink":         "notes",
				"notes/escape.md": "{outside}",
			},
			wantKept:    []string{"dirlink"},
			wantRemoved: []string{"notes/escape.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			outside := filepath.Join(base, "outside.txt")
			writeTestFile(t, outside, "secret")
			root := filepath.Join(base, "tree")
			regular := map[string]string{
				"real.md":       "real",
				"notes/real.md": "notes",
				".git/config":   "cfg",
			}
			for rel, content := range regular {
				writeTestFile(t, filepath.Join(root, rel), content)
			}
			targets := map[string]string{}
			for link, target := range tt.links {
				target = strings.ReplaceAll(target, "{outside}", outside)
				targets[link] = target
			}
			// Links are created after all directories exist so order does not matter.
			for link, target := range targets {
				if err := os.MkdirAll(filepath.Dir(filepath.Join(root, link)), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(root, link)); err != nil {
					t.Fatal(err)
				}
			}

			var logs bytes.Buffer
			logger := hclog.New(&hclog.LoggerOptions{Output: &logs, Level: hclog.Debug})
			if err := RemoveUnsafeSymlinks(root, logger); err != nil {
				t.Fatalf("RemoveUnsafeSymlinks() error = %v", err)
			}

			for _, link := range tt.wantKept {
				got, err := os.Readlink(filepath.Join(root, link))
				if err != nil {
					t.Errorf("link %q should be kept: %v", link, err)
				} else if got != targets[link] {
					t.Errorf("link %q target = %q, want %q", link, got, targets[link])
				}
			}
			for _, link := range tt.wantRemoved {
				if _, err := os.Lstat(filepath.Join(root, link)); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("link %q should be removed, Lstat error = %v", link, err)
				}
			}
			for rel, content := range regular {
				data, err := os.ReadFile(filepath.Join(root, rel))
				if err != nil || string(data) != content {
					t.Errorf("regular file %q changed: %q, %v", rel, data, err)
				}
			}
			if data, err := os.ReadFile(outside); err != nil || string(data) != "secret" {
				t.Errorf("file outside the tree changed: %q, %v", data, err)
			}
			if got := strings.Count(logs.String(), "[WARN]"); got != len(tt.wantRemoved) {
				t.Errorf("WARN lines = %d, want %d\n%s", got, len(tt.wantRemoved), logs.String())
			}
			for _, link := range tt.wantRemoved {
				if !strings.Contains(logs.String(), "link="+link) {
					t.Errorf("no warning for %q\n%s", link, logs.String())
				}
			}
		})
	}
}

func TestRemoveUnsafeSymlinksMissingRoot(t *testing.T) {
	err := RemoveUnsafeSymlinks(filepath.Join(t.TempDir(), "missing"), hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error for a missing root")
	}
}

func TestRemoveUnsafeSymlinksRemovalFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("directory permissions do not apply to root")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "locked")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/absolute/outside", filepath.Join(dir, "evil")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	if err := RemoveUnsafeSymlinks(root, hclog.NewNullLogger()); err == nil {
		t.Fatal("expected an error when the link cannot be removed")
	}
}

func TestOpenInRootPath(t *testing.T) {
	base := t.TempDir()
	realRoot := filepath.Join(base, "real")
	outside := filepath.Join(base, "outside")
	for _, d := range []string{realRoot, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(realRoot, "inside.txt"), "inside")
	write(filepath.Join(outside, "outside.txt"), "outside")
	link := func(target, name string) {
		t.Helper()
		if err := os.Symlink(target, filepath.Join(realRoot, name)); err != nil {
			t.Fatal(err)
		}
	}
	link(filepath.Join(outside, "outside.txt"), "abs-out.txt")
	link("inside.txt", "to-inside.txt")
	rootLink := filepath.Join(base, "rootlink")
	if err := os.Symlink(realRoot, rootLink); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		root        string
		path        string
		want        string
		wantOutside bool
		wantErr     bool
	}{
		{name: "relative inside", root: realRoot, path: "inside.txt", want: "inside"},
		{name: "absolute inside", root: realRoot, path: filepath.Join(realRoot, "inside.txt"), want: "inside"},
		{name: "absolute outside", root: realRoot, path: filepath.Join(outside, "outside.txt"), wantOutside: true},
		{name: "parent escape", root: realRoot, path: "../outside/outside.txt", wantOutside: true},
		{name: "symlink to absolute outside file", root: realRoot, path: "abs-out.txt", wantOutside: true},
		{name: "symlink to inside file", root: realRoot, path: "to-inside.txt", want: "inside"},
		{name: "root through symlink, relative", root: rootLink, path: "inside.txt", want: "inside"},
		{name: "root through symlink, absolute via real root", root: rootLink, path: filepath.Join(realRoot, "inside.txt"), want: "inside"},
		{name: "root through symlink, absolute via link", root: rootLink, path: filepath.Join(rootLink, "inside.txt"), want: "inside"},
		{name: "missing file is not an outside error", root: realRoot, path: "missing.txt", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := OpenInRootPath(tt.root, tt.path)
			if tt.wantOutside || tt.wantErr {
				if err == nil {
					f.Close()
					t.Fatal("expected error, got nil")
				}
				if got := errors.Is(err, ErrOutsideRoot); got != tt.wantOutside {
					t.Fatalf("errors.Is(err, ErrOutsideRoot) = %v, want %v (err: %v)", got, tt.wantOutside, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer f.Close()
			data, err := io.ReadAll(f)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tt.want {
				t.Fatalf("content = %q, want %q", data, tt.want)
			}
		})
	}
}
