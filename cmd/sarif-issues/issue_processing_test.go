package sarifissues

import "testing"

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
