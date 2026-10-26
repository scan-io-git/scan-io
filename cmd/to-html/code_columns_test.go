package tohtml

import (
	"bytes"
	"strings"
	"testing"
	"time"

	gosarif "github.com/owenrumney/go-sarif/v2/sarif"
	scaniosarif "github.com/scan-io-git/scan-io/internal/sarif"
	scaniotemplate "github.com/scan-io-git/scan-io/internal/template"
)

func TestReportRendersSnippetWithColumnsOutsideSnippet(t *testing.T) {
	tmpl, err := scaniotemplate.NewTemplate("../../templates/tohtml/report.html")
	if err != nil {
		t.Fatalf("load template: %v", err)
	}

	snippet := `APIKey = "lp_live_FAKE51JXk2mQZTr9vCwNa8fH3sKp0dE7bY"`
	uri, ruleID := "src/app.go", "example-rule"
	line, startCol, endCol := 12, 21, 74
	loc := gosarif.NewLocationWithPhysicalLocation(&gosarif.PhysicalLocation{
		ArtifactLocation: gosarif.NewSimpleArtifactLocation(uri),
		Region: &gosarif.Region{
			StartLine: &line, EndLine: &line, StartColumn: &startCol, EndColumn: &endCol,
			Snippet: &gosarif.ArtifactContent{Text: &snippet},
		},
	})
	result := &gosarif.Result{RuleID: &ruleID, Locations: []*gosarif.Location{loc}}
	result.Properties = gosarif.Properties{"Severity": "high", "Title": "Example finding"}
	report := &scaniosarif.Report{Report: &gosarif.Report{
		Runs: []*gosarif.Run{{Results: []*gosarif.Result{result}}},
	}}
	report.EnrichResultsCodeFlowProperty(func(*gosarif.Location) string { return "" })

	data := struct {
		Metadata *ReportMetadata
		Report   *scaniosarif.Report
		CSP      cspData
	}{
		Metadata: &ReportMetadata{
			Title:           "Test",
			Time:            time.Now(),
			SeverityInfo:    map[string]int{"total": 1, "active": 1, "high": 1},
			SuppressionInfo: map[string]int{"total": 1, "active": 1},
		},
		Report: report,
		CSP:    cspData{Enabled: true, Nonce: "testnonce"},
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute template: %v", err)
	}
	if !strings.Contains(buf.String(), "lp_live_FAKE51JXk2mQZTr9vCwNa8fH3sKp0dE7bY") {
		t.Errorf("snippet text missing from rendered report")
	}
}
