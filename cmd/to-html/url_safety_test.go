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

func TestReportNeutralizesDangerousURLSchemes(t *testing.T) {
	tmpl, err := scaniotemplate.NewTemplate("../../templates/tohtml/report.html")
	if err != nil {
		t.Fatalf("load template: %v", err)
	}

	startLine := 1

	artifactLocation := &gosarif.ArtifactLocation{
		PropertyBag: gosarif.PropertyBag{Properties: gosarif.Properties{"URI": "app.go"}},
	}

	location := &gosarif.Location{
		PhysicalLocation: &gosarif.PhysicalLocation{
			ArtifactLocation: artifactLocation,
			Region:           &gosarif.Region{StartLine: &startLine},
		},
		PropertyBag: gosarif.PropertyBag{Properties: gosarif.Properties{"WebURL": "javascript:alert(1)"}},
	}

	result := &gosarif.Result{
		Locations: []*gosarif.Location{location},
		PropertyBag: gosarif.PropertyBag{Properties: gosarif.Properties{
			"Severity":   "high",
			"Title":      "XSS probe",
			"References": []string{"javascript:alert(document.domain)"},
		}},
	}

	report := &scaniosarif.Report{
		Report: &gosarif.Report{
			Runs: []*gosarif.Run{{Results: []*gosarif.Result{result}}},
		},
	}

	data := struct {
		Metadata *ReportMetadata
		Report   *scaniosarif.Report
		CSP      cspData
	}{
		Metadata: &ReportMetadata{
			Title: "Test",
			Time:  time.Now(),
			SeverityInfo: map[string]int{
				"total":    1,
				"active":   1,
				"high":     1,
				"critical": 0,
				"medium":   0,
				"low":      0,
				"info":     0,
				"unknown":  0,
			},
			SuppressionInfo: map[string]int{
				"total":      1,
				"active":     1,
				"suppressed": 0,
			},
		},
		Report: report,
		CSP:    cspData{Enabled: true, Nonce: "testnonce"},
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute template: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, `href="javascript:`) {
		t.Fatalf("dangerous javascript: URL was rendered into an href")
	}
	if !strings.Contains(out, "#ZgotmplZ") {
		t.Fatalf("expected html/template to neutralize the dangerous URL to #ZgotmplZ")
	}
}
