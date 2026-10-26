package files

import (
	"bytes"
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
