package sarifissues

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/internal/git"
	internalsarif "github.com/scan-io-git/scan-io/internal/sarif"
)

func TestDisplaySeverity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Standard SARIF severity levels
		{
			name:     "error level",
			input:    "error",
			expected: "High",
		},
		{
			name:     "warning level",
			input:    "warning",
			expected: "Medium",
		},
		{
			name:     "note level",
			input:    "note",
			expected: "Low",
		},
		{
			name:     "none level",
			input:    "none",
			expected: "Info",
		},
		// Case insensitive tests
		{
			name:     "ERROR uppercase",
			input:    "ERROR",
			expected: "High",
		},
		{
			name:     "Warning mixed case",
			input:    "Warning",
			expected: "Medium",
		},
		{
			name:     "NOTE uppercase",
			input:    "NOTE",
			expected: "Low",
		},
		{
			name:     "NONE uppercase",
			input:    "NONE",
			expected: "Info",
		},
		// Whitespace handling
		{
			name:     "error with leading space",
			input:    " error",
			expected: "High",
		},
		{
			name:     "warning with trailing space",
			input:    "warning ",
			expected: "Medium",
		},
		{
			name:     "note with surrounding spaces",
			input:    " note ",
			expected: "Low",
		},
		{
			name:     "none with tabs",
			input:    "\tnone\t",
			expected: "Info",
		},
		// Edge cases
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "tab only",
			input:    "\t",
			expected: "",
		},
		{
			name:     "newline only",
			input:    "\n",
			expected: "",
		},
		// Custom/unknown severity levels (should be title-cased)
		{
			name:     "custom severity lowercase",
			input:    "critical",
			expected: "Critical",
		},
		{
			name:     "custom severity uppercase",
			input:    "FATAL",
			expected: "Fatal",
		},
		{
			name:     "custom severity mixed case",
			input:    "sEvErE",
			expected: "Severe",
		},
		{
			name:     "custom multi-word severity",
			input:    "very high",
			expected: "Very High",
		},
		{
			name:     "custom with numbers",
			input:    "level1",
			expected: "Level1",
		},
		{
			name:     "custom with special chars",
			input:    "high-priority",
			expected: "High-Priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := displaySeverity(tt.input)
			if result != tt.expected {
				t.Errorf("displaySeverity(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestComputeSnippetHash(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "snippet_hash_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with known content
	testFileContent := `line 1
line 2
line 3
line 4
line 5`

	testFilePath := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFilePath, []byte(testFileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create another test file with different content
	singleLineContent := "single line content"
	singleLineFilePath := filepath.Join(tempDir, "single.txt")
	err = os.WriteFile(singleLineFilePath, []byte(singleLineContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create single line test file: %v", err)
	}

	// Create empty file
	emptyFilePath := filepath.Join(tempDir, "empty.txt")
	err = os.WriteFile(emptyFilePath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty test file: %v", err)
	}

	// Helper function to compute expected hash
	computeExpectedHash := func(content string) string {
		sum := sha256.Sum256([]byte(content))
		return fmt.Sprintf("%x", sum[:])
	}

	tests := []struct {
		name      string
		localPath string
		line      int
		endLine   int
		expected  string
	}{
		// Valid cases
		{
			name:      "single line from middle",
			localPath: testFilePath,
			line:      2,
			endLine:   2,
			expected:  computeExpectedHash("line 2"),
		},
		{
			name:      "multiple lines range",
			localPath: testFilePath,
			line:      2,
			endLine:   4,
			expected:  computeExpectedHash("line 2\nline 3\nline 4"),
		},
		{
			name:      "first line only",
			localPath: testFilePath,
			line:      1,
			endLine:   1,
			expected:  computeExpectedHash("line 1"),
		},
		{
			name:      "last line only",
			localPath: testFilePath,
			line:      5,
			endLine:   5,
			expected:  computeExpectedHash("line 5"),
		},
		{
			name:      "entire file",
			localPath: testFilePath,
			line:      1,
			endLine:   5,
			expected:  computeExpectedHash(testFileContent),
		},
		{
			name:      "single line file",
			localPath: singleLineFilePath,
			line:      1,
			endLine:   1,
			expected:  computeExpectedHash(singleLineContent),
		},
		{
			name:      "endLine same as line (no range)",
			localPath: testFilePath,
			line:      3,
			endLine:   3,
			expected:  computeExpectedHash("line 3"),
		},
		{
			name:      "endLine less than line (should use single line)",
			localPath: testFilePath,
			line:      3,
			endLine:   2,
			expected:  computeExpectedHash("line 3"),
		},

		// Edge cases that should return empty string
		{
			name:      "empty path",
			localPath: "",
			line:      1,
			endLine:   1,
			expected:  "",
		},
		{
			name:      "zero line number",
			localPath: testFilePath,
			line:      0,
			endLine:   1,
			expected:  "",
		},
		{
			name:      "negative line number",
			localPath: testFilePath,
			line:      -1,
			endLine:   1,
			expected:  "",
		},
		{
			name:      "line number beyond file length",
			localPath: testFilePath,
			line:      10,
			endLine:   10,
			expected:  "",
		},
		{
			name:      "file does not exist",
			localPath: filepath.Join(tempDir, "nonexistent.txt"),
			line:      1,
			endLine:   1,
			expected:  "",
		},

		// Boundary cases
		{
			name:      "endLine beyond file length (should clamp)",
			localPath: testFilePath,
			line:      4,
			endLine:   10,
			expected:  computeExpectedHash("line 4\nline 5"),
		},
		{
			name:      "empty file",
			localPath: emptyFilePath,
			line:      1,
			endLine:   1,
			expected:  computeExpectedHash(""), // Empty file has one empty line
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeSnippetHash(tt.localPath, tt.line, tt.endLine)
			if result != tt.expected {
				t.Errorf("computeSnippetHash(%q, %d, %d) = %q, want %q",
					tt.localPath, tt.line, tt.endLine, result, tt.expected)
			}
		})
	}
}

// TestComputeSnippetHash_DifferentContentDifferentHash tests that different
// content produces different hashes
func TestComputeSnippetHash_DifferentContentDifferentHash(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "snippet_hash_different_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create two files with different content
	file1Path := filepath.Join(tempDir, "file1.txt")
	file2Path := filepath.Join(tempDir, "file2.txt")

	err = os.WriteFile(file1Path, []byte("content A"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	err = os.WriteFile(file2Path, []byte("content B"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	hash1 := computeSnippetHash(file1Path, 1, 1)
	hash2 := computeSnippetHash(file2Path, 1, 1)

	if hash1 == hash2 {
		t.Errorf("Different content produced same hash: %q", hash1)
	}

	if hash1 == "" || hash2 == "" {
		t.Error("One or both hashes were empty")
	}
}

// TestComputeSnippetHash_SameContentSameHash tests that identical content
// produces identical hashes regardless of file name
func TestComputeSnippetHash_SameContentSameHash(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "snippet_hash_same_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create two files with identical content but different names
	content := "identical content\nline 2\nline 3"
	file1Path := filepath.Join(tempDir, "identical1.txt")
	file2Path := filepath.Join(tempDir, "identical2.txt")

	err = os.WriteFile(file1Path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	err = os.WriteFile(file2Path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	hash1 := computeSnippetHash(file1Path, 1, 2)
	hash2 := computeSnippetHash(file2Path, 1, 2)

	if hash1 != hash2 {
		t.Errorf("Identical content produced different hashes: %q vs %q", hash1, hash2)
	}

	if hash1 == "" {
		t.Error("Hash was empty for valid content")
	}
}

func TestBuildGitHubPermalink(t *testing.T) {
	fileURI := filepath.ToSlash(filepath.Join("apps", "demo", "main.py"))
	options := RunOptions{
		Namespace:  "scan-io-git",
		Repository: "scanio-test",
		Ref:        "aec0b795c350ff53fe9ab01adf862408aa34c3fd",
	}

	link := buildGitHubPermalink(options, nil, fileURI, 11, 29)
	expected := "https://github.com/scan-io-git/scanio-test/blob/aec0b795c350ff53fe9ab01adf862408aa34c3fd/apps/demo/main.py#L11-L29"
	if link != expected {
		t.Fatalf("expected permalink %q, got %q", expected, link)
	}

	// Ref fallback to repository metadata
	options.Ref = ""
	commit := "1234567890abcdef"
	metadata := &git.RepositoryMetadata{
		RepoRootFolder: "/tmp/repo",
		CommitHash:     &commit,
	}
	link = buildGitHubPermalink(options, metadata, fileURI, 5, 5)
	expected = "https://github.com/scan-io-git/scanio-test/blob/1234567890abcdef/apps/demo/main.py#L5"
	if link != expected {
		t.Fatalf("expected metadata permalink %q, got %q", expected, link)
	}

	// Missing ref and metadata commit should return empty string
	options.Ref = ""
	metadata.CommitHash = nil
	link = buildGitHubPermalink(options, metadata, fileURI, 1, 1)
	if link != "" {
		t.Fatalf("expected empty permalink when ref and metadata are missing, got %q", link)
	}
}

func TestBuildNewIssuesFromSARIFManualScenarios(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sarif_scenarios")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoRoot := filepath.Join(tempDir, "scanio-test")
	subfolder := filepath.Join(repoRoot, "apps", "demo")
	if err := os.MkdirAll(subfolder, 0o755); err != nil {
		t.Fatalf("Failed to create repo subfolder: %v", err)
	}

	mainFile := filepath.Join(subfolder, "main.py")
	var builder strings.Builder
	for i := 1; i <= 60; i++ {
		builder.WriteString(fmt.Sprintf("line %d\n", i))
	}
	if err := os.WriteFile(mainFile, []byte(builder.String()), 0o644); err != nil {
		t.Fatalf("Failed to write main.py: %v", err)
	}

	logger := hclog.NewNullLogger()
	commit := "aec0b795c350ff53fe9ab01adf862408aa34c3fd"

	metadata := &git.RepositoryMetadata{
		RepoRootFolder: repoRoot,
		Subfolder:      filepath.ToSlash(filepath.Join("apps", "demo")),
		CommitHash:     &commit,
	}

	options := RunOptions{
		Namespace:    "scan-io-git",
		Repository:   "scanio-test",
		Ref:          commit,
		SourceFolder: subfolder,
	}

	expectedRepoPath := filepath.ToSlash(filepath.Join("apps", "demo", "main.py"))
	permalink := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s#L%d-L%d",
		options.Namespace,
		options.Repository,
		commit,
		expectedRepoPath,
		11,
		29,
	)

	scenarios := []struct {
		name            string
		uri             string
		sourceFolderCLI string
		sourceFolderAbs string
	}{
		{
			name:            "outside project absolute",
			uri:             mainFile,
			sourceFolderCLI: subfolder,
			sourceFolderAbs: subfolder,
		},
		{
			name:            "outside project relative",
			uri:             filepath.ToSlash(filepath.Join("..", "scanio-test", "apps", "demo", "main.py")),
			sourceFolderCLI: filepath.Join("..", "scanio-test", "apps", "demo"),
			sourceFolderAbs: subfolder,
		},
		{
			name:            "from root absolute",
			uri:             mainFile,
			sourceFolderCLI: subfolder,
			sourceFolderAbs: subfolder,
		},
		{
			name:            "from root relative",
			uri:             filepath.ToSlash(filepath.Join("apps", "demo", "main.py")),
			sourceFolderCLI: filepath.Join("apps", "demo"),
			sourceFolderAbs: subfolder,
		},
		{
			name:            "from subfolder absolute",
			uri:             mainFile,
			sourceFolderCLI: subfolder,
			sourceFolderAbs: subfolder,
		},
		{
			name:            "from subfolder relative",
			uri:             "main.py",
			sourceFolderCLI: ".",
			sourceFolderAbs: subfolder,
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			ruleID := "test.rule"
			uriValue := scenario.uri
			startLine := 11
			endLine := 29
			message := "Test finding"
			baseID := "%SRCROOT%"

			result := &sarif.Result{
				RuleID: &ruleID,
				Message: sarif.Message{
					Text: &message,
				},
				Locations: []*sarif.Location{
					{
						PhysicalLocation: &sarif.PhysicalLocation{
							ArtifactLocation: &sarif.ArtifactLocation{
								URI:       &uriValue,
								URIBaseId: &baseID,
							},
							Region: &sarif.Region{
								StartLine: &startLine,
								EndLine:   &endLine,
							},
						},
					},
				},
			}
			result.PropertyBag = *sarif.NewPropertyBag()
			result.Add("Level", "error")

			report := &internalsarif.Report{
				Report: &sarif.Report{
					Runs: []*sarif.Run{
						{
							Tool: sarif.Tool{
								Driver: &sarif.ToolComponent{
									Name: "Semgrep",
									Rules: []*sarif.ReportingDescriptor{
										{ID: ruleID},
									},
								},
							},
							Results: []*sarif.Result{result},
						},
					},
				},
			}

			scenarioOptions := options
			scenarioOptions.SourceFolder = scenario.sourceFolderCLI

			issues := buildNewIssuesFromSARIF(report, scenarioOptions, scenario.sourceFolderAbs, metadata, logger)
			if len(issues) == 0 {
				t.Fatalf("expected issues for scenario %q", scenario.name)
			}

			issue := issues[0]
			if issue.Metadata.Filename != expectedRepoPath {
				t.Fatalf("scenario %q expected repo path %q, got %q", scenario.name, expectedRepoPath, issue.Metadata.Filename)
			}
			if issue.Metadata.SnippetHash == "" {
				t.Fatalf("scenario %q expected snippet hash to be populated", scenario.name)
			}
			if !strings.Contains(issue.Body, permalink) {
				t.Fatalf("scenario %q issue body missing permalink %q", scenario.name, permalink)
			}
		})
	}
}

func TestResolveSourceFolder(t *testing.T) {
	// Create a test logger
	logger := hclog.NewNullLogger()

	tests := []struct {
		name     string
		input    string
		expected string
		setup    func() (string, func()) // setup function that returns a test path and cleanup function
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   \t\n  ",
			expected: "",
		},
		{
			name:     "relative path",
			input:    "",
			expected: "", // Will be resolved to absolute path
			setup: func() (string, func()) {
				// Create a temporary directory and change to it
				tempDir, err := os.MkdirTemp("", "sarif-test-*")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				testDir := filepath.Join(tempDir, "testdir")
				err = os.Mkdir(testDir, 0755)
				if err != nil {
					os.RemoveAll(tempDir)
					t.Fatalf("failed to create test dir: %v", err)
				}
				return testDir, func() { os.RemoveAll(tempDir) }
			},
		},
		{
			name:     "absolute path",
			input:    "",
			expected: "", // Will be set by setup
			setup: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "sarif-test-*")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				return tempDir, func() { os.RemoveAll(tempDir) }
			},
		},
		{
			name:     "path with tilde expansion",
			input:    "~/testdir",
			expected: "", // Will be resolved to actual home directory path
			setup: func() (string, func()) {
				tempHome, err := os.MkdirTemp("", "sarif-home-*")
				if err != nil {
					t.Fatalf("failed to create temp home dir: %v", err)
				}
				t.Setenv("HOME", tempHome)
				testDir := filepath.Join(tempHome, "testdir")
				if err := os.MkdirAll(testDir, 0o755); err != nil {
					os.RemoveAll(tempHome)
					t.Fatalf("failed to create test dir: %v", err)
				}
				return testDir, func() { os.RemoveAll(tempHome) }
			},
		},
		{
			name:     "path with dots and slashes",
			input:    "",
			expected: "", // Will be cleaned and resolved
			setup: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "sarif-test-*")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				// Create a parent directory structure
				parentDir := filepath.Join(tempDir, "parent")
				err = os.Mkdir(parentDir, 0755)
				if err != nil {
					os.RemoveAll(tempDir)
					t.Fatalf("failed to create parent dir: %v", err)
				}
				testDir := filepath.Join(parentDir, "testdir")
				err = os.Mkdir(testDir, 0755)
				if err != nil {
					os.RemoveAll(tempDir)
					t.Fatalf("failed to create test dir: %v", err)
				}
				return testDir, func() { os.RemoveAll(tempDir) }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cleanup func()
			var expectedPath string

			if tt.setup != nil {
				testPath, cleanupFunc := tt.setup()
				cleanup = cleanupFunc
				expectedPath = filepath.Clean(testPath)

				// Set the input based on test type
				if tt.name == "relative path" {
					// Use absolute path for relative path test since we can't control working directory
					tt.input = testPath
				} else if tt.name == "absolute path" {
					tt.input = testPath
				} else if tt.name == "path with tilde expansion" {
					tt.input = "~/testdir"
				} else if tt.name == "path with dots and slashes" {
					// Use absolute path for this test too
					tt.input = testPath
				}
			} else {
				expectedPath = tt.expected
			}

			result := ResolveSourceFolder(tt.input, logger)

			if tt.setup != nil {
				// For tests with setup, verify the result is an absolute path
				if !filepath.IsAbs(result) {
					t.Errorf("expected absolute path, got relative path: %s", result)
				}
				// Verify the resolved path points to the same directory
				if result != expectedPath {
					t.Errorf("expected %s, got %s", expectedPath, result)
				}
			} else {
				// For tests without setup, verify exact match
				if result != expectedPath {
					t.Errorf("expected %s, got %s", expectedPath, result)
				}
			}

			if cleanup != nil {
				cleanup()
			}
		})
	}
}

func TestResolveSourceFolderErrorHandling(t *testing.T) {
	logger := hclog.NewNullLogger()

	t.Run("non-existent path", func(t *testing.T) {
		// Test with a path that doesn't exist - should still resolve to absolute path
		result := ResolveSourceFolder("/non/existent/path", logger)

		// Should still return an absolute path even if it doesn't exist
		if !filepath.IsAbs(result) {
			t.Errorf("expected absolute path even for non-existent path, got: %s", result)
		}

		expected := "/non/existent/path"
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("invalid characters in path", func(t *testing.T) {
		// Test with path containing invalid characters
		result := ResolveSourceFolder("/tmp/test\x00invalid", logger)

		// Should handle gracefully and return the path as-is
		if result == "" {
			t.Error("expected non-empty result for invalid path")
		}
	})
}

func TestResolveSourceFolderRelativePaths(t *testing.T) {
	logger := hclog.NewNullLogger()

	t.Run("relative path with working directory change", func(t *testing.T) {
		// Create a temporary directory structure
		tempDir, err := os.MkdirTemp("", "sarif-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a test directory
		testDir := filepath.Join(tempDir, "testdir")
		err = os.Mkdir(testDir, 0755)
		if err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		// Change to the temp directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		err = os.Chdir(tempDir)
		if err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}

		// Test relative path
		result := ResolveSourceFolder("./testdir", logger)
		expected := filepath.Clean(testDir)

		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}

		if !filepath.IsAbs(result) {
			t.Errorf("expected absolute path, got relative path: %s", result)
		}
	})
}

func TestApplyEnvironmentFallbacks(t *testing.T) {
	tests := []struct {
		name         string
		initialOpts  RunOptions
		envVars      map[string]string
		expectedOpts RunOptions
	}{
		{
			name: "no environment variables set",
			initialOpts: RunOptions{
				Namespace:  "test-namespace",
				Repository: "test-repo",
				Ref:        "test-ref",
			},
			envVars: map[string]string{},
			expectedOpts: RunOptions{
				Namespace:  "test-namespace",
				Repository: "test-repo",
				Ref:        "test-ref",
			},
		},
		{
			name: "all options already set - no fallbacks applied",
			initialOpts: RunOptions{
				Namespace:  "existing-namespace",
				Repository: "existing-repo",
				Ref:        "existing-ref",
			},
			envVars: map[string]string{
				"GITHUB_REPOSITORY_OWNER": "env-namespace",
				"GITHUB_REPOSITORY":       "env-owner/env-repo",
				"GITHUB_SHA":              "env-sha",
			},
			expectedOpts: RunOptions{
				Namespace:  "existing-namespace",
				Repository: "existing-repo",
				Ref:        "existing-ref",
			},
		},
		{
			name: "namespace fallback applied",
			initialOpts: RunOptions{
				Namespace:  "",
				Repository: "test-repo",
				Ref:        "test-ref",
			},
			envVars: map[string]string{
				"GITHUB_REPOSITORY_OWNER": "env-namespace",
			},
			expectedOpts: RunOptions{
				Namespace:  "env-namespace",
				Repository: "test-repo",
				Ref:        "test-ref",
			},
		},
		{
			name: "repository fallback applied with slash",
			initialOpts: RunOptions{
				Namespace:  "test-namespace",
				Repository: "",
				Ref:        "test-ref",
			},
			envVars: map[string]string{
				"GITHUB_REPOSITORY": "env-owner/env-repo",
			},
			expectedOpts: RunOptions{
				Namespace:  "test-namespace",
				Repository: "env-repo",
				Ref:        "test-ref",
			},
		},
		{
			name: "repository fallback applied without slash",
			initialOpts: RunOptions{
				Namespace:  "test-namespace",
				Repository: "",
				Ref:        "test-ref",
			},
			envVars: map[string]string{
				"GITHUB_REPOSITORY": "env-repo-only",
			},
			expectedOpts: RunOptions{
				Namespace:  "test-namespace",
				Repository: "env-repo-only",
				Ref:        "test-ref",
			},
		},
		{
			name: "ref fallback applied",
			initialOpts: RunOptions{
				Namespace:  "test-namespace",
				Repository: "test-repo",
				Ref:        "",
			},
			envVars: map[string]string{
				"GITHUB_SHA": "env-sha-123",
			},
			expectedOpts: RunOptions{
				Namespace:  "test-namespace",
				Repository: "test-repo",
				Ref:        "env-sha-123",
			},
		},
		{
			name: "all fallbacks applied",
			initialOpts: RunOptions{
				Namespace:  "",
				Repository: "",
				Ref:        "",
			},
			envVars: map[string]string{
				"GITHUB_REPOSITORY_OWNER": "env-namespace",
				"GITHUB_REPOSITORY":       "env-owner/env-repo",
				"GITHUB_SHA":              "env-sha-123",
			},
			expectedOpts: RunOptions{
				Namespace:  "env-namespace",
				Repository: "env-repo",
				Ref:        "env-sha-123",
			},
		},
		{
			name: "whitespace handling",
			initialOpts: RunOptions{
				Namespace:  "   ",
				Repository: "\t",
				Ref:        "\n",
			},
			envVars: map[string]string{
				"GITHUB_REPOSITORY_OWNER": "  env-namespace  ",
				"GITHUB_REPOSITORY":       "\tenv-owner/env-repo\t",
				"GITHUB_SHA":              "\nenv-sha-123\n",
			},
			expectedOpts: RunOptions{
				Namespace:  "env-namespace",
				Repository: "env-repo",
				Ref:        "env-sha-123",
			},
		},
		{
			name: "repository with multiple slashes",
			initialOpts: RunOptions{
				Namespace:  "test-namespace",
				Repository: "",
				Ref:        "test-ref",
			},
			envVars: map[string]string{
				"GITHUB_REPOSITORY": "env-owner/subdir/env-repo",
			},
			expectedOpts: RunOptions{
				Namespace:  "test-namespace",
				Repository: "subdir/env-repo",
				Ref:        "test-ref",
			},
		},
		{
			name: "repository with slash at end",
			initialOpts: RunOptions{
				Namespace:  "test-namespace",
				Repository: "",
				Ref:        "test-ref",
			},
			envVars: map[string]string{
				"GITHUB_REPOSITORY": "env-owner/env-repo/",
			},
			expectedOpts: RunOptions{
				Namespace:  "test-namespace",
				Repository: "env-repo/",
				Ref:        "test-ref",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Create a copy of initial options
			opts := tt.initialOpts

			// Apply environment fallbacks
			ApplyEnvironmentFallbacks(&opts)

			// Verify results
			if opts.Namespace != tt.expectedOpts.Namespace {
				t.Errorf("Namespace: expected %q, got %q", tt.expectedOpts.Namespace, opts.Namespace)
			}
			if opts.Repository != tt.expectedOpts.Repository {
				t.Errorf("Repository: expected %q, got %q", tt.expectedOpts.Repository, opts.Repository)
			}
			if opts.Ref != tt.expectedOpts.Ref {
				t.Errorf("Ref: expected %q, got %q", tt.expectedOpts.Ref, opts.Ref)
			}
		})
	}
}

func TestApplyEnvironmentFallbacksEdgeCases(t *testing.T) {
	t.Run("empty environment variables", func(t *testing.T) {
		opts := RunOptions{
			Namespace:  "",
			Repository: "",
			Ref:        "",
		}

		// Set empty environment variables
		t.Setenv("GITHUB_REPOSITORY_OWNER", "")
		t.Setenv("GITHUB_REPOSITORY", "")
		t.Setenv("GITHUB_SHA", "")

		ApplyEnvironmentFallbacks(&opts)

		// Should remain empty
		if opts.Namespace != "" {
			t.Errorf("Expected empty namespace, got %q", opts.Namespace)
		}
		if opts.Repository != "" {
			t.Errorf("Expected empty repository, got %q", opts.Repository)
		}
		if opts.Ref != "" {
			t.Errorf("Expected empty ref, got %q", opts.Ref)
		}
	})

	t.Run("repository with only slash", func(t *testing.T) {
		opts := RunOptions{
			Repository: "",
		}

		t.Setenv("GITHUB_REPOSITORY", "/")

		ApplyEnvironmentFallbacks(&opts)

		// Should fall back to the whole value
		if opts.Repository != "/" {
			t.Errorf("Expected repository to be '/', got %q", opts.Repository)
		}
	})

	t.Run("repository with slash at beginning", func(t *testing.T) {
		opts := RunOptions{
			Repository: "",
		}

		t.Setenv("GITHUB_REPOSITORY", "/env-repo")

		ApplyEnvironmentFallbacks(&opts)

		// Should extract the part after the slash since idx=0 and idx < len(gr)-1
		if opts.Repository != "env-repo" {
			t.Errorf("Expected repository to be 'env-repo', got %q", opts.Repository)
		}
	})
}

func TestBuildIssueBodyWithCodeQLMessage(t *testing.T) {
	// Test integration with CodeQL-style SARIF data using mocks
	tempDir, err := os.MkdirTemp("", "sarif_codeql_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file structure
	repoRoot := filepath.Join(tempDir, "scanio-test")
	subfolder := filepath.Join(repoRoot, "apps", "demo")
	if err := os.MkdirAll(subfolder, 0o755); err != nil {
		t.Fatalf("Failed to create repo subfolder: %v", err)
	}

	mainFile := filepath.Join(subfolder, "main.py")
	if err := os.WriteFile(mainFile, []byte("line 1\nline 2\nline 3\n"), 0o644); err != nil {
		t.Fatalf("Failed to write main.py: %v", err)
	}

	logger := hclog.NewNullLogger()
	commit := "aec0b795c350ff53fe9ab01adf862408aa34c3fd"

	metadata := &git.RepositoryMetadata{
		RepoRootFolder: repoRoot,
		Subfolder:      filepath.ToSlash(filepath.Join("apps", "demo")),
		CommitHash:     &commit,
	}

	options := RunOptions{
		Namespace:    "test-org",
		Repository:   "test-repo",
		Ref:          commit,
		SourceFolder: subfolder,
	}

	// Create mock SARIF report with CodeQL-style message
	sarifReport := &sarif.Report{
		Runs: []*sarif.Run{
			{
				Tool: sarif.Tool{
					Driver: &sarif.ToolComponent{
						Rules: []*sarif.ReportingDescriptor{
							{
								ID: "py/template-injection",
								Properties: map[string]interface{}{
									"problem.severity": "error",
								},
							},
						},
					},
				},
				Results: []*sarif.Result{
					{
						RuleID: stringPtr("py/template-injection"),
						Level:  stringPtr("error"),
						Message: sarif.Message{
							Text: stringPtr("This template construction depends on a [user-provided value](1)."),
						},
						RelatedLocations: []*sarif.Location{
							{
								Id: uintPtr(1),
								PhysicalLocation: &sarif.PhysicalLocation{
									ArtifactLocation: &sarif.ArtifactLocation{
										URI: stringPtr("main.py"),
									},
									Region: &sarif.Region{
										StartLine:   intPtr(1),
										StartColumn: intPtr(50),
										EndLine:     intPtr(1),
										EndColumn:   intPtr(57),
									},
								},
							},
						},
						Locations: []*sarif.Location{
							{
								PhysicalLocation: &sarif.PhysicalLocation{
									ArtifactLocation: &sarif.ArtifactLocation{
										URI: stringPtr("main.py"),
									},
									Region: &sarif.Region{
										StartLine:   intPtr(10),
										StartColumn: intPtr(5),
										EndLine:     intPtr(10),
										EndColumn:   intPtr(15),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	report := &internalsarif.Report{
		Report: sarifReport,
	}

	// Enrich results with level property
	report.EnrichResultsLevelProperty()

	// Process SARIF results
	issues := buildNewIssuesFromSARIF(report, options, subfolder, metadata, logger)

	// Verify that formatted messages are included in issue bodies
	if len(issues) == 0 {
		t.Fatalf("Expected at least one issue, got 0")
	}

	issue := issues[0]
	if !strings.Contains(issue.Body, "### Description") {
		t.Errorf("Expected issue body to contain '### Description' section")
	}

	// Check that the formatted message contains a hyperlink
	if !strings.Contains(issue.Body, "https://github.com/test-org/test-repo/blob/") {
		t.Errorf("Expected issue body to contain GitHub permalink")
	}

	// Check for CodeQL-style formatting (single reference)
	if !strings.Contains(issue.Body, "[user-provided value](") {
		t.Errorf("Expected issue body to contain CodeQL-style formatted reference")
	}
}

func TestBuildIssueBodyWithSnykMessage(t *testing.T) {
	// Test integration with Snyk-style SARIF data using mocks
	tempDir, err := os.MkdirTemp("", "sarif_snyk_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file structure
	repoRoot := filepath.Join(tempDir, "scanio-test")
	subfolder := filepath.Join(repoRoot, "apps", "demo")
	if err := os.MkdirAll(subfolder, 0o755); err != nil {
		t.Fatalf("Failed to create repo subfolder: %v", err)
	}

	mainFile := filepath.Join(subfolder, "main.py")
	if err := os.WriteFile(mainFile, []byte("line 1\nline 2\nline 3\n"), 0o644); err != nil {
		t.Fatalf("Failed to write main.py: %v", err)
	}

	logger := hclog.NewNullLogger()
	commit := "aec0b795c350ff53fe9ab01adf862408aa34c3fd"

	metadata := &git.RepositoryMetadata{
		RepoRootFolder: repoRoot,
		Subfolder:      filepath.ToSlash(filepath.Join("apps", "demo")),
		CommitHash:     &commit,
	}

	options := RunOptions{
		Namespace:    "test-org",
		Repository:   "test-repo",
		Ref:          commit,
		SourceFolder: subfolder,
	}

	// Create mock SARIF report with Snyk-style message
	sarifReport := &sarif.Report{
		Runs: []*sarif.Run{
			{
				Tool: sarif.Tool{
					Driver: &sarif.ToolComponent{
						Rules: []*sarif.ReportingDescriptor{
							{
								ID: "python/Ssti",
								Properties: map[string]interface{}{
									"problem.severity": "error",
								},
							},
						},
					},
				},
				Results: []*sarif.Result{
					{
						RuleID: stringPtr("python/Ssti"),
						Level:  stringPtr("error"),
						Message: sarif.Message{
							Markdown: stringPtr("Unsanitized input from {0} {1} into {2}, where it is used to render an HTML page returned to the user. This may result in a Cross-Site Scripting attack (XSS)."),
							Arguments: []string{
								"[an HTTP parameter](0)",
								"[flows](1),(2),(3),(4),(5),(6)",
								"[flask.render_template_string](7)",
							},
						},
						RelatedLocations: []*sarif.Location{
							{
								Id: uintPtr(0),
								PhysicalLocation: &sarif.PhysicalLocation{
									ArtifactLocation: &sarif.ArtifactLocation{
										URI: stringPtr("main.py"),
									},
									Region: &sarif.Region{
										StartLine:   intPtr(1),
										StartColumn: intPtr(50),
										EndLine:     intPtr(1),
										EndColumn:   intPtr(57),
									},
								},
							},
							{
								Id: uintPtr(1),
								PhysicalLocation: &sarif.PhysicalLocation{
									ArtifactLocation: &sarif.ArtifactLocation{
										URI: stringPtr("main.py"),
									},
									Region: &sarif.Region{
										StartLine:   intPtr(8),
										StartColumn: intPtr(18),
										EndLine:     intPtr(8),
										EndColumn:   intPtr(25),
									},
								},
							},
							{
								Id: uintPtr(2),
								PhysicalLocation: &sarif.PhysicalLocation{
									ArtifactLocation: &sarif.ArtifactLocation{
										URI: stringPtr("main.py"),
									},
									Region: &sarif.Region{
										StartLine:   intPtr(8),
										StartColumn: intPtr(18),
										EndLine:     intPtr(8),
										EndColumn:   intPtr(30),
									},
								},
							},
							{
								Id: uintPtr(3),
								PhysicalLocation: &sarif.PhysicalLocation{
									ArtifactLocation: &sarif.ArtifactLocation{
										URI: stringPtr("main.py"),
									},
									Region: &sarif.Region{
										StartLine:   intPtr(8),
										StartColumn: intPtr(18),
										EndLine:     intPtr(8),
										EndColumn:   intPtr(46),
									},
								},
							},
							{
								Id: uintPtr(4),
								PhysicalLocation: &sarif.PhysicalLocation{
									ArtifactLocation: &sarif.ArtifactLocation{
										URI: stringPtr("main.py"),
									},
									Region: &sarif.Region{
										StartLine:   intPtr(8),
										StartColumn: intPtr(5),
										EndLine:     intPtr(8),
										EndColumn:   intPtr(15),
									},
								},
							},
							{
								Id: uintPtr(5),
								PhysicalLocation: &sarif.PhysicalLocation{
									ArtifactLocation: &sarif.ArtifactLocation{
										URI: stringPtr("main.py"),
									},
									Region: &sarif.Region{
										StartLine:   intPtr(11),
										StartColumn: intPtr(5),
										EndLine:     intPtr(11),
										EndColumn:   intPtr(13),
									},
								},
							},
							{
								Id: uintPtr(6),
								PhysicalLocation: &sarif.PhysicalLocation{
									ArtifactLocation: &sarif.ArtifactLocation{
										URI: stringPtr("main.py"),
									},
									Region: &sarif.Region{
										StartLine:   intPtr(29),
										StartColumn: intPtr(35),
										EndLine:     intPtr(29),
										EndColumn:   intPtr(43),
									},
								},
							},
							{
								Id: uintPtr(7),
								PhysicalLocation: &sarif.PhysicalLocation{
									ArtifactLocation: &sarif.ArtifactLocation{
										URI: stringPtr("main.py"),
									},
									Region: &sarif.Region{
										StartLine:   intPtr(29),
										StartColumn: intPtr(12),
										EndLine:     intPtr(29),
										EndColumn:   intPtr(44),
									},
								},
							},
						},
						Locations: []*sarif.Location{
							{
								PhysicalLocation: &sarif.PhysicalLocation{
									ArtifactLocation: &sarif.ArtifactLocation{
										URI: stringPtr("main.py"),
									},
									Region: &sarif.Region{
										StartLine:   intPtr(29),
										StartColumn: intPtr(12),
										EndLine:     intPtr(29),
										EndColumn:   intPtr(44),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	report := &internalsarif.Report{
		Report: sarifReport,
	}

	// Enrich results with level property
	report.EnrichResultsLevelProperty()

	// Process SARIF results
	issues := buildNewIssuesFromSARIF(report, options, subfolder, metadata, logger)

	// Verify that formatted messages are included in issue bodies
	if len(issues) == 0 {
		t.Fatalf("Expected at least one issue, got 0")
	}

	issue := issues[0]

	if !strings.Contains(issue.Body, "### Description") {
		t.Errorf("Expected issue body to contain '### Description' section")
	}

	// Check that the formatted message contains hyperlinks
	if !strings.Contains(issue.Body, "https://github.com/test-org/test-repo/blob/") {
		t.Errorf("Expected issue body to contain GitHub permalink")
	}

	// Check for flow chain formatting (multiple references)
	if !strings.Contains(issue.Body, " > ") {
		t.Errorf("Expected issue body to contain flow chain formatting")
	}

	// Check for Snyk-style formatting (multiple references in flow chain)
	if !strings.Contains(issue.Body, "flows (") {
		t.Errorf("Expected issue body to contain Snyk-style formatted flow reference")
	}
}

func TestFormatCodeFlows(t *testing.T) {
	tests := []struct {
		name     string
		result   *sarif.Result
		expected string
	}{
		{
			name:     "no code flows",
			result:   &sarif.Result{},
			expected: "",
		},
		{
			name: "nil code flows",
			result: &sarif.Result{
				CodeFlows: nil,
			},
			expected: "",
		},
		{
			name: "empty code flows",
			result: &sarif.Result{
				CodeFlows: []*sarif.CodeFlow{},
			},
			expected: "",
		},
		{
			name: "single thread flow with message text",
			result: &sarif.Result{
				CodeFlows: []*sarif.CodeFlow{
					{
						ThreadFlows: []*sarif.ThreadFlow{
							{
								Locations: []*sarif.ThreadFlowLocation{
									{
										Location: &sarif.Location{
											PhysicalLocation: &sarif.PhysicalLocation{
												ArtifactLocation: &sarif.ArtifactLocation{
													URI: stringPtr("main.py"),
												},
												Region: &sarif.Region{
													StartLine: intPtr(1),
													EndLine:   intPtr(1),
												},
											},
											Message: &sarif.Message{
												Text: stringPtr("ControlFlowNode for ImportMember"),
											},
										},
									},
									{
										Location: &sarif.Location{
											PhysicalLocation: &sarif.PhysicalLocation{
												ArtifactLocation: &sarif.ArtifactLocation{
													URI: stringPtr("main.py"),
												},
												Region: &sarif.Region{
													StartLine: intPtr(8),
													EndLine:   intPtr(8),
												},
											},
											Message: &sarif.Message{
												Text: stringPtr("ControlFlowNode for request"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: `<details>
<summary>Code Flow 1</summary>

Step 1: ControlFlowNode for ImportMember
https://github.com/test-org/test-repo/blob/test-ref/main.py#L1

Step 2: ControlFlowNode for request
https://github.com/test-org/test-repo/blob/test-ref/main.py#L8
</details>`,
		},
		{
			name: "single thread flow without message text",
			result: &sarif.Result{
				CodeFlows: []*sarif.CodeFlow{
					{
						ThreadFlows: []*sarif.ThreadFlow{
							{
								Locations: []*sarif.ThreadFlowLocation{
									{
										Location: &sarif.Location{
											PhysicalLocation: &sarif.PhysicalLocation{
												ArtifactLocation: &sarif.ArtifactLocation{
													URI: stringPtr("app.py"),
												},
												Region: &sarif.Region{
													StartLine: intPtr(5),
													EndLine:   intPtr(5),
												},
											},
											Message: &sarif.Message{
												Text: stringPtr(""),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: `<details>
<summary>Code Flow 1</summary>

Step 1:
https://github.com/test-org/test-repo/blob/test-ref/app.py#L5
</details>`,
		},
		{
			name: "multiple thread flows",
			result: &sarif.Result{
				CodeFlows: []*sarif.CodeFlow{
					{
						ThreadFlows: []*sarif.ThreadFlow{
							{
								Locations: []*sarif.ThreadFlowLocation{
									{
										Location: &sarif.Location{
											PhysicalLocation: &sarif.PhysicalLocation{
												ArtifactLocation: &sarif.ArtifactLocation{
													URI: stringPtr("file1.py"),
												},
												Region: &sarif.Region{
													StartLine: intPtr(1),
													EndLine:   intPtr(1),
												},
											},
											Message: &sarif.Message{
												Text: stringPtr("First flow step"),
											},
										},
									},
								},
							},
							{
								Locations: []*sarif.ThreadFlowLocation{
									{
										Location: &sarif.Location{
											PhysicalLocation: &sarif.PhysicalLocation{
												ArtifactLocation: &sarif.ArtifactLocation{
													URI: stringPtr("file2.py"),
												},
												Region: &sarif.Region{
													StartLine: intPtr(10),
													EndLine:   intPtr(10),
												},
											},
											Message: &sarif.Message{
												Text: stringPtr("Second flow step"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: `<details>
<summary>Code Flow 1</summary>

Step 1: First flow step
https://github.com/test-org/test-repo/blob/test-ref/file1.py#L1
</details>

<details>
<summary>Code Flow 2</summary>

Step 1: Second flow step
https://github.com/test-org/test-repo/blob/test-ref/file2.py#L10
</details>`,
		},
		{
			name: "thread flow with line range",
			result: &sarif.Result{
				CodeFlows: []*sarif.CodeFlow{
					{
						ThreadFlows: []*sarif.ThreadFlow{
							{
								Locations: []*sarif.ThreadFlowLocation{
									{
										Location: &sarif.Location{
											PhysicalLocation: &sarif.PhysicalLocation{
												ArtifactLocation: &sarif.ArtifactLocation{
													URI: stringPtr("main.py"),
												},
												Region: &sarif.Region{
													StartLine: intPtr(10),
													EndLine:   intPtr(15),
												},
											},
											Message: &sarif.Message{
												Text: stringPtr("Multi-line location"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: `<details>
<summary>Code Flow 1</summary>

Step 1: Multi-line location
https://github.com/test-org/test-repo/blob/test-ref/main.py#L10-L15
</details>`,
		},
		{
			name: "duplicate steps with same permalink and message",
			result: &sarif.Result{
				CodeFlows: []*sarif.CodeFlow{
					{
						ThreadFlows: []*sarif.ThreadFlow{
							{
								Locations: []*sarif.ThreadFlowLocation{
									{
										Location: &sarif.Location{
											PhysicalLocation: &sarif.PhysicalLocation{
												ArtifactLocation: &sarif.ArtifactLocation{
													URI: stringPtr("main.py"),
												},
												Region: &sarif.Region{
													StartLine: intPtr(1),
													EndLine:   intPtr(1),
												},
											},
											Message: &sarif.Message{
												Text: stringPtr("First step"),
											},
										},
									},
									{
										Location: &sarif.Location{
											PhysicalLocation: &sarif.PhysicalLocation{
												ArtifactLocation: &sarif.ArtifactLocation{
													URI: stringPtr("main.py"),
												},
												Region: &sarif.Region{
													StartLine: intPtr(1),
													EndLine:   intPtr(1),
												},
											},
											Message: &sarif.Message{
												Text: stringPtr("First step"), // Same message as previous
											},
										},
									},
									{
										Location: &sarif.Location{
											PhysicalLocation: &sarif.PhysicalLocation{
												ArtifactLocation: &sarif.ArtifactLocation{
													URI: stringPtr("main.py"),
												},
												Region: &sarif.Region{
													StartLine: intPtr(2),
													EndLine:   intPtr(2),
												},
											},
											Message: &sarif.Message{
												Text: stringPtr("Second step"), // Different message
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: `<details>
<summary>Code Flow 1</summary>

Step 1: First step
https://github.com/test-org/test-repo/blob/test-ref/main.py#L1

Step 2: Second step
https://github.com/test-org/test-repo/blob/test-ref/main.py#L2
</details>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := RunOptions{
				Namespace:  "test-org",
				Repository: "test-repo",
				Ref:        "test-ref",
			}

			repoMetadata := &git.RepositoryMetadata{
				RepoRootFolder: "/test/repo",
			}

			result := FormatCodeFlows(tt.result, options, repoMetadata, "/test/repo")

			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

// Helper functions for creating test data
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func uintPtr(u uint) *uint {
	return &u
}
