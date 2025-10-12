package sarifissues

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/internal/git"
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
