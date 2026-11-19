package tohtml

import (
	"strings"
	"testing"
)

func TestFindingID_ReplacesPositionalNumber(t *testing.T) {
	dir := t.TempDir()
	in := writeTestSARIF(t, dir, "in.sarif", nil,
		testResult{Rule: "demo.with-id", Props: map[string]any{"findingId": "F-3f9a2c1d"}},
		testResult{Rule: "demo.duplicate", Props: map[string]any{"findingId": "F-3f9a2c1d.81d0"}},
		testResult{Rule: "demo.no-id", Level: "note"},
	)
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir})

	mustContain(t, html, `<span class="finding__num finding__num--id" title="F-3f9a2c1d">F-3f9a2c1d</span>`)
	mustContain(t, html, `<span class="finding__num finding__num--id" title="F-3f9a2c1d.81d0">F-3f9a2c1d.81d0</span>`)
	mustContain(t, html, `data-finding-id="F-3f9a2c1d.81d0"`)
	mustContain(t, html, `<span class="finding__num">#3</span>`)
	mustNotContain(t, html, `<span class="finding__num">#1</span>`)
	if n := strings.Count(html, `data-finding-id=`); n != 2 {
		t.Errorf("data-finding-id appears %d times, want 2 (only findings that carry an ID)", n)
	}
}

func TestFindingID_AbsentKeepsPositionalNumbers(t *testing.T) {
	dir := t.TempDir()
	in := writeTestSARIF(t, dir, "in.sarif", nil, testResult{Rule: "demo.a"}, testResult{Rule: "demo.b"})
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir})

	mustContain(t, html, `<span class="finding__num">#1</span>`)
	mustContain(t, html, `<span class="finding__num">#2</span>`)
	mustNotContain(t, html, `finding__num--id"`)
	mustNotContain(t, html, `data-finding-id=`)
}

func TestFindingID_MarkupInIDIsEscaped(t *testing.T) {
	dir := t.TempDir()
	in := writeTestSARIF(t, dir, "in.sarif", nil,
		testResult{Rule: "demo.a", Props: map[string]any{"findingId": "<img src=x onerror=alert(1)>"}})
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir})

	mustNotContain(t, html, "<img src=x")
	mustContain(t, html, "&lt;img src=x onerror=alert(1)&gt;")
}

func TestFindingID_LongIDRendersInFullWithTitle(t *testing.T) {
	const long = "F-0123456789abcdef0123456789abcdef.81d0"
	dir := t.TempDir()
	in := writeTestSARIF(t, dir, "in.sarif", nil, testResult{Rule: "demo.a", Props: map[string]any{"findingId": long}})
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir})

	mustContain(t, html, `<span class="finding__num finding__num--id" title="`+long+`">`+long+`</span>`)
	mustContain(t, html, `data-finding-id="`+long+`"`)
}

func TestFindingID_WhitespaceOnlyAndNonStringFallBack(t *testing.T) {
	dir := t.TempDir()
	in := writeTestSARIF(t, dir, "in.sarif", nil,
		testResult{Rule: "demo.blank", Props: map[string]any{"findingId": "   "}},
		testResult{Rule: "demo.number", Props: map[string]any{"findingId": 42}},
	)
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir})

	mustContain(t, html, `<span class="finding__num">#1</span>`)
	mustContain(t, html, `<span class="finding__num">#2</span>`)
	mustNotContain(t, html, `data-finding-id=`)
}
