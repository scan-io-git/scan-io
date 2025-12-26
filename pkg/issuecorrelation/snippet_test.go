package issuecorrelation

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

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
			result := ComputeSnippetHash(tt.localPath, tt.line, tt.endLine)
			if result != tt.expected {
				t.Errorf("ComputeSnippetHash(%q, %d, %d) = %q, want %q",
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

	hash1 := ComputeSnippetHash(file1Path, 1, 1)
	hash2 := ComputeSnippetHash(file2Path, 1, 1)

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

	hash1 := ComputeSnippetHash(file1Path, 1, 2)
	hash2 := ComputeSnippetHash(file2Path, 1, 2)

	if hash1 != hash2 {
		t.Errorf("Identical content produced different hashes: %q vs %q", hash1, hash2)
	}

	if hash1 == "" {
		t.Error("Hash was empty for valid content")
	}
}
