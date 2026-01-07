package issuecorrelation

import "testing"

func TestCorrelator_SnippetHashMatch(t *testing.T) {
	known := []IssueMetadata{{Scanner: "s1", RuleID: "R1", SnippetHash: "h1"}}
	new := []IssueMetadata{{Scanner: "s1", RuleID: "R1", SnippetHash: "h1"}}

	c := NewCorrelator(new, known)
	c.Process()

	matches := c.Matches()
	if len(matches) != 1 {
		t.Fatalf("expected 1 match got %d", len(matches))
	}
	if len(matches[0].New) != 1 {
		t.Fatalf("expected 1 new in match got %d", len(matches[0].New))
	}

	if got := len(c.UnmatchedNew()); got != 0 {
		t.Fatalf("expected 0 unmatched new, got %d", got)
	}
	if got := len(c.UnmatchedKnown()); got != 0 {
		t.Fatalf("expected 0 unmatched known, got %d", got)
	}
}

func TestCorrelator_LineAndRuleMatch(t *testing.T) {
	known := []IssueMetadata{{Scanner: "s2", RuleID: "R2", Filename: "f.go", StartLine: 10, EndLine: 12}}
	new := []IssueMetadata{{Scanner: "s2", RuleID: "R2", Filename: "f.go", StartLine: 10, EndLine: 12}}

	c := NewCorrelator(new, known)
	c.Process()
	if len(c.Matches()) != 1 {
		t.Fatalf("expected match by lines/rule")
	}
}

func TestCorrelator_Unmatched(t *testing.T) {
	known := []IssueMetadata{{Scanner: "s3", RuleID: "R3", Filename: "x.go", StartLine: 1}}
	new := []IssueMetadata{{Scanner: "s4", RuleID: "R4", Filename: "y.go", StartLine: 2}}

	c := NewCorrelator(new, known)
	c.Process()

	if len(c.UnmatchedNew()) != 1 {
		t.Fatalf("expected 1 unmatched new")
	}
	if len(c.UnmatchedKnown()) != 1 {
		t.Fatalf("expected 1 unmatched known")
	}
	if len(c.Matches()) != 0 {
		t.Fatalf("expected 0 matches")
	}
}

func TestCorrelator_SameExceptLines(t *testing.T) {
	// known and new have identical scanner, ruleid, filename and snippethash
	// but different start and end lines -> should match at stage 2
	known := []IssueMetadata{{Scanner: "s5", RuleID: "R5", Filename: "g.go", StartLine: 10, EndLine: 12, SnippetHash: "sh5"}}
	new := []IssueMetadata{{Scanner: "s5", RuleID: "R5", Filename: "g.go", StartLine: 20, EndLine: 22, SnippetHash: "sh5"}}

	c := NewCorrelator(new, known)
	c.Process()

	matches := c.Matches()
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for same metadata except lines, got %d", len(matches))
	}
	if len(matches[0].New) != 1 {
		t.Fatalf("expected 1 new in match got %d", len(matches[0].New))
	}
	if got := len(c.UnmatchedNew()); got != 0 {
		t.Fatalf("expected 0 unmatched new, got %d", got)
	}
	if got := len(c.UnmatchedKnown()); got != 0 {
		t.Fatalf("expected 0 unmatched known, got %d", got)
	}
}

func TestCorrelator_KnownPlusSimilarNews(t *testing.T) {
	// One known issue. New issues include:
	// - an exact issue (same scanner, ruleid, filename, start/end and snippethash)
	// - a similar issue (same scanner, ruleid, filename, snippethash but different lines)
	// Because stage 1 runs before stage 2 and matches are excluded from later
	// stages, the known issue should match only the exact new issue and the
	// similar one should remain unmatched.
	known := []IssueMetadata{{Scanner: "sx", RuleID: "Rx", Filename: "h.go", StartLine: 5, EndLine: 7, SnippetHash: "shx"}}
	new := []IssueMetadata{
		{Scanner: "sx", RuleID: "Rx", Filename: "h.go", StartLine: 5, EndLine: 7, SnippetHash: "shx"},
		{Scanner: "sx", RuleID: "Rx", Filename: "h.go", StartLine: 50, EndLine: 52, SnippetHash: "shx"},
	}

	c := NewCorrelator(new, known)
	c.Process()

	matches := c.Matches()
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for known issue, got %d", len(matches))
	}
	if len(matches[0].New) != 1 {
		t.Fatalf("expected known to match only the exact new issue, got %d", len(matches[0].New))
	}

	// the similar new issue should be unmatched
	unmatchedNew := c.UnmatchedNew()
	if len(unmatchedNew) != 1 {
		t.Fatalf("expected 1 unmatched new issue (the similar one), got %d", len(unmatchedNew))
	}
	if len(c.UnmatchedKnown()) != 0 {
		t.Fatalf("expected 0 unmatched known issues, got %d", len(c.UnmatchedKnown()))
	}
}

func TestCorrelator_TwoPairsShiftedLines(t *testing.T) {
	// Two known issues. New issues are the same two but with start/end lines
	// shifted by +10 and identical SnippetHash. They should match via stage 2.
	known := []IssueMetadata{
		{Scanner: "sa", RuleID: "Ra", Filename: "p.go", StartLine: 1, EndLine: 3, SnippetHash: "sha"},
		{Scanner: "sb", RuleID: "Rb", Filename: "q.go", StartLine: 5, EndLine: 8, SnippetHash: "shb"},
	}
	new := []IssueMetadata{
		{Scanner: "sa", RuleID: "Ra", Filename: "p.go", StartLine: 11, EndLine: 13, SnippetHash: "sha"},
		{Scanner: "sb", RuleID: "Rb", Filename: "q.go", StartLine: 15, EndLine: 18, SnippetHash: "shb"},
	}

	c := NewCorrelator(new, known)
	c.Process()

	matches := c.Matches()
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches for the two pairs, got %d", len(matches))
	}

	if len(c.UnmatchedNew()) != 0 {
		t.Fatalf("expected 0 unmatched new issues, got %d", len(c.UnmatchedNew()))
	}
	if len(c.UnmatchedKnown()) != 0 {
		t.Fatalf("expected 0 unmatched known issues, got %d", len(c.UnmatchedKnown()))
	}
}
