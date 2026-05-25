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
		RuleID:  &ruleIDTwo,
		Level:   &resultLevel,
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

// --- new tests covering Severity and precedence ---

func makeReport(rule *gosarif.ReportingDescriptor, result *gosarif.Result) Report {
	var rules []*gosarif.ReportingDescriptor
	if rule != nil {
		rules = append(rules, rule)
	}
	return Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{
					Tool: gosarif.Tool{
						Driver: &gosarif.ToolComponent{
							Name:  "Tool",
							Rules: rules,
						},
					},
					Results: []*gosarif.Result{result},
				},
			},
		},
	}
}

func assertLevelSeverity(t *testing.T, result *gosarif.Result, wantLevel, wantSeverity string) {
	t.Helper()
	if result.Properties == nil {
		t.Fatal("result.Properties is nil")
	}
	if got, _ := result.Properties["Level"].(string); got != wantLevel {
		t.Errorf("Level: want %q, got %q", wantLevel, got)
	}
	if got, _ := result.Properties["Severity"].(string); got != wantSeverity {
		t.Errorf("Severity: want %q, got %q", wantSeverity, got)
	}
}

func TestEnrichResultsLevelPropertySecuritySeverityStringCritical(t *testing.T) {
	ruleID := "R"
	rule := gosarif.NewRule(ruleID).WithProperties(gosarif.Properties{
		"security-severity": "9.2",
	})
	result := &gosarif.Result{RuleID: &ruleID}
	r := makeReport(rule, result)
	r.EnrichResultsLevelProperty()
	assertLevelSeverity(t, result, "error", "critical")
}

func TestEnrichResultsLevelPropertySecuritySeverityFloatHigh(t *testing.T) {
	ruleID := "R"
	rule := gosarif.NewRule(ruleID).WithProperties(gosarif.Properties{
		"security-severity": float64(7.5),
	})
	result := &gosarif.Result{RuleID: &ruleID}
	r := makeReport(rule, result)
	r.EnrichResultsLevelProperty()
	assertLevelSeverity(t, result, "error", "high")
}

func TestEnrichResultsLevelPropertySecuritySeverityMedium(t *testing.T) {
	ruleID := "R"
	rule := gosarif.NewRule(ruleID).WithProperties(gosarif.Properties{
		"security-severity": "5.0",
	})
	result := &gosarif.Result{RuleID: &ruleID}
	r := makeReport(rule, result)
	r.EnrichResultsLevelProperty()
	assertLevelSeverity(t, result, "warning", "medium")
}

func TestEnrichResultsLevelPropertySecuritySeverityInfo(t *testing.T) {
	ruleID := "R"
	rule := gosarif.NewRule(ruleID).WithProperties(gosarif.Properties{
		"security-severity": "0.0",
	})
	result := &gosarif.Result{RuleID: &ruleID}
	r := makeReport(rule, result)
	r.EnrichResultsLevelProperty()
	assertLevelSeverity(t, result, "none", "info")
}

func TestEnrichResultsLevelPropertyProblemSeverityCritical(t *testing.T) {
	ruleID := "R"
	rule := gosarif.NewRule(ruleID).WithProperties(gosarif.Properties{
		"problem.severity": "CRITICAL",
	})
	result := &gosarif.Result{RuleID: &ruleID}
	r := makeReport(rule, result)
	r.EnrichResultsLevelProperty()
	assertLevelSeverity(t, result, "error", "critical")
}

func TestEnrichResultsLevelPropertyProblemSeverityInfo(t *testing.T) {
	ruleID := "R"
	rule := gosarif.NewRule(ruleID).WithProperties(gosarif.Properties{
		"problem.severity": "INFO",
	})
	result := &gosarif.Result{RuleID: &ruleID}
	r := makeReport(rule, result)
	r.EnrichResultsLevelProperty()
	assertLevelSeverity(t, result, "none", "info")
}

func TestEnrichResultsLevelPropertyResultLevelFallback(t *testing.T) {
	ruleID := "R"
	rule := gosarif.NewRule(ruleID) // no properties, no defaultConfig
	lvl := "warning"
	result := &gosarif.Result{RuleID: &ruleID, Level: &lvl}
	r := makeReport(rule, result)
	r.EnrichResultsLevelProperty()
	assertLevelSeverity(t, result, "warning", "medium")
}

func TestEnrichResultsLevelPropertyFallbackUnknown(t *testing.T) {
	ruleID := "R"
	rule := gosarif.NewRule(ruleID)
	result := &gosarif.Result{RuleID: &ruleID}
	r := makeReport(rule, result)
	r.EnrichResultsLevelProperty()
	assertLevelSeverity(t, result, "unknown", "unknown")
}

// security-severity should win over problem.severity.
func TestEnrichResultsLevelPropertySecuritySeverityWinsOverProblemSeverity(t *testing.T) {
	ruleID := "R"
	rule := gosarif.NewRule(ruleID).WithProperties(gosarif.Properties{
		"security-severity": "9.5",
		"problem.severity":  "warning",
	})
	result := &gosarif.Result{RuleID: &ruleID}
	r := makeReport(rule, result)
	r.EnrichResultsLevelProperty()
	assertLevelSeverity(t, result, "error", "critical")
}

// problem.severity should win over result.Level.
func TestEnrichResultsLevelPropertyProblemSeverityWinsOverResultLevel(t *testing.T) {
	ruleID := "R"
	rule := gosarif.NewRule(ruleID).WithProperties(gosarif.Properties{
		"problem.severity": "CRITICAL",
	})
	lvl := "error"
	result := &gosarif.Result{RuleID: &ruleID, Level: &lvl}
	r := makeReport(rule, result)
	r.EnrichResultsLevelProperty()
	assertLevelSeverity(t, result, "error", "critical")
}

// Already-set Level and Severity must not be overwritten (idempotence).
func TestEnrichResultsLevelPropertyIdempotent(t *testing.T) {
	ruleID := "R"
	rule := gosarif.NewRule(ruleID).WithProperties(gosarif.Properties{
		"problem.severity": "warning",
	})
	result := &gosarif.Result{RuleID: &ruleID}
	result.Properties = gosarif.Properties{
		"Level":    "error",
		"Severity": "critical",
	}
	r := makeReport(rule, result)
	r.EnrichResultsLevelProperty()
	assertLevelSeverity(t, result, "error", "critical")
}

// --- CollectSeverityInfo ---

func TestCollectSeverityInfo(t *testing.T) {
	mkResult := func(severity string) *gosarif.Result {
		r := &gosarif.Result{}
		r.Properties = gosarif.Properties{"Severity": severity}
		return r
	}

	report := Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{
					Results: []*gosarif.Result{
						mkResult("critical"),
						mkResult("critical"),
						mkResult("high"),
						mkResult("medium"),
						mkResult("medium"),
						mkResult("medium"),
						mkResult("low"),
						mkResult("info"),
						mkResult("unknown"),
					},
				},
			},
		},
	}

	counts := report.CollectSeverityInfo()

	cases := map[string]int{
		"critical": 2,
		"high":     1,
		"medium":   3,
		"low":      1,
		"info":     1,
		"unknown":  1,
		"total":    9,
	}
	for bucket, want := range cases {
		if got := counts[bucket]; got != want {
			t.Errorf("bucket %q: want %d, got %d", bucket, want, got)
		}
	}
}

func TestCollectSeverityInfoMissingSeverityCountsAsUnknown(t *testing.T) {
	// result with no Severity property should not panic and should count as unknown.
	result := &gosarif.Result{} // no Properties

	report := Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{Results: []*gosarif.Result{result}},
			},
		},
	}

	counts := report.CollectSeverityInfo()
	if counts["unknown"] != 1 {
		t.Errorf("want unknown=1, got %d", counts["unknown"])
	}
	if counts["total"] != 1 {
		t.Errorf("want total=1, got %d", counts["total"])
	}
}

// --- SortResultsBySeverity ---

func TestEnrichResultsLocationURIPropertyPRWebURL(t *testing.T) {
	uri := "src/main.go"
	makeReport := func() Report {
		al := gosarif.NewSimpleArtifactLocation(uri)
		al.Properties = gosarif.Properties{}
		loc := gosarif.NewLocationWithPhysicalLocation(
			&gosarif.PhysicalLocation{
				ArtifactLocation: al,
				Region:           &gosarif.Region{},
			},
		)
		loc.Properties = gosarif.Properties{}
		result := &gosarif.Result{}
		result.Locations = []*gosarif.Location{loc}
		return Report{
			Report: &gosarif.Report{
				Version: string(gosarif.Version210),
				Runs:    []*gosarif.Run{{Results: []*gosarif.Result{result}}},
			},
		}
	}

	webURL := "https://github.com/org/repo/blob/abc/src/main.go#L1"
	prURL := "https://github.com/org/repo/pull/42/files"

	t.Run("both callbacks set", func(t *testing.T) {
		report := makeReport()
		report.EnrichResultsLocationURIProperty(
			func(_ *gosarif.Location) string { return webURL },
			func(_ *gosarif.Location) string { return prURL },
		)
		loc := report.Runs[0].Results[0].Locations[0]
		if got, _ := loc.Properties["WebURL"].(string); got != webURL {
			t.Errorf("WebURL = %q, want %q", got, webURL)
		}
		if got, _ := loc.Properties["PRWebURL"].(string); got != prURL {
			t.Errorf("PRWebURL = %q, want %q", got, prURL)
		}
	})

	t.Run("nil prDiffURLCallback does not set PRWebURL", func(t *testing.T) {
		report := makeReport()
		report.EnrichResultsLocationURIProperty(
			func(_ *gosarif.Location) string { return webURL },
			nil,
		)
		loc := report.Runs[0].Results[0].Locations[0]
		if _, ok := loc.Properties["PRWebURL"]; ok {
			t.Error("PRWebURL should not be set when prDiffURLCallback is nil")
		}
	})
}

func TestSortResultsBySeverity(t *testing.T) {
	mkResult := func(severity string) *gosarif.Result {
		r := &gosarif.Result{}
		r.Properties = gosarif.Properties{"Severity": severity}
		return r
	}

	r1 := mkResult("low")
	r2 := mkResult("critical")
	r3 := mkResult("info")
	r4 := mkResult("high")
	r5 := mkResult("medium")

	report := Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{Results: []*gosarif.Result{r1, r2, r3, r4, r5}},
			},
		},
	}

	report.SortResultsBySeverity()

	want := []string{"critical", "high", "medium", "low", "info"}
	for i, want := range want {
		got, _ := report.Runs[0].Results[i].Properties["Severity"].(string)
		if got != want {
			t.Errorf("position %d: want %q, got %q", i, want, got)
		}
	}
}
