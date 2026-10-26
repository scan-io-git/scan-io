package tohtml

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// extractJSRegion pulls the source between the named markers in report.html so
// tests run the exact code shipped in the report.
func extractJSRegion(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile("../../templates/tohtml/report.html")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	startMark := "// === " + name + ":start"
	endMark := "// === " + name + ":end ==="
	s := string(src)
	i := strings.Index(s, startMark)
	j := strings.Index(s, endMark)
	if i < 0 || j < 0 || j < i {
		t.Fatalf("could not locate %s markers in template", name)
	}
	// start at the line after the start marker, end at the line before end marker
	body := s[i:j]
	if nl := strings.IndexByte(body, '\n'); nl >= 0 {
		body = body[nl+1:]
	}
	return body
}

// chipStripResult is the shape the node driver reports back for one strip.
type chipStripResult struct {
	Anchors      int      `json:"anchors"`
	Overflow     int      `json:"overflow"`
	Buttons      int      `json:"buttons"`
	MoreCount    string   `json:"moreCount"`
	AriaExpanded bool     `json:"ariaExpanded"`
	Order        []string `json:"order"`
	StripClass   string   `json:"stripClass"`
	ParentAttr   string   `json:"parentAttr"`
}

// runRenderChipStrip renders a strip of n findings via the extracted template
// function and reports structural facts about the emitted HTML. Findings are
// fed in descending order so the caller can assert the strip re-sorts them.
func runRenderChipStrip(t *testing.T, fn string, n int, chipClass string) chipStripResult {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed; skipping renderChipStrip JS unit test")
	}
	driver := `
const esc = s => String(s)
  .replace(/&/g, '&amp;').replace(/</g, '&lt;')
  .replace(/>/g, '&gt;').replace(/"/g, '&quot;');
` + fn + `
const n = parseInt(process.argv[1], 10);
const chipClass = process.argv[2];
// Descending input: the strip is responsible for sorting chips ascending.
const items = Array.from({ length: n }, (_, i) => ({
  id: 'issue-' + (n - i), num: String(n - i), line: 10 + i, file: 'a.py',
}));
const html = renderChipStrip(items, {
  parents: 'file-1 sev-1',
  chipClass: chipClass,
  titleFor: d => esc('rule') + ' (' + esc(d.file) + ':' + esc(d.line) + ')',
  classFor: () => '',
});
const stripOpen = (html.match(/^<div class="([^"]*)"([^>]*)>/) || ['', '', '']);
process.stdout.write(JSON.stringify({
  anchors: (html.match(/<a /g) || []).length,
  overflow: (html.match(/tv-chip--overflow/g) || []).length,
  buttons: (html.match(/<button /g) || []).length,
  moreCount: ((html.match(/>\+ (\d+) more</) || [])[1]) || '',
  ariaExpanded: /aria-expanded="false"/.test(html),
  order: (html.match(/>#(\d+)</g) || []).map(s => s.replace(/\D/g, '')),
  stripClass: stripOpen[1],
  parentAttr: stripOpen[2].trim(),
}));
`
	cmd := exec.Command(node, "-e", driver, fmt.Sprint(n), chipClass)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node exec failed: %v\noutput: %s", err, out)
	}
	var result chipStripResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("parse node output %q: %v", out, err)
	}
	return result
}

// chipCap mirrors CHIP_CAP in report.html. A cluster larger than this strands
// chips outside the strip's max-height unless they are hidden behind the
// "+N more" button, which is the bug these tests guard.
const chipCap = 48

func TestRenderChipStripCap(t *testing.T) {
	fn := extractJSRegion(t, "renderChipStrip")

	cases := []struct {
		name         string
		items        int
		wantOverflow int
		wantButtons  int
		wantMore     string
	}{
		{name: "small cluster renders every chip", items: 3, wantOverflow: 0, wantButtons: 0, wantMore: ""},
		{name: "cluster at the cap has no button", items: chipCap, wantOverflow: 0, wantButtons: 0, wantMore: ""},
		{name: "one past the cap hides one chip", items: chipCap + 1, wantOverflow: 1, wantButtons: 1, wantMore: "1"},
		{name: "large cluster hides the tail", items: 925, wantOverflow: 925 - chipCap, wantButtons: 1, wantMore: "877"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runRenderChipStrip(t, fn, tc.items, "")

			if got.Anchors != tc.items {
				t.Errorf("anchors = %d, want %d (every finding must stay reachable)", got.Anchors, tc.items)
			}
			if got.Overflow != tc.wantOverflow {
				t.Errorf("overflow chips = %d, want %d", got.Overflow, tc.wantOverflow)
			}
			if got.Buttons != tc.wantButtons {
				t.Errorf("more buttons = %d, want %d", got.Buttons, tc.wantButtons)
			}
			if got.MoreCount != tc.wantMore {
				t.Errorf("more button count = %q, want %q", got.MoreCount, tc.wantMore)
			}
			if tc.wantButtons > 0 && !got.AriaExpanded {
				t.Error("more button must start with aria-expanded=\"false\"")
			}
		})
	}
}

func TestRenderChipStripSortsAscending(t *testing.T) {
	fn := extractJSRegion(t, "renderChipStrip")
	got := runRenderChipStrip(t, fn, 60, "")

	if len(got.Order) != 60 {
		t.Fatalf("chip count = %d, want 60", len(got.Order))
	}
	for i, num := range got.Order {
		if want := fmt.Sprint(i + 1); num != want {
			t.Fatalf("chip %d = #%s, want #%s (chips must sort by finding number)", i, num, want)
		}
	}
}

func TestRenderChipStripKeepsStripAttributes(t *testing.T) {
	fn := extractJSRegion(t, "renderChipStrip")
	got := runRenderChipStrip(t, fn, 2, "tv-grp--suppressed")

	if !strings.Contains(got.StripClass, "tv-chips") {
		t.Errorf("strip class = %q, want it to contain tv-chips", got.StripClass)
	}
	if !strings.Contains(got.StripClass, "tv-grp--suppressed") {
		t.Errorf("strip class = %q, want the caller's chipClass preserved", got.StripClass)
	}
	if !strings.Contains(got.ParentAttr, `data-grp-parent="file-1 sev-1"`) {
		t.Errorf("strip attrs = %q, want the collapse parents preserved", got.ParentAttr)
	}
}
