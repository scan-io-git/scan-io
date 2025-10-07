package sarifissues

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
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
		name         string
		fileURI      string
		line         int
		endLine      int
		sourceFolder string
		expected     string
	}{
		// Valid cases
		{
			name:         "single line from middle",
			fileURI:      "test.txt",
			line:         2,
			endLine:      2,
			sourceFolder: tempDir,
			expected:     computeExpectedHash("line 2"),
		},
		{
			name:         "multiple lines range",
			fileURI:      "test.txt",
			line:         2,
			endLine:      4,
			sourceFolder: tempDir,
			expected:     computeExpectedHash("line 2\nline 3\nline 4"),
		},
		{
			name:         "first line only",
			fileURI:      "test.txt",
			line:         1,
			endLine:      1,
			sourceFolder: tempDir,
			expected:     computeExpectedHash("line 1"),
		},
		{
			name:         "last line only",
			fileURI:      "test.txt",
			line:         5,
			endLine:      5,
			sourceFolder: tempDir,
			expected:     computeExpectedHash("line 5"),
		},
		{
			name:         "entire file",
			fileURI:      "test.txt",
			line:         1,
			endLine:      5,
			sourceFolder: tempDir,
			expected:     computeExpectedHash(testFileContent),
		},
		{
			name:         "single line file",
			fileURI:      "single.txt",
			line:         1,
			endLine:      1,
			sourceFolder: tempDir,
			expected:     computeExpectedHash(singleLineContent),
		},
		{
			name:         "endLine same as line (no range)",
			fileURI:      "test.txt",
			line:         3,
			endLine:      3,
			sourceFolder: tempDir,
			expected:     computeExpectedHash("line 3"),
		},
		{
			name:         "endLine less than line (should use single line)",
			fileURI:      "test.txt",
			line:         3,
			endLine:      2,
			sourceFolder: tempDir,
			expected:     computeExpectedHash("line 3"),
		},

		// Edge cases that should return empty string
		{
			name:         "empty fileURI",
			fileURI:      "",
			line:         1,
			endLine:      1,
			sourceFolder: tempDir,
			expected:     "",
		},
		{
			name:         "unknown fileURI",
			fileURI:      "<unknown>",
			line:         1,
			endLine:      1,
			sourceFolder: tempDir,
			expected:     "",
		},
		{
			name:         "zero line number",
			fileURI:      "test.txt",
			line:         0,
			endLine:      1,
			sourceFolder: tempDir,
			expected:     "",
		},
		{
			name:         "negative line number",
			fileURI:      "test.txt",
			line:         -1,
			endLine:      1,
			sourceFolder: tempDir,
			expected:     "",
		},
		{
			name:         "empty sourceFolder",
			fileURI:      "test.txt",
			line:         1,
			endLine:      1,
			sourceFolder: "",
			expected:     "",
		},
		{
			name:         "line number beyond file length",
			fileURI:      "test.txt",
			line:         10,
			endLine:      10,
			sourceFolder: tempDir,
			expected:     "",
		},
		{
			name:         "file does not exist",
			fileURI:      "nonexistent.txt",
			line:         1,
			endLine:      1,
			sourceFolder: tempDir,
			expected:     "",
		},
		{
			name:         "invalid sourceFolder",
			fileURI:      "test.txt",
			line:         1,
			endLine:      1,
			sourceFolder: "/nonexistent/path",
			expected:     "",
		},

		// Boundary cases
		{
			name:         "endLine beyond file length (should clamp)",
			fileURI:      "test.txt",
			line:         4,
			endLine:      10,
			sourceFolder: tempDir,
			expected:     computeExpectedHash("line 4\nline 5"),
		},
		{
			name:         "empty file",
			fileURI:      "empty.txt",
			line:         1,
			endLine:      1,
			sourceFolder: tempDir,
			expected:     computeExpectedHash(""), // Empty file has one empty line
		},

		// Path handling
		{
			name:         "fileURI with forward slashes",
			fileURI:      "test.txt",
			line:         1,
			endLine:      1,
			sourceFolder: tempDir,
			expected:     computeExpectedHash("line 1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeSnippetHash(tt.fileURI, tt.line, tt.endLine, tt.sourceFolder)
			if result != tt.expected {
				t.Errorf("computeSnippetHash(%q, %d, %d, %q) = %q, want %q",
					tt.fileURI, tt.line, tt.endLine, tt.sourceFolder, result, tt.expected)
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

	hash1 := computeSnippetHash("file1.txt", 1, 1, tempDir)
	hash2 := computeSnippetHash("file2.txt", 1, 1, tempDir)

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

	hash1 := computeSnippetHash("identical1.txt", 1, 2, tempDir)
	hash2 := computeSnippetHash("identical2.txt", 1, 2, tempDir)

	if hash1 != hash2 {
		t.Errorf("Identical content produced different hashes: %q vs %q", hash1, hash2)
	}

	if hash1 == "" {
		t.Error("Hash was empty for valid content")
	}
}
