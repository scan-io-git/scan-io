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

func TestEnrichResultsLevelPropertyHandlesMultipleRuns(t *testing.T) {
	ruleIDOne := "RULE-ONE"
	ruleIDTwo := "RULE-TWO"
	resultLevel := "note"

	runOneRule := gosarif.NewRule(ruleIDOne).WithProperties(gosarif.Properties{
		"problem.severity": "warning",
	})
	runTwoRule := gosarif.NewRule(ruleIDTwo)

	runOneResult := &gosarif.Result{
		RuleID: &ruleIDOne,
	}
	runTwoResult := &gosarif.Result{
		RuleID: &ruleIDTwo,
		Level:  &resultLevel,
	}

	report := Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{
					Tool: gosarif.Tool{
						Driver: &gosarif.ToolComponent{
							Name:  "ToolOne",
							Rules: []*gosarif.ReportingDescriptor{runOneRule},
						},
					},
					Results: []*gosarif.Result{runOneResult},
				},
				{
					Tool: gosarif.Tool{
						Driver: &gosarif.ToolComponent{
							Name:  "ToolTwo",
							Rules: []*gosarif.ReportingDescriptor{runTwoRule},
						},
					},
					Results: []*gosarif.Result{runTwoResult},
				},
			},
		},
	}

	report.EnrichResultsLevelProperty()

	if runOneResult.Properties == nil {
		t.Fatalf("expected runOneResult properties to be initialised")
	}
	if lvl := runOneResult.Properties["Level"]; lvl != "warning" {
		t.Fatalf("expected runOneResult level to be %q, got %v", "warning", lvl)
	}

	if runTwoResult.Properties == nil {
		t.Fatalf("expected runTwoResult properties to be initialised")
	}
	if lvl := runTwoResult.Properties["Level"]; lvl != "note" {
		t.Fatalf("expected runTwoResult level to be %q, got %v", "note", lvl)
	}
}

func TestEnrichResultsLevelPropertyUsesDefaultConfigurationLevel(t *testing.T) {
	ruleID := "RULE-DEFAULT"
	rule := gosarif.NewRule(ruleID)
	rule.DefaultConfiguration = gosarif.NewReportingConfiguration().WithLevel("error")

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
							Name:  "Tool",
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
		t.Fatalf("expected result properties to be initialised")
	}
	level, ok := result.Properties["Level"].(string)
	if !ok {
		t.Fatalf("expected Level property to be a string, got %T", result.Properties["Level"])
	}
	if level != "error" {
		t.Fatalf("expected Level property to be %q, got %q", "error", level)
	}
}
