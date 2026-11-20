package tohtml

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
)

// testResult is one synthetic SARIF result. Props is merged into result.properties.
type testResult struct {
	Rule  string
	Level string
	URI   string
	Line  int
	Props map[string]any
}

func (r testResult) toMap() map[string]any {
	if r.Level == "" {
		r.Level = "error"
	}
	if r.URI == "" {
		r.URI = "app/main.py"
	}
	if r.Line == 0 {
		r.Line = 10
	}
	m := map[string]any{
		"ruleId":  r.Rule,
		"level":   r.Level,
		"message": map[string]any{"text": "Synthetic finding for " + r.Rule},
		"locations": []any{map[string]any{"physicalLocation": map[string]any{
			"artifactLocation": map[string]any{"uri": r.URI},
			"region": map[string]any{
				"startLine": r.Line, "endLine": r.Line,
				"snippet": map[string]any{"text": "do_something(user_input)"},
			},
		}}},
	}
	if len(r.Props) > 0 {
		m["properties"] = r.Props
	}
	return m
}

// writeTestSARIF writes a one-run SARIF holding the results (plus optional
// versionControlProvenance) and returns its path.
func writeTestSARIF(t *testing.T, dir, name string, provenance []map[string]any, results ...testResult) string {
	t.Helper()
	rs := make([]any, 0, len(results))
	for _, r := range results {
		rs = append(rs, r.toMap())
	}
	run := map[string]any{
		"tool":    map[string]any{"driver": map[string]any{"name": "Example Scanner", "semanticVersion": "1.0.0"}},
		"results": rs,
	}
	if provenance != nil {
		run["versionControlProvenance"] = provenance
	}
	doc := map[string]any{"version": "2.1.0", "runs": []any{run}}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// renderReport runs the command against the shipped template and returns the HTML.
func renderReport(t *testing.T, opts ToHTMLOptions) string {
	t.Helper()
	Init(nil, hclog.NewNullLogger())
	dir := t.TempDir()
	opts.OutputFile = filepath.Join(dir, "out.html")
	opts.TempatesPath = "../../templates/tohtml"
	allToHTMLOptions = opts
	t.Setenv("GITHUB_REF", "")
	t.Setenv("CI_MERGE_REQUEST_IID", "")
	t.Setenv("BITBUCKET_PR_ID", "")
	if err := ToHtmlCmd.RunE(ToHtmlCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	b, err := os.ReadFile(opts.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func mustContain(t *testing.T, html, sub string) {
	t.Helper()
	if !contains(html, sub) {
		t.Errorf("rendered report is missing %q", sub)
	}
}

func mustNotContain(t *testing.T, html, sub string) {
	t.Helper()
	if contains(html, sub) {
		t.Errorf("rendered report unexpectedly contains %q", sub)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func provenanceEntry(repo, rev, branch string) []map[string]any {
	return []map[string]any{{"repositoryUri": repo, "revisionId": rev, "branch": branch}}
}
