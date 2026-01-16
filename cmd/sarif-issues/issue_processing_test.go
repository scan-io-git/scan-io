package sarifissues

import (
	"bytes"
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/internal/git"
	issuecorrelation "github.com/scan-io-git/scan-io/pkg/issuecorrelation"
)

func TestParseIssueBodyUsesMetadataRuleID(t *testing.T) {
	body := `
## 🐞 legacy-header

> **Rule ID**: semgrep.python.django.security
> **Severity**: High,  **Scanner**: Semgrep OSS
> **File**: app.py, **Lines**: 11-29

> **Snippet SHA256**: abcdef123456
`

	rep := parseIssueBody(body)
	if rep.RuleID != "semgrep.python.django.security" {
		t.Fatalf("expected RuleID from metadata, got %q", rep.RuleID)
	}
	if rep.Scanner != "Semgrep OSS" {
		t.Fatalf("expected scanner parsed from metadata, got %q", rep.Scanner)
	}
	if rep.Severity != "High" {
		t.Fatalf("expected severity parsed from metadata, got %q", rep.Severity)
	}
	if rep.FilePath != "app.py" {
		t.Fatalf("expected filepath parsed from metadata, got %q", rep.FilePath)
	}
	if rep.StartLine != 11 || rep.EndLine != 29 {
		t.Fatalf("expected start/end lines 11/29, got %d/%d", rep.StartLine, rep.EndLine)
	}
	if rep.Hash != "abcdef123456" {
		t.Fatalf("expected hash parsed from metadata, got %q", rep.Hash)
	}
}

func TestParseIssueBodyFallsBackToHeaderRuleID(t *testing.T) {
	body := `
## 🐞 fallback.rule.id

> **Severity**: High,  **Scanner**: Semgrep OSS
> **File**: main.py, **Lines**: 5-5
`

	rep := parseIssueBody(body)
	if rep.RuleID != "fallback.rule.id" {
		t.Fatalf("expected RuleID parsed from header fallback, got %q", rep.RuleID)
	}
}

func TestDisplayRuleTitleComponentPrefersName(t *testing.T) {
	name := "Descriptive Rule"
	rule := &sarif.ReportingDescriptor{
		Name: &name,
	}
	got := displayRuleTitleComponent("rule.id", rule)
	if got != "Descriptive Rule" {
		t.Fatalf("expected rule name for title component, got %q", got)
	}
}

func TestDisplayRuleTitleComponentFallsBackToID(t *testing.T) {
	got := displayRuleTitleComponent("rule.id", nil)
	if got != "rule.id" {
		t.Fatalf("expected rule id fallback for title component, got %q", got)
	}
}

func TestExtractRuleDetailPrefersHelpMarkdown(t *testing.T) {
	markdown := "Detailed explanation"
	rule := &sarif.ReportingDescriptor{
		Help: &sarif.MultiformatMessageString{
			Markdown: &markdown,
		},
	}

	detail, refs := extractRuleDetail(rule)
	if detail != "Detailed explanation" {
		t.Fatalf("expected help markdown detail, got %q", detail)
	}
	if len(refs) != 0 {
		t.Fatalf("expected no references for plain markdown, got %d", len(refs))
	}
}

func TestExtractRuleDetailFallsBackToFullDescription(t *testing.T) {
	full := "Full description text"
	rule := &sarif.ReportingDescriptor{
		FullDescription: &sarif.MultiformatMessageString{
			Text: &full,
		},
	}

	detail, refs := extractRuleDetail(rule)
	if detail != "Full description text" {
		t.Fatalf("expected full description fallback, got %q", detail)
	}
	if refs != nil {
		t.Fatalf("expected nil references for full description fallback, got %#v", refs)
	}
}

func TestExtractRuleDetailEmptyWhenNoContent(t *testing.T) {
	detail, refs := extractRuleDetail(nil)
	if detail != "" || refs != nil {
		t.Fatalf("expected empty detail and nil refs, got %q %#v", detail, refs)
	}
}

func TestFilterIssuesBySourceFolder(t *testing.T) {
	// Create test logger
	logger := hclog.NewNullLogger()

	// Test cases
	tests := []struct {
		name           string
		repoMetadata   *git.RepositoryMetadata
		openIssues     map[int]OpenIssueEntry
		expectedCount  int
		expectedIssues []int // issue numbers that should be included
	}{
		{
			name: "no subfolder - include all issues",
			repoMetadata: &git.RepositoryMetadata{
				Subfolder: "",
			},
			openIssues: map[int]OpenIssueEntry{
				1: {OpenIssueReport: OpenIssueReport{FilePath: "apps/demo/main.py"}},
				2: {OpenIssueReport: OpenIssueReport{FilePath: "apps/another/main.py"}},
				3: {OpenIssueReport: OpenIssueReport{FilePath: "root/file.py"}},
			},
			expectedCount:  3,
			expectedIssues: []int{1, 2, 3},
		},
		{
			name: "subfolder scope - filter correctly",
			repoMetadata: &git.RepositoryMetadata{
				Subfolder: "apps/demo",
			},
			openIssues: map[int]OpenIssueEntry{
				1: {OpenIssueReport: OpenIssueReport{FilePath: "apps/demo/main.py"}},
				2: {OpenIssueReport: OpenIssueReport{FilePath: "apps/another/main.py"}},
				3: {OpenIssueReport: OpenIssueReport{FilePath: "apps/demo/utils.py"}},
				4: {OpenIssueReport: OpenIssueReport{FilePath: "root/file.py"}},
			},
			expectedCount:  2,
			expectedIssues: []int{1, 3},
		},
		{
			name: "exact subfolder match",
			repoMetadata: &git.RepositoryMetadata{
				Subfolder: "apps/demo",
			},
			openIssues: map[int]OpenIssueEntry{
				1: {OpenIssueReport: OpenIssueReport{FilePath: "apps/demo"}},
				2: {OpenIssueReport: OpenIssueReport{FilePath: "apps/demo/main.py"}},
				3: {OpenIssueReport: OpenIssueReport{FilePath: "apps/demo/subdir/file.py"}},
			},
			expectedCount:  3,
			expectedIssues: []int{1, 2, 3},
		},
		{
			name:         "nil repo metadata - include all",
			repoMetadata: nil,
			openIssues: map[int]OpenIssueEntry{
				1: {OpenIssueReport: OpenIssueReport{FilePath: "any/path/file.py"}},
				2: {OpenIssueReport: OpenIssueReport{FilePath: "another/path/file.py"}},
			},
			expectedCount:  2,
			expectedIssues: []int{1, 2},
		},
		{
			name: "empty open issues",
			repoMetadata: &git.RepositoryMetadata{
				Subfolder: "apps/demo",
			},
			openIssues:     map[int]OpenIssueEntry{},
			expectedCount:  0,
			expectedIssues: []int{},
		},
		{
			name: "subfolder with trailing slashes",
			repoMetadata: &git.RepositoryMetadata{
				Subfolder: "/apps/demo/",
			},
			openIssues: map[int]OpenIssueEntry{
				1: {OpenIssueReport: OpenIssueReport{FilePath: "apps/demo/main.py"}},
				2: {OpenIssueReport: OpenIssueReport{FilePath: "apps/another/main.py"}},
			},
			expectedCount:  1,
			expectedIssues: []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterIssuesBySourceFolder(tt.openIssues, tt.repoMetadata, logger)

			if len(filtered) != tt.expectedCount {
				t.Fatalf("expected %d filtered issues, got %d", tt.expectedCount, len(filtered))
			}

			// Check that only expected issues are present
			for _, expectedNum := range tt.expectedIssues {
				if _, exists := filtered[expectedNum]; !exists {
					t.Fatalf("expected issue %d to be included in filtered results", expectedNum)
				}
			}

			// Check that no unexpected issues are present
			for num := range filtered {
				found := false
				for _, expectedNum := range tt.expectedIssues {
					if num == expectedNum {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("unexpected issue %d found in filtered results", num)
				}
			}
		})
	}
}

func TestCreateUnmatchedIssuesDryRun(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test data
	unmatchedNew := []issuecorrelation.IssueMetadata{
		{
			IssueID:     "test-1",
			Scanner:     "Semgrep",
			RuleID:      "sql-injection",
			Severity:    "error",
			Filename:    "app.py",
			StartLine:   11,
			EndLine:     29,
			SnippetHash: "abc123",
		},
		{
			IssueID:     "test-2",
			Scanner:     "Snyk",
			RuleID:      "xss-vulnerability",
			Severity:    "warning",
			Filename:    "main.js",
			StartLine:   5,
			EndLine:     5,
			SnippetHash: "def456",
		},
	}

	newIssues := []issuecorrelation.IssueMetadata{
		unmatchedNew[0], unmatchedNew[1],
	}

	newBodies := []string{
		"Test body for issue 1",
		"Test body for issue 2",
	}

	newTitles := []string{
		"[Semgrep][High][sql-injection] at app.py:11-29",
		"[Snyk][Medium][xss-vulnerability] at main.js:5",
	}

	options := RunOptions{
		DryRun: true,
	}

	logger := hclog.NewNullLogger()

	// Test dry-run mode
	created, err := createUnmatchedIssues(unmatchedNew, newIssues, newBodies, newTitles, options, logger)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify results
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created != 2 {
		t.Fatalf("expected 2 created issues, got %d", created)
	}

	// Check output contains expected dry-run information
	expectedOutputs := []string{
		"[DRY RUN] Would create issue:",
		"Title: [Semgrep][High][sql-injection] at app.py:11-29",
		"File: app.py",
		"Lines: 11-29",
		"Severity: High",
		"Scanner: Semgrep",
		"Rule ID: sql-injection",
		"Title: [Snyk][Medium][xss-vulnerability] at main.js:5",
		"File: main.js",
		"Lines: 5",
		"Severity: Medium",
		"Scanner: Snyk",
		"Rule ID: xss-vulnerability",
	}

	for _, expected := range expectedOutputs {
		if !bytes.Contains(buf.Bytes(), []byte(expected)) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestCloseUnmatchedIssuesDryRun(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test data
	unmatchedKnown := []issuecorrelation.IssueMetadata{
		{
			IssueID:     "42",
			Scanner:     "Semgrep",
			RuleID:      "deprecated-rule",
			Severity:    "error",
			Filename:    "old-file.py",
			StartLine:   5,
			EndLine:     10,
			SnippetHash: "xyz789",
		},
		{
			IssueID:     "123",
			Scanner:     "Snyk",
			RuleID:      "old-vulnerability",
			Severity:    "warning",
			Filename:    "legacy.js",
			StartLine:   15,
			EndLine:     15,
			SnippetHash: "abc123",
		},
	}

	options := RunOptions{
		DryRun: true,
	}

	logger := hclog.NewNullLogger()

	// Test dry-run mode
	closed, err := closeUnmatchedIssues(unmatchedKnown, options, logger)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify results
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if closed != 2 {
		t.Fatalf("expected 2 closed issues, got %d", closed)
	}

	// Check output contains expected dry-run information
	expectedOutputs := []string{
		"[DRY RUN] Would close issue #42:",
		"File: old-file.py",
		"Lines: 5-10",
		"Rule ID: deprecated-rule",
		"Reason: Not found in current scan",
		"[DRY RUN] Would close issue #123:",
		"File: legacy.js",
		"Lines: 15",
		"Rule ID: old-vulnerability",
	}

	for _, expected := range expectedOutputs {
		if !bytes.Contains(buf.Bytes(), []byte(expected)) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}
}
