package tohtml

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// extractJSFunc pulls the source between the buildFileLabels markers in
// report.html so the test runs the exact code shipped in the report.
func extractJSFunc(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("../../templates/tohtml/report.html")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	const startMark = "// === buildFileLabels:start"
	const endMark = "// === buildFileLabels:end ==="
	s := string(src)
	i := strings.Index(s, startMark)
	j := strings.Index(s, endMark)
	if i < 0 || j < 0 || j < i {
		t.Fatalf("could not locate buildFileLabels markers in template")
	}
	// start at the line after the start marker, end at the line before end marker
	body := s[i:j]
	if nl := strings.IndexByte(body, '\n'); nl >= 0 {
		body = body[nl+1:]
	}
	return body
}

// runBuildFileLabels feeds inputJSON (a JSON array of path strings) to the
// extracted function via node and returns the resulting label map.
func runBuildFileLabels(t *testing.T, fn, inputJSON string) map[string]string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed; skipping buildFileLabels JS unit test")
	}
	driver := fn + `
const files = JSON.parse(process.argv[1]);
const ds = files.map(f => ({ file: f }));
const m = buildFileLabels(ds);
const obj = {};
for (const [k, v] of m) obj[k] = v;
process.stdout.write(JSON.stringify(obj));
`
	cmd := exec.Command(node, "-e", driver, inputJSON)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node exec failed: %v\noutput: %s", err, out)
	}
	var result map[string]string
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("parse node output %q: %v", out, err)
	}
	return result
}

func TestBuildFileLabels(t *testing.T) {
	fn := extractJSFunc(t)

	cases := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "collision shows one parent segment",
			input: `["src/foo/utils.go","src/bar/utils.go"]`,
			want: map[string]string{
				"src/foo/utils.go": "foo/utils.go",
				"src/bar/utils.go": "bar/utils.go",
			},
		},
		{
			name:  "no collision stays basename",
			input: `["a/main.go","b/handler.go"]`,
			want: map[string]string{
				"a/main.go":    "main.go",
				"b/handler.go": "handler.go",
			},
		},
		{
			name:  "deep collision grows to three segments",
			input: `["x/foo/utils.go","y/foo/utils.go"]`,
			want: map[string]string{
				"x/foo/utils.go": "x/foo/utils.go",
				"y/foo/utils.go": "y/foo/utils.go",
			},
		},
		{
			name:  "root file vs nested same basename",
			input: `["utils.go","a/utils.go"]`,
			want: map[string]string{
				"utils.go":   "utils.go",
				"a/utils.go": "a/utils.go",
			},
		},
		{
			name:  "empty path skipped",
			input: `["src/app.go",""]`,
			want: map[string]string{
				"src/app.go": "app.go",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runBuildFileLabels(t, fn, tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d entries, want %d: %v", len(got), len(tc.want), got)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("label for %q = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}
