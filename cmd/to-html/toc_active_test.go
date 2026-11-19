package tohtml

import (
	"encoding/json"
	"os/exec"
	"testing"
)

// activeCountsResult is what the node driver reports for the activeCounts region.
type activeCountsResult struct {
	IsActive   map[string]bool `json:"isActive"`
	TabCounts  map[string]int  `json:"tabCounts"`
	AllTab     summaryResult   `json:"allTab"`
	ScannerTab summaryResult   `json:"scannerTab"`
	NoSeverity int             `json:"noSeverity"`
}

type summaryResult struct {
	All         int            `json:"all"`
	Sev         map[string]int `json:"sev"`
	Required    int            `json:"required"`
	Recommended int            `json:"recommended"`
	Excluded    int            `json:"excluded"`
}

func runActiveCounts(t *testing.T) activeCountsResult {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed; skipping activeCounts JS unit test")
	}
	driver := extractJSRegion(t, "activeCounts") + `
const f = (cls, sev, scanner, suppressed) => ({ dataset: {
  classification: cls, severity: sev, scanner: scanner, suppressed: suppressed ? 'true' : undefined } });
const list = [
  f('required',    'high',   'A'),
  f('recommended', 'low',    'A'),
  f('required',    'high',   'B'),
  f('excluded',    'high',   'A'),
  f('excluded',    'medium', 'B'),
  f('excluded',    'medium', 'B'),
  f('',            'info',   'A', true),   // suppressed
  f('',            'medium', 'B'),         // active without classification (no --required)
  f('required',    undefined, 'B'),
];
process.stdout.write(JSON.stringify({
  isActive: {
    plain: isActive(f('', 'low', 'A')),
    required: isActive(f('required', 'low', 'A')),
    excluded: isActive(f('excluded', 'low', 'A')),
    suppressed: isActive(f('', 'low', 'A', true)),
    suppressedAndExcluded: isActive(f('excluded', 'low', 'A', true)),
  },
  tabCounts: { all: scannerCount(list, ''), A: scannerCount(list, 'A'), B: scannerCount(list, 'B') },
  allTab: summaryCounts(list, ''),
  scannerTab: summaryCounts(list, 'B'),
}));
`
	out, err := exec.Command(node, "-e", driver).CombinedOutput()
	if err != nil {
		t.Fatalf("node exec failed: %v\noutput: %s", err, out)
	}
	var res activeCountsResult
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("parse node output %q: %v", out, err)
	}
	return res
}

func TestIsActive_SuppressedAndExcludedAreNot(t *testing.T) {
	got := runActiveCounts(t).IsActive
	want := map[string]bool{"plain": true, "required": true, "excluded": false, "suppressed": false, "suppressedAndExcluded": false}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("isActive(%s) = %v, want %v", k, got[k], w)
		}
	}
}

func TestScannerCount_IgnoresExcludedAndSuppressed(t *testing.T) {
	got := runActiveCounts(t).TabCounts
	// A: required high, recommended low. B: required high, plain medium, required with no severity.
	want := map[string]int{"all": 5, "A": 2, "B": 3}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("scannerCount(%s) = %d, want %d", k, got[k], w)
		}
	}
}

func TestSummaryCounts_ExcludedHasItsOwnCount(t *testing.T) {
	res := runActiveCounts(t)

	all := res.AllTab
	if all.All != 5 || all.Required != 3 || all.Recommended != 1 || all.Excluded != 3 {
		t.Errorf("all tabs = %+v, want all 5 required 3 recommended 1 excluded 3", all)
	}
	if all.Sev["high"] != 2 || all.Sev["low"] != 1 || all.Sev["medium"] != 1 || all.Sev["info"] != 0 {
		t.Errorf("all tabs severity = %v, want high 2 low 1 medium 1 info 0 (excluded and suppressed are not counted)", all.Sev)
	}

	b := res.ScannerTab
	if b.All != 3 || b.Required != 2 || b.Recommended != 0 || b.Excluded != 2 {
		t.Errorf("scanner B tab = %+v, want all 3 required 2 recommended 0 excluded 2", b)
	}
}
