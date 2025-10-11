package sarif

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scan-io-git/scan-io/internal/git"
)

func TestNormalisedSubfolder(t *testing.T) {
	tests := []struct {
		name     string
		metadata *git.RepositoryMetadata
		expected string
	}{
		{
			name:     "nil metadata",
			metadata: nil,
			expected: "",
		},
		{
			name: "empty subfolder",
			metadata: &git.RepositoryMetadata{
				Subfolder: "",
			},
			expected: "",
		},
		{
			name: "subfolder with forward slash",
			metadata: &git.RepositoryMetadata{
				Subfolder: "apps/demo",
			},
			expected: "apps/demo",
		},
		{
			name: "subfolder with leading slash",
			metadata: &git.RepositoryMetadata{
				Subfolder: "/apps/demo",
			},
			expected: "apps/demo",
		},
		{
			name: "subfolder with trailing slash",
			metadata: &git.RepositoryMetadata{
				Subfolder: "apps/demo/",
			},
			expected: "apps/demo",
		},
		{
			name: "subfolder with backslash",
			metadata: &git.RepositoryMetadata{
				Subfolder: "apps\\demo",
			},
			expected: "apps/demo",
		},
		{
			name: "subfolder with both slashes",
			metadata: &git.RepositoryMetadata{
				Subfolder: "/apps/demo\\",
			},
			expected: "apps/demo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalisedSubfolder(tt.metadata)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPathWithin(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "path_within_test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subdir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		root     string
		expected bool
	}{
		{
			name:     "empty root always returns true",
			path:     "/any/path",
			root:     "",
			expected: true,
		},
		{
			name:     "path equals root",
			path:     tempDir,
			root:     tempDir,
			expected: true,
		},
		{
			name:     "path within root",
			path:     subdir,
			root:     tempDir,
			expected: true,
		},
		{
			name:     "path outside root",
			path:     tempDir,
			root:     subdir,
			expected: false,
		},
		{
			name:     "relative path within root",
			path:     filepath.Join(tempDir, ".", "subdir"),
			root:     tempDir,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PathWithin(tt.path, tt.root)
			if result != tt.expected {
				t.Errorf("PathWithin(%q, %q) = %v, expected %v", tt.path, tt.root, result, tt.expected)
			}
		})
	}
}

func TestResolveRelativeLocalPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "resolve_path_test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a repository structure
	repoRoot := filepath.Join(tempDir, "repo")
	subfolder := filepath.Join(repoRoot, "apps", "demo")
	if err := os.MkdirAll(subfolder, 0755); err != nil {
		t.Fatalf("failed to create subfolder: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(subfolder, "main.py")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name      string
		cleanURI  string
		repoRoot  string
		subfolder string
		absSource string
		expected  string
	}{
		{
			name:      "relative file exists in subfolder",
			cleanURI:  "main.py",
			repoRoot:  repoRoot,
			subfolder: "apps/demo",
			absSource: subfolder,
			expected:  testFile,
		},
		{
			name:      "relative file with repo root only",
			cleanURI:  filepath.Join("apps", "demo", "main.py"),
			repoRoot:  repoRoot,
			subfolder: "",
			absSource: "",
			expected:  testFile,
		},
		{
			name:      "fallback to absSource",
			cleanURI:  "main.py",
			repoRoot:  "",
			subfolder: "",
			absSource: subfolder,
			expected:  testFile,
		},
		{
			name:      "non-existent file returns constructed path",
			cleanURI:  "nonexistent.py",
			repoRoot:  repoRoot,
			subfolder: "",
			absSource: subfolder,
			expected:  filepath.Join(repoRoot, "nonexistent.py"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveRelativeLocalPath(tt.cleanURI, tt.repoRoot, tt.subfolder, tt.absSource)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConvertToRepoRelativePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "convert_path_test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoRoot := filepath.Join(tempDir, "scanio-test")
	subfolder := filepath.Join(repoRoot, "apps", "demo")
	if err := os.MkdirAll(subfolder, 0755); err != nil {
		t.Fatalf("failed to create subfolder: %v", err)
	}

	absoluteFile := filepath.Join(subfolder, "main.py")
	if err := os.WriteFile(absoluteFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	metadata := &git.RepositoryMetadata{
		RepoRootFolder: repoRoot,
		Subfolder:      "apps/demo",
	}

	tests := []struct {
		name         string
		rawURI       string
		metadata     *git.RepositoryMetadata
		sourceFolder string
		expected     string
	}{
		{
			name:         "empty URI",
			rawURI:       "",
			metadata:     metadata,
			sourceFolder: subfolder,
			expected:     "",
		},
		{
			name:         "absolute URI with metadata",
			rawURI:       absoluteFile,
			metadata:     metadata,
			sourceFolder: subfolder,
			expected:     "apps/demo/main.py",
		},
		{
			name:         "relative URI with metadata",
			rawURI:       "main.py",
			metadata:     metadata,
			sourceFolder: subfolder,
			expected:     "apps/demo/main.py",
		},
		{
			name:         "relative URI with file:// prefix",
			rawURI:       "file://main.py",
			metadata:     metadata,
			sourceFolder: subfolder,
			expected:     "apps/demo/main.py",
		},
		{
			name:         "absolute URI without metadata",
			rawURI:       absoluteFile,
			metadata:     nil,
			sourceFolder: subfolder,
			expected:     "main.py",
		},
		{
			name:         "relative URI with parent path",
			rawURI:       filepath.ToSlash(filepath.Join("..", "scanio-test", "apps", "demo", "main.py")),
			metadata:     metadata,
			sourceFolder: subfolder,
			expected:     "apps/demo/main.py",
		},
		{
			name:         "URI already with subfolder prefix",
			rawURI:       "apps/demo/main.py",
			metadata:     metadata,
			sourceFolder: subfolder,
			expected:     "apps/demo/main.py",
		},
		{
			name:         "relative URI without metadata or source folder",
			rawURI:       "src/main.py",
			metadata:     nil,
			sourceFolder: "",
			expected:     "src/main.py",
		},
		{
			name:         "whitespace in URI",
			rawURI:       "  main.py  ",
			metadata:     metadata,
			sourceFolder: subfolder,
			expected:     "apps/demo/main.py",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToRepoRelativePath(tt.rawURI, tt.metadata, tt.sourceFolder)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConvertToRepoRelativePathWithoutRepoRoot(t *testing.T) {
	// Test scenarios where we don't have repository metadata
	tests := []struct {
		name         string
		rawURI       string
		sourceFolder string
		expected     string
	}{
		{
			name:         "relative path without metadata",
			rawURI:       "src/main.py",
			sourceFolder: "/tmp/project",
			expected:     "src/main.py",
		},
		{
			name:         "absolute path without metadata falls back to source folder",
			rawURI:       "/tmp/project/src/main.py",
			sourceFolder: "/tmp/project",
			expected:     "src/main.py",
		},
		{
			name:         "absolute path with no context",
			rawURI:       "/home/user/project/main.py",
			sourceFolder: "",
			expected:     "home/user/project/main.py",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToRepoRelativePath(tt.rawURI, nil, tt.sourceFolder)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConvertToRepoRelativePathCrossPlatform(t *testing.T) {
	// Test that paths are always normalized to forward slashes
	tempDir, err := os.MkdirTemp("", "cross_platform_test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoRoot := filepath.Join(tempDir, "repo")
	subfolder := filepath.Join(repoRoot, "apps", "demo")
	if err := os.MkdirAll(subfolder, 0755); err != nil {
		t.Fatalf("failed to create subfolder: %v", err)
	}

	// Create the file
	mainFile := filepath.Join(subfolder, "main.py")
	if err := os.WriteFile(mainFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	metadata := &git.RepositoryMetadata{
		RepoRootFolder: repoRoot,
		Subfolder:      "apps/demo",
	}

	// Test with a relative path - the internal logic will normalize it
	rawURI := "main.py"
	result := ConvertToRepoRelativePath(rawURI, metadata, subfolder)

	// Result should always use forward slashes
	if strings.Contains(result, "\\") {
		t.Errorf("expected forward slashes only, got %q", result)
	}

	expected := "apps/demo/main.py"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
