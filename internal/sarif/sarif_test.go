package sarif

import (
	"testing"

	gosarif "github.com/owenrumney/go-sarif/v2/sarif"
)

func TestEnrichResultsLevelPropertyInitialisesResultProperties(t *testing.T) {
	ruleID := "CODEQL-0001"

	rule := &gosarif.ReportingDescriptor{
		ID: ruleID,
		Properties: gosarif.Properties{
			"problem.severity": "warning",
		},
	}

	result := &gosarif.Result{
		RuleID: &ruleID,
	}

	report := Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{
					Tool: gosarif.Tool{
						Driver: &gosarif.ToolComponent{
							Name:  "CodeQL",
							Rules: []*gosarif.ReportingDescriptor{rule},
						},
					},
					Results: []*gosarif.Result{result},
				},
			},
		},
	}

	report.EnrichResultsLevelProperty()

	if result.Properties == nil {
		t.Fatalf("expected result properties to be initialised, but it was nil")
	}

	level, ok := result.Properties["Level"]
	if !ok {
		t.Fatalf("expected Level property to be set on result properties")
	}

	if level != "warning" {
		t.Fatalf("expected Level property to be %q, got %v", "warning", level)
	}
}
