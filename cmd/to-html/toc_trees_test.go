package tohtml

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"testing"
)

// runTocTrees builds the three TOC trees for a fixed dataset by running the
// shipped tocTrees region under node, and reduces each tree to its row shape:
// "<depth>:<text>" per row, "hdr:<text>" per group header, "chips:<labels>" per
// chip strip and "sep" per separator. Depth is the tv-dN indent level, which is
// what makes a group read as nested.
func runTocTrees(t *testing.T) map[string][]string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed; skipping TOC tree JS unit test")
	}
	driver := `
let fileLabels = new Map();
let requiredEnabled = true;
` + extractJSRegion(t, "tocTrees") + `
const mk = (n, o) => Object.assign({
  id: 'issue-' + n, num: String(n), sev: 'high', title: 'Title ' + n, ruleId: 'rule.r' + n, ruleShort: 'r' + n,
  file: 'a/f' + n + '.py', line: '1', findingId: '', label: '#' + n, classification: 'recommended',
  scanner: 'S1', suppressed: false, excluded: false,
}, o);
const data = [
  mk(1, { classification: 'required' }),
  mk(2, { sev: 'medium' }),
  mk(3, { classification: 'excluded', excluded: true, scanner: 'S1', file: 'a/x.py', ruleId: 'rule.x', title: 'Excluded rule' }),
  mk(4, { classification: 'excluded', excluded: true, scanner: 'S1', file: 'a/x.py', ruleId: 'rule.x', title: 'Excluded rule' }),
  mk(5, { classification: 'excluded', excluded: true, scanner: 'S2', sev: 'low', file: 'a/y.py' }),
  mk(6, { classification: '', suppressed: true, scanner: 'S1', sev: 'low', file: 'a/s.py' }),
  mk(7, { classification: '', suppressed: true, scanner: 'S2', sev: 'medium', file: 'a/s2.py' }),
  mk(8, { scanner: 'S2', sev: 'low' }),
];
fileLabels = buildFileLabels(data);

function shape(html) {
  const chipStrips = [];
  html = html.replace(/<div class="tv-chips[^"]*"[^>]*>(.*?)<\/div>/g, (_, inner) => {
    chipStrips.push((inner.match(/>[^<]+<\/a>/g) || []).map(a => a.slice(1, -4)).join(','));
    return '<CHIPS/>';
  });
  const out = [];
  const re = /<CHIPS\/>|<div class="tv-sep"[^>]*><\/div>|<(?:div|a) class="(tv-prio-hdr[^"]*|tv-tr [^"]*)"[^>]*>(.*?)<\/(?:div|a)>/g;
  let m, ci = 0;
  while ((m = re.exec(html))) {
    if (m[0] === '<CHIPS/>') { out.push('chips:' + chipStrips[ci++]); continue; }
    if (m[0].startsWith('<div class="tv-sep"')) { out.push('sep'); continue; }
    const text = m[2].replace(/&#8856;/g, '\u2298').replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim();
    if (m[1].startsWith('tv-prio-hdr')) { out.push('hdr:' + text); continue; }
    out.push(m[1].match(/tv-d(\d)/)[1] + ':' + text);
  }
  return out;
}

process.stdout.write(JSON.stringify({
  sev: shape(buildSevTree(data, fileLabels)),
  rule: shape(buildRuleTree(data, fileLabels)),
  scanner: shape(buildScannerTree(data, fileLabels, ['S1', 'S2'])),
}));
`
	out, err := exec.Command(node, "-e", driver).CombinedOutput()
	if err != nil {
		t.Fatalf("node exec failed: %v\noutput: %s", err, out)
	}
	var res map[string][]string
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("parse node output %q: %v", out, err)
	}
	return res
}

// tail returns the rows from the first one equal to marker onward.
func tail(t *testing.T, rows []string, marker string) []string {
	t.Helper()
	for i, r := range rows {
		if r == marker {
			return rows[i:]
		}
	}
	t.Fatalf("no row %q in %v", marker, rows)
	return nil
}

func assertRows(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rows differ\n got: %q\nwant: %q", got, want)
	}
}

func TestTocTrees_BySeverity(t *testing.T) {
	rows := runTocTrees(t)["sev"]

	assertRows(t, tail(t, rows, "hdr:Excluded in this PR 3"), []string{
		"hdr:Excluded in this PR 3",
		"0:High 2", "1:x.py 2", "2:Excluded rule 2", "chips:#3,#4",
		"sep",
		"0:Low 1", "1:y.py 1", "2:Title 5 #5",
		"sep",
		"hdr:⊘ Suppressed 2",
		"0:Medium 1", "1:s2.py 1", "2:Title 7 #7",
		"sep",
		"0:Low 1", "1:s.py 1", "2:Title 6 #6",
	})
}

func TestTocTrees_ByFinding(t *testing.T) {
	rows := runTocTrees(t)["rule"]

	assertRows(t, tail(t, rows, "hdr:Excluded in this PR 3"), []string{
		"hdr:Excluded in this PR 3",
		"sep",
		"0:Excluded rule 2", "1:x.py 2", "chips:#3,#4",
		"sep",
		"0:Title 5 1", "1:y.py #5",
		"sep",
		"hdr:⊘ Suppressed 2",
		"sep",
		"0:Title 7 1", "1:s2.py #7",
		"sep",
		"0:Title 6 1", "1:s.py #6",
	})
}

func TestTocTrees_ByScanner(t *testing.T) {
	rows := runTocTrees(t)["scanner"]

	// Excluded: header, then scanner, severity, file, finding rows or chips, each a level deeper.
	assertRows(t, tail(t, rows, "hdr:Excluded in this PR 3"), []string{
		"hdr:Excluded in this PR 3",
		"1:S1 2",
		"2:High 2", "3:x.py 2", "4:Excluded rule 2", "chips:#3,#4",
		"sep",
		"1:S2 1",
		"2:Low 1", "3:y.py 1", "4:Title 5 #5",
		"sep",
		"hdr:⊘ Suppressed 2",
		"1:S1 1",
		"2:Low 1", "3:s.py 1", "4:Title 6 #6",
		"sep",
		"1:S2 1",
		"2:Medium 1", "3:s2.py 1", "4:Title 7 #7",
	})
}

func TestTocTrees_ActiveGroupsDoNotContainExcludedOrSuppressed(t *testing.T) {
	for mode, rows := range runTocTrees(t) {
		active := rows
		for i, r := range rows {
			if r == "hdr:Excluded in this PR 3" {
				active = rows[:i]
				break
			}
		}
		for _, r := range active {
			for _, banned := range []string{"Excluded rule", "Title 5", "Title 6", "Title 7"} {
				if contains(r, banned) {
					t.Errorf("%s tree: active rows contain %q: %v", mode, banned, active)
				}
			}
		}
	}
}
