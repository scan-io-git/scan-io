package sarif

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/internal/git"
)

func TestFormatMessageWithSingleReference(t *testing.T) {
	// Test CodeQL style single reference: "[user-provided value](1)"
	message := &sarif.Message{
		Markdown: stringPtr("This template construction depends on a {0}."),
		Arguments: []string{
			"[user-provided value](0)",
		},
	}

	locations := []*sarif.Location{
		createTestLocation("main.py", 1, 50, 1, 57),
	}

	repoMetadata := &git.RepositoryMetadata{
		RepoRootFolder: "/test/source",
		CommitHash:     stringPtr("abc123"),
	}

	options := MessageFormatOptions{
		Namespace:    "test-org",
		Repository:   "test-repo",
		Ref:          "main",
		SourceFolder: "/test/source",
	}

	result := formatMessageWithArguments(message, locations, repoMetadata, options)
	expected := "This template construction depends on a [user-provided value](https://github.com/test-org/test-repo/blob/main/main.py#L1)."

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

func TestFormatMessageWithMultipleReferences(t *testing.T) {
	// Test Snyk style multiple references: "[flows](1),(2),(3),(4),(5),(6)"
	message := &sarif.Message{
		Markdown: stringPtr("Unsanitized input from {0} {1} into {2}, where it is used to render an HTML page returned to the user. This may result in a Cross-Site Scripting attack (XSS)."),
		Arguments: []string{
			"[an HTTP parameter](0)",
			"[flows](1),(2),(3),(4),(5),(6)",
			"[flask.render_template_string](7)",
		},
	}

	locations := []*sarif.Location{
		createTestLocation("main.py", 1, 50, 1, 57),   // 0
		createTestLocation("main.py", 8, 18, 8, 25),   // 1
		createTestLocation("main.py", 8, 18, 8, 30),   // 2
		createTestLocation("main.py", 8, 18, 8, 46),   // 3
		createTestLocation("main.py", 8, 5, 8, 15),    // 4
		createTestLocation("main.py", 11, 5, 11, 13),  // 5
		createTestLocation("main.py", 29, 35, 29, 43), // 6
		createTestLocation("main.py", 29, 12, 29, 44), // 7
	}

	repoMetadata := &git.RepositoryMetadata{
		RepoRootFolder: "/test/source",
		CommitHash:     stringPtr("abc123"),
	}

	options := MessageFormatOptions{
		Namespace:    "test-org",
		Repository:   "test-repo",
		Ref:          "main",
		SourceFolder: "/test/source",
	}

	result := formatMessageWithArguments(message, locations, repoMetadata, options)
	expected := "Unsanitized input from [an HTTP parameter](https://github.com/test-org/test-repo/blob/main/main.py#L1) flows ([1](https://github.com/test-org/test-repo/blob/main/main.py#L8) > [2](https://github.com/test-org/test-repo/blob/main/main.py#L8) > [3](https://github.com/test-org/test-repo/blob/main/main.py#L8) > [4](https://github.com/test-org/test-repo/blob/main/main.py#L8) > [5](https://github.com/test-org/test-repo/blob/main/main.py#L11) > [6](https://github.com/test-org/test-repo/blob/main/main.py#L29)) into [flask.render_template_string](https://github.com/test-org/test-repo/blob/main/main.py#L29), where it is used to render an HTML page returned to the user. This may result in a Cross-Site Scripting attack (XSS)."

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

func TestFormatMessageWithPlaceholders(t *testing.T) {
	// Test template with placeholders
	message := &sarif.Message{
		Markdown: stringPtr("Input from {0} flows to {1}"),
		Arguments: []string{
			"[user input](0)",
			"[template](1)",
		},
	}

	locations := []*sarif.Location{
		createTestLocation("main.py", 1, 1, 1, 10),
		createTestLocation("main.py", 2, 1, 2, 10),
	}

	repoMetadata := &git.RepositoryMetadata{
		RepoRootFolder: "/test/source",
		CommitHash:     stringPtr("abc123"),
	}

	options := MessageFormatOptions{
		Namespace:    "test-org",
		Repository:   "test-repo",
		Ref:          "main",
		SourceFolder: "/test/source",
	}

	result := formatMessageWithArguments(message, locations, repoMetadata, options)
	expected := "Input from [user input](https://github.com/test-org/test-repo/blob/main/main.py#L1) flows to [template](https://github.com/test-org/test-repo/blob/main/main.py#L2)"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

func TestFormatMessageNoMarkdown(t *testing.T) {
	// Test fallback to plain text when no markdown template
	message := &sarif.Message{
		Text: stringPtr("Plain text message without formatting"),
	}

	locations := []*sarif.Location{}

	repoMetadata := &git.RepositoryMetadata{
		RepoRootFolder: "/test/source",
		CommitHash:     stringPtr("abc123"),
	}

	options := MessageFormatOptions{
		Namespace:    "test-org",
		Repository:   "test-repo",
		Ref:          "main",
		SourceFolder: "/test/source",
	}

	result := formatMessageWithArguments(message, locations, repoMetadata, options)
	expected := "Plain text message without formatting"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

func TestFormatMessageMissingLocations(t *testing.T) {
	// Test graceful degradation when locations are missing
	message := &sarif.Message{
		Markdown: stringPtr("Input from {0} flows to {1}"),
		Arguments: []string{
			"[user input](0)",
			"[template](1)",
		},
	}

	locations := []*sarif.Location{} // Empty locations

	repoMetadata := &git.RepositoryMetadata{
		RepoRootFolder: "/test/source",
		CommitHash:     stringPtr("abc123"),
	}

	options := MessageFormatOptions{
		Namespace:    "test-org",
		Repository:   "test-repo",
		Ref:          "main",
		SourceFolder: "/test/source",
	}

	result := formatMessageWithArguments(message, locations, repoMetadata, options)
	expected := "Input from user input (0) flows to template (1)" // References show as plain text with numbers

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

func TestExtractLocationsFromRelatedLocations(t *testing.T) {
	// Test priority 1: relatedLocations
	result := &sarif.Result{
		RelatedLocations: []*sarif.Location{
			createTestLocation("main.py", 1, 50, 1, 57),
		},
	}

	locations := extractLocationsForFormatting(result)

	if len(locations) != 1 {
		t.Errorf("Expected 1 location, got %d", len(locations))
	}

	if locations[0].PhysicalLocation.ArtifactLocation.URI == nil || *locations[0].PhysicalLocation.ArtifactLocation.URI != "main.py" {
		t.Errorf("Expected location URI 'main.py', got %v", locations[0].PhysicalLocation.ArtifactLocation.URI)
	}
}

func TestExtractLocationsFromCodeFlows(t *testing.T) {
	// Test priority 2: codeFlows fallback
	result := &sarif.Result{
		CodeFlows: []*sarif.CodeFlow{
			{
				ThreadFlows: []*sarif.ThreadFlow{
					{
						Locations: []*sarif.ThreadFlowLocation{
							{
								Location: createTestLocation("main.py", 1, 50, 1, 57),
							},
							{
								Location: createTestLocation("main.py", 8, 18, 8, 25),
							},
						},
					},
				},
			},
		},
	}

	locations := extractLocationsForFormatting(result)

	if len(locations) != 2 {
		t.Errorf("Expected 2 locations, got %d", len(locations))
	}
}

func TestExtractLocationsEmpty(t *testing.T) {
	// Test priority 3: empty fallback
	result := &sarif.Result{}

	locations := extractLocationsForFormatting(result)

	if len(locations) != 0 {
		t.Errorf("Expected 0 locations, got %d", len(locations))
	}
}

func TestParseLocationReference(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		refs     []int
	}{
		{
			name:     "single reference",
			input:    "[user-provided value](1)",
			expected: "user-provided value",
			refs:     []int{1},
		},
		{
			name:     "multiple references",
			input:    "[flows](1),(2),(3),(4),(5),(6)",
			expected: "flows",
			refs:     []int{1, 2, 3, 4, 5, 6},
		},
		{
			name:     "malformed input",
			input:    "plain text without brackets",
			expected: "plain text without brackets",
			refs:     nil,
		},
		{
			name:     "invalid reference numbers",
			input:    "[text](abc),(def)",
			expected: "text",
			refs:     []int{}, // Invalid numbers are skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, refs := parseLocationReference(tt.input)
			if text != tt.expected {
				t.Errorf("Expected text '%s', got '%s'", tt.expected, text)
			}
			if len(refs) != len(tt.refs) {
				t.Errorf("Expected %d refs, got %d", len(tt.refs), len(refs))
			}
			for i, ref := range refs {
				if ref != tt.refs[i] {
					t.Errorf("Expected ref[%d] = %d, got %d", i, tt.refs[i], ref)
				}
			}
		})
	}
}

func TestBuildLocationLink(t *testing.T) {
	location := createTestLocation("main.py", 10, 5, 10, 15)

	repoMetadata := &git.RepositoryMetadata{
		RepoRootFolder: "/test/source",
		CommitHash:     stringPtr("abc123"),
	}

	options := MessageFormatOptions{
		Namespace:    "test-org",
		Repository:   "test-repo",
		Ref:          "main",
		SourceFolder: "/test/source",
	}

	result := buildLocationLink(location, repoMetadata, options)
	expected := "https://github.com/test-org/test-repo/blob/main/main.py#L10"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

func TestBuildLocationLinkRange(t *testing.T) {
	location := createTestLocation("main.py", 10, 5, 15, 20)

	repoMetadata := &git.RepositoryMetadata{
		RepoRootFolder: "/test/source",
		CommitHash:     stringPtr("abc123"),
	}

	options := MessageFormatOptions{
		Namespace:    "test-org",
		Repository:   "test-repo",
		Ref:          "main",
		SourceFolder: "/test/source",
	}

	result := buildLocationLink(location, repoMetadata, options)
	expected := "https://github.com/test-org/test-repo/blob/main/main.py#L10-L15"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

func TestBuildLocationLinkAbsolutePath(t *testing.T) {
	location := createTestLocation("/test/source/main.py", 10, 5, 10, 15)

	repoMetadata := &git.RepositoryMetadata{
		RepoRootFolder: "/test/source",
		CommitHash:     stringPtr("abc123"),
	}

	options := MessageFormatOptions{
		Namespace:    "test-org",
		Repository:   "test-repo",
		Ref:          "main",
		SourceFolder: "/test/source",
	}

	result := buildLocationLink(location, repoMetadata, options)
	expected := "https://github.com/test-org/test-repo/blob/main/main.py#L10"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

func TestBuildLocationLinkWithSubfolder(t *testing.T) {
	// Test the specific case mentioned in the issue: subfolder path resolution
	// Create a temporary directory structure to simulate the repository
	tempDir, err := os.MkdirTemp("", "sarif_test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoRoot := filepath.Join(tempDir, "scanio-test")
	subfolder := filepath.Join(repoRoot, "apps", "demo")
	if err := os.MkdirAll(subfolder, 0755); err != nil {
		t.Fatalf("failed to create subfolder: %v", err)
	}

	// Create the file so path resolution works correctly
	mainFile := filepath.Join(subfolder, "main.py")
	if err := os.WriteFile(mainFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	location := createTestLocation("main.py", 34, 1, 34, 10)

	repoMetadata := &git.RepositoryMetadata{
		RepoRootFolder: repoRoot,
		Subfolder:      "apps/demo",
		CommitHash:     stringPtr("aec0b795c350ff53fe9ab01adf862408aa34c3fd"),
	}

	options := MessageFormatOptions{
		Namespace:    "scan-io-git",
		Repository:   "scanio-test",
		Ref:          "aec0b795c350ff53fe9ab01adf862408aa34c3fd",
		SourceFolder: subfolder,
	}

	result := buildLocationLink(location, repoMetadata, options)
	expected := "https://github.com/scan-io-git/scanio-test/blob/aec0b795c350ff53fe9ab01adf862408aa34c3fd/apps/demo/main.py#L34"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

func TestFormatCodeQLStyleMessage(t *testing.T) {
	// Test CodeQL style message with direct markdown links
	message := &sarif.Message{
		Text: stringPtr("This template construction depends on a [user-provided value](1)."),
	}

	result := &sarif.Result{
		Message: *message,
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
	}

	repoMetadata := &git.RepositoryMetadata{
		RepoRootFolder: "/test/source",
		CommitHash:     stringPtr("abc123"),
	}

	options := MessageFormatOptions{
		Namespace:    "test-org",
		Repository:   "test-repo",
		Ref:          "main",
		SourceFolder: "/test/source",
	}

	formatted := FormatResultMessage(result, repoMetadata, options)
	expected := "This template construction depends on a [user-provided value](https://github.com/test-org/test-repo/blob/main/main.py#L1)."

	if formatted != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, formatted)
	}
}

// Helper functions for creating test data

func createTestLocation(uri string, startLine, startCol, endLine, endCol int) *sarif.Location {
	return &sarif.Location{
		PhysicalLocation: &sarif.PhysicalLocation{
			ArtifactLocation: &sarif.ArtifactLocation{
				URI: stringPtr(uri),
			},
			Region: &sarif.Region{
				StartLine:   intPtr(startLine),
				StartColumn: intPtr(startCol),
				EndLine:     intPtr(endLine),
				EndColumn:   intPtr(endCol),
			},
		},
	}
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func uintPtr(u uint) *uint {
	return &u
}
