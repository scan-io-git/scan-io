package tohtml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
)

const (
	loc      = `{"physicalLocation":{"artifactLocation":{"uri":"app/main.py"},"region":{"startLine":7,"endLine":7,"snippet":{"text":"run(user_input)"}}}}`
	rulesDoc = `[{"id":"py.known-rule","shortDescription":{"text":"Known rule title"}}]`
)

// sarifDoc wraps results JSON in a one-run document with the given rules JSON.
func sarifDoc(rules, results string) string {
	return `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"Example Scanner","rules":` + rules + `}},"results":` + results + `}]}`
}

// renderRaw runs the command on raw SARIF text with a repository provenance and a pull
// request, so permalinks and PR-diff anchors are produced. A panic is reported as a
// test failure instead of aborting the test binary.
func renderRaw(t *testing.T, doc string) (html string, err error) {
	t.Helper()
	html, _, err = renderRawTo(t, doc, filepath.Join(t.TempDir(), "out.html"))
	return html, err
}

func renderRawTo(t *testing.T, doc, outPath string) (html string, outExists bool, err error) {
	t.Helper()
	dir := t.TempDir()
	in := filepath.Join(dir, "in.sarif")
	if werr := os.WriteFile(in, []byte(doc), 0o644); werr != nil {
		t.Fatal(werr)
	}
	Init(nil, hclog.NewNullLogger())
	allToHTMLOptions = ToHTMLOptions{
		Inputs:       []string{in},
		OutputFile:   outPath,
		TempatesPath: "../../templates/tohtml",
		SourceFolder: t.TempDir(),
		PullRequest:  "7",
	}
	t.Setenv("GITHUB_REF", "")
	t.Setenv("CI_MERGE_REQUEST_IID", "")
	t.Setenv("BITBUCKET_PR_ID", "")
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("to-html panicked: %v", p)
		}
	}()
	err = ToHtmlCmd.RunE(ToHtmlCmd, nil)
	b, rerr := os.ReadFile(outPath)
	if rerr != nil {
		return "", false, err
	}
	return string(b), true, err
}

func TestMalformedResultsRenderWithoutPanic(t *testing.T) {
	const (
		pathChip   = `class="finding__path"`
		lineChip   = `class="finding__line"`
		locationEl = `class="finding__location"`
		codeLine   = `class="finding__df-codeline`
	)
	tests := []struct {
		name         string
		rules        string
		results      string
		wantTitle    string
		wantDesc     string
		wantLocation bool
	}{
		{
			name:         "result without ruleId",
			rules:        rulesDoc,
			results:      `[{"level":"error","message":{"text":"Untagged problem. Second sentence."},"locations":[` + loc + `]}]`,
			wantTitle:    "Untagged problem.",
			wantDesc:     "Untagged problem. Second sentence.",
			wantLocation: true,
		},
		{
			name:      "location without artifact uri",
			rules:     rulesDoc,
			results:   `[{"ruleId":"py.known-rule","message":{"text":"No uri here"},"locations":[{"physicalLocation":{"artifactLocation":{},"region":{"startLine":7,"snippet":{"text":"run(user_input)"}}}}]}]`,
			wantTitle: "Known rule title",
			wantDesc:  "No uri here",
		},
		{
			name:      "location without artifactLocation",
			rules:     rulesDoc,
			results:   `[{"ruleId":"py.known-rule","message":{"text":"No artifact"},"locations":[{"physicalLocation":{"region":{"startLine":7}}}]}]`,
			wantTitle: "Known rule title",
			wantDesc:  "No artifact",
		},
		{
			name:      "location without physicalLocation",
			rules:     rulesDoc,
			results:   `[{"ruleId":"py.known-rule","message":{"text":"No physical"},"locations":[{}]}]`,
			wantTitle: "Known rule title",
			wantDesc:  "No physical",
		},
		{
			name:      "location without region",
			rules:     rulesDoc,
			results:   `[{"ruleId":"py.known-rule","message":{"text":"No region"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"app/main.py"}}}]}]`,
			wantTitle: "Known rule title",
			wantDesc:  "No region",
		},
		{
			name:      "result without locations",
			rules:     rulesDoc,
			results:   `[{"ruleId":"py.known-rule","message":{"text":"Nowhere"}}]`,
			wantTitle: "Known rule title",
			wantDesc:  "Nowhere",
		},
		{
			name:      "result with empty locations",
			rules:     rulesDoc,
			results:   `[{"ruleId":"py.known-rule","message":{"text":"Nowhere"},"locations":[]}]`,
			wantTitle: "Known rule title",
			wantDesc:  "Nowhere",
		},
		{
			name:         "ruleId missing from driver rules",
			rules:        rulesDoc,
			results:      `[{"ruleId":"py.other-rule","message":{"text":"Orphan finding\nsecond line"},"locations":[` + loc + `]}]`,
			wantTitle:    "Orphan finding",
			wantDesc:     "Orphan finding\nsecond line",
			wantLocation: true,
		},
		{
			name:         "driver without rules",
			rules:        `[]`,
			results:      `[{"ruleId":"py.other-rule","message":{"text":"Rules array is empty"},"locations":[` + loc + `]}]`,
			wantTitle:    "Rules array is empty",
			wantDesc:     "Rules array is empty",
			wantLocation: true,
		},
		{
			name:         "rule without any description falls back to message",
			rules:        `[{"id":"py.bare-rule"}]`,
			results:      `[{"ruleId":"py.bare-rule","message":{"text":"Bare rule message"},"locations":[` + loc + `]}]`,
			wantTitle:    "Bare rule message",
			wantDesc:     "Bare rule message",
			wantLocation: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := renderRaw(t, sarifDoc(tt.rules, tt.results))
			if err != nil {
				t.Fatalf("RunE: %v", err)
			}
			mustContain(t, html, `<span class="finding__title" role="heading" aria-level="2" title="`+tt.wantTitle+`">`+tt.wantTitle+`</span>`)
			mustContain(t, html, `<p class="finding__description">`+tt.wantDesc+`</p>`)
			mustNotContain(t, html, "&lt;nil&gt;")
			mustNotContain(t, html, "&lt;no value&gt;")
			mustNotContain(t, html, "<nil>")
			for _, el := range []string{pathChip, locationEl, codeLine} {
				if got := strings.Contains(html, el); got != tt.wantLocation {
					t.Errorf("%s present = %v, want %v", el, got, tt.wantLocation)
				}
			}
			if tt.wantLocation {
				mustContain(t, html, `app/main.py</span><span class="finding__line">:7</span>`)
			} else {
				mustNotContain(t, html, lineChip)
				mustNotContain(t, html, "Location:")
				mustNotContain(t, html, "Location in PR:")
				mustNotContain(t, html, `class="finding__df-block"`)
			}
		})
	}
}

func TestResultWithoutRuleIDKeepsRuleFieldsOut(t *testing.T) {
	html, err := renderRaw(t, sarifDoc(rulesDoc, `[{"message":{"text":"Untagged"},"locations":[`+loc+`]}]`))
	if err != nil {
		t.Fatalf("RunE: %v", err)
	}
	mustContain(t, html, `data-rule-id=""`)
	mustNotContain(t, html, `<dt class="finding__meta-key">Rule:</dt>`)
}

func TestNoRunsRendersEmptyReport(t *testing.T) {
	for name, doc := range map[string]string{
		"empty runs array": `{"version":"2.1.0","runs":[]}`,
		"run without tool": `{"version":"2.1.0","runs":[{"results":[]}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			html, err := renderRaw(t, doc)
			if err != nil {
				t.Fatalf("RunE: %v", err)
			}
			mustContain(t, html, "</footer>")
			mustNotContain(t, html, `<details id="issue-`)
		})
	}
}

func TestRegionWithoutStartLineOmitsLine(t *testing.T) {
	noLine := `{"physicalLocation":{"artifactLocation":{"uri":"app/main.py"},"region":{"snippet":{"text":"run(user_input)"}}}}`
	doc := `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"Example Scanner"}},` +
		`"versionControlProvenance":[{"repositoryUri":"https://github.com/example/demo","revisionId":"0123456789abcdef0123456789abcdef01234567"}],` +
		`"results":[{"ruleId":"r","message":{"text":"No line"},"locations":[` + noLine + `]}]}]}`
	html, err := renderRaw(t, doc)
	if err != nil {
		t.Fatalf("RunE: %v", err)
	}
	mustContain(t, html, `<span class="finding__path">app/main.py</span>`)
	mustContain(t, html, `href="https://github.com/example/demo/blob/0123456789abcdef0123456789abcdef01234567/app/main.py"`)
	mustContain(t, html, `Location in PR: <strong><a href="https://github.com/example/demo/pull/7/files"`)
	for _, bad := range []string{`class="finding__line"`, "&lt;nil&gt;", "(line ", "app/main.py#L", "files#diff-"} {
		mustNotContain(t, html, bad)
	}
}

func TestSuppressedResultWithoutLocationRenders(t *testing.T) {
	doc := sarifDoc(rulesDoc, `[{"ruleId":"py.known-rule","message":{"text":"Hidden"},"suppressions":[{"kind":"inSource","status":"accepted"}]}]`)
	html, err := renderRaw(t, doc)
	if err != nil {
		t.Fatalf("RunE: %v", err)
	}
	mustContain(t, html, `data-suppressed="true"`)
	mustNotContain(t, html, `class="finding__path"`)
	mustNotContain(t, html, "&lt;nil&gt;")
}

func TestExplicitCodeFlowWithoutUsableLocationRenders(t *testing.T) {
	flow := `"codeFlows":[{"threadFlows":[{"locations":[{"location":{"physicalLocation":{"artifactLocation":{}}}},{"location":` + loc + `}]}]}]`
	doc := sarifDoc(rulesDoc, `[{"ruleId":"py.known-rule","message":{"text":"Flow"},"locations":[`+loc+`],`+flow+`}]`)
	html, err := renderRaw(t, doc)
	if err != nil {
		t.Fatalf("RunE: %v", err)
	}
	mustContain(t, html, "run(user_input)")
}

func TestFailedRenderLeavesNoOutputFile(t *testing.T) {
	tmplDir := t.TempDir()
	broken := "<html>partial output {{ index .Report.Runs 5 }}</html>"
	if err := os.WriteFile(filepath.Join(tmplDir, "report.html"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.html")
	Init(nil, hclog.NewNullLogger())
	in := filepath.Join(t.TempDir(), "in.sarif")
	if err := os.WriteFile(in, []byte(sarifDoc(rulesDoc, `[]`)), 0o644); err != nil {
		t.Fatal(err)
	}
	allToHTMLOptions = ToHTMLOptions{Inputs: []string{in}, OutputFile: out, TempatesPath: tmplDir, SourceFolder: t.TempDir()}
	if err := ToHtmlCmd.RunE(ToHtmlCmd, nil); err == nil {
		t.Fatal("expected a template error")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("output file exists after a failed render (stat err = %v)", err)
	}
}

func TestFailedRenderKeepsPreviousOutputFile(t *testing.T) {
	tmplDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmplDir, "report.html"), []byte("{{ index .Report.Runs 5 }}"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.html")
	if err := os.WriteFile(out, []byte("previous report"), 0o644); err != nil {
		t.Fatal(err)
	}
	Init(nil, hclog.NewNullLogger())
	in := filepath.Join(t.TempDir(), "in.sarif")
	if err := os.WriteFile(in, []byte(sarifDoc(rulesDoc, `[]`)), 0o644); err != nil {
		t.Fatal(err)
	}
	allToHTMLOptions = ToHTMLOptions{Inputs: []string{in}, OutputFile: out, TempatesPath: tmplDir, SourceFolder: t.TempDir()}
	if err := ToHtmlCmd.RunE(ToHtmlCmd, nil); err == nil {
		t.Fatal("expected a template error")
	}
	b, err := os.ReadFile(out)
	if err != nil || string(b) != "previous report" {
		t.Errorf("previous output was modified: %q, %v", b, err)
	}
}
