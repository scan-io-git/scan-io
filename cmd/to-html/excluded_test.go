package tohtml

import (
	"regexp"
	"strings"
	"testing"
)

func excludedFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	excluded := map[string]any{"decision": "false_positive", "status": "applied", "author": "alice", "reason": "test data"}
	return writeTestSARIF(t, dir, "in.sarif", nil,
		testResult{Rule: "demo.required", Level: "high", Props: map[string]any{"findingId": "F-00000001"}},
		testResult{Rule: "demo.recommended", Level: "low", Props: map[string]any{"findingId": "F-00000002"}},
		testResult{Rule: "demo.excluded-critical", Level: "critical", Props: map[string]any{"findingId": "F-00000003", "triage": excluded}},
		testResult{Rule: "demo.excluded-low", Level: "low", Props: map[string]any{"findingId": "F-00000004", "triage": excluded}},
	)
}

func pillCount(t *testing.T, html, filter string) string {
	t.Helper()
	re := regexp.MustCompile(`data-filter="` + filter + `"[^>]*>.*?<span class="pill__n">(\d+)</span>`)
	m := re.FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("no %q pill in report", filter)
	}
	return m[1]
}

func TestExcluded_SectionOrderPillAndCounts(t *testing.T) {
	in := excludedFixture(t)
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: t.TempDir(), Required: "high"})

	req := strings.Index(html, `data-section="required"`)
	rec := strings.Index(html, `data-section="recommended"`)
	exc := strings.Index(html, `data-section="excluded"`)
	if req < 0 || rec < req || exc < rec {
		t.Fatalf("section order wrong: required=%d recommended=%d excluded=%d", req, rec, exc)
	}
	mustContain(t, html, `aria-expanded="false" aria-controls="excluded-section-content"`)
	mustContain(t, html, `id="excluded-section-content" class="findings-section-content section-collapsed"`)
	mustContain(t, html, `Excluded in this PR`)

	if got := pillCount(t, html, "all"); got != "2" {
		t.Errorf("All pill = %s, want 2 (excluded findings are not active)", got)
	}
	if got := pillCount(t, html, "critical"); got != "0" {
		t.Errorf("Critical pill = %s, want 0 (the only critical finding is excluded)", got)
	}
	if got := pillCount(t, html, "excluded"); got != "2" {
		t.Errorf("Excluded pill = %s, want 2", got)
	}
	if got := pillCount(t, html, "required"); got != "1" {
		t.Errorf("Required pill = %s, want 1", got)
	}
	if got := pillCount(t, html, "recommended"); got != "1" {
		t.Errorf("Recommended pill = %s, want 1", got)
	}
	if n := strings.Count(html, `data-classification="excluded"`); n != 2 {
		t.Errorf("excluded cards = %d, want 2", n)
	}
	mustContain(t, html, `<span class="findings-section__count">2</span>`)
}

func TestExcluded_AppliesWithoutRequired(t *testing.T) {
	in := excludedFixture(t)
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: t.TempDir()})

	mustContain(t, html, `data-section="excluded"`)
	mustContain(t, html, `<button class="pill pill--excluded"`)
	mustNotContain(t, html, `<button class="pill pill--required"`)
	mustNotContain(t, html, `data-section="required"`)
	if got := pillCount(t, html, "all"); got != "2" {
		t.Errorf("All pill = %s, want 2", got)
	}
}

func TestExcluded_AbsentWithoutDecisions(t *testing.T) {
	dir := t.TempDir()
	in := writeTestSARIF(t, dir, "in.sarif", nil, testResult{Rule: "demo.a"}, testResult{Rule: "demo.b", Level: "low"})
	for _, required := range []string{"", "high"} {
		html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir, Required: required})
		mustNotContain(t, html, `<button class="pill pill--excluded"`)
		mustNotContain(t, html, `data-section="excluded"`)
		mustNotContain(t, html, `id="tb-circle-off"`)
		mustNotContain(t, html, `data-classification="excluded"`)
	}
}
