package sarif

import (
	"testing"
	"unicode/utf8"

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

func TestEnrichResultsTitleProperty_PrefersResultMessage(t *testing.T) {
	ruleID := "my-rule"
	ruleDesc := "`$1` has been added to the ignored list."
	resultMsg := "`node_modules/` has been added to the ignored list."
	shortDesc := "My rule title"

	rule := &gosarif.ReportingDescriptor{
		ID: ruleID,
		ShortDescription: &gosarif.MultiformatMessageString{
			Text: &shortDesc,
		},
		FullDescription: &gosarif.MultiformatMessageString{
			Text: &ruleDesc,
		},
	}
	result := &gosarif.Result{
		RuleID:  &ruleID,
		Message: *gosarif.NewMessage().WithText(resultMsg),
	}
	report := Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{
					Tool: gosarif.Tool{
						Driver: &gosarif.ToolComponent{
							Rules: []*gosarif.ReportingDescriptor{rule},
						},
					},
					Results: []*gosarif.Result{result},
				},
			},
		},
	}

	report.EnrichResultsTitleProperty()

	got, _ := result.Properties["Description"].(string)
	if got != resultMsg {
		t.Errorf("Description: want result message %q, got %q", resultMsg, got)
	}
	gotTitle, _ := result.Properties["Title"].(string)
	if gotTitle != shortDesc {
		t.Errorf("Title: want %q, got %q", shortDesc, gotTitle)
	}
}

func TestEnrichResultsTitleProperty_FallsBackToRuleDescription(t *testing.T) {
	ruleID := "my-rule"
	ruleDesc := "Rule full description text."

	rule := &gosarif.ReportingDescriptor{
		ID: ruleID,
		FullDescription: &gosarif.MultiformatMessageString{
			Text: &ruleDesc,
		},
	}
	result := &gosarif.Result{
		RuleID:  &ruleID,
		Message: gosarif.Message{},
	}
	report := Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{
					Tool: gosarif.Tool{
						Driver: &gosarif.ToolComponent{
							Rules: []*gosarif.ReportingDescriptor{rule},
						},
					},
					Results: []*gosarif.Result{result},
				},
			},
		},
	}

	report.EnrichResultsTitleProperty()

	got, _ := result.Properties["Description"].(string)
	if got != ruleDesc {
		t.Errorf("Description: want rule fallback %q, got %q", ruleDesc, got)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func makeSimpleReport(_ string, rule *gosarif.ReportingDescriptor, result *gosarif.Result) Report {
	return Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{
					Tool: gosarif.Tool{
						Driver: &gosarif.ToolComponent{
							Name:  "Test Scanner",
							Rules: []*gosarif.ReportingDescriptor{rule},
						},
					},
					Results: []*gosarif.Result{result},
				},
			},
		},
	}
}

func ruleWithTags(id string, tags ...string) *gosarif.ReportingDescriptor {
	tagIfaces := make([]any, len(tags))
	for i, t := range tags {
		tagIfaces[i] = t
	}
	return &gosarif.ReportingDescriptor{
		ID:         id,
		Properties: gosarif.Properties{"tags": tagIfaces},
	}
}

func resultFor(ruleID string) *gosarif.Result {
	return &gosarif.Result{RuleID: &ruleID}
}

// ── EnrichResultsCategoryProperty ────────────────────────────────────────────

func TestEnrichResultsCategoryProperty_CWESemgrep(t *testing.T) {
	id := "semgrep.use-after-free"
	rule := ruleWithTags(id, "CWE-416: Use After Free", "security")
	result := resultFor(id)
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsCategoryProperty()

	cat, _ := result.Properties["Category"].(string)
	if cat != "Memory safety" {
		t.Errorf("Category: want %q, got %q", "Memory safety", cat)
	}
	slug, _ := result.Properties["CategorySlug"].(string)
	if slug != "MEMORY_SAFETY" {
		t.Errorf("CategorySlug: want %q, got %q", "MEMORY_SAFETY", slug)
	}
}

func TestEnrichResultsCategoryProperty_CWECodeQL(t *testing.T) {
	id := "codeql.injection"
	rule := ruleWithTags(id, "external/cwe/cwe-089")
	result := resultFor(id)
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsCategoryProperty()

	cat, _ := result.Properties["Category"].(string)
	if cat != "Injection" {
		t.Errorf("Category: want %q, got %q", "Injection", cat)
	}
}

func TestEnrichResultsCategoryProperty_RuleIDFallback(t *testing.T) {
	id := "gitleaks.aws-access-token"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsCategoryProperty()

	cat, _ := result.Properties["Category"].(string)
	if cat != "Hardcoded secret" {
		t.Errorf("Category: want %q, got %q", "Hardcoded secret", cat)
	}
}

func TestEnrichResultsCategoryProperty_FallsBackToOther(t *testing.T) {
	id := "scanner.unknown-rule-xyz"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsCategoryProperty()

	cat, _ := result.Properties["Category"].(string)
	if cat != "Other" {
		t.Errorf("Category: want %q, got %q", "Other", cat)
	}
	slug, _ := result.Properties["CategorySlug"].(string)
	if slug != "OTHER" {
		t.Errorf("CategorySlug: want %q, got %q", "OTHER", slug)
	}
}

// ── EnrichResultsConfidenceProperty ──────────────────────────────────────────

func TestEnrichResultsConfidenceProperty_FloatPropOverridesTag(t *testing.T) {
	id := "rule.test"
	rule := ruleWithTags(id, "HIGH CONFIDENCE")
	result := resultFor(id)
	result.Properties = gosarif.Properties{"confidence": float64(0.42)}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsConfidenceProperty()

	got, _ := result.Properties["Confidence"].(string)
	if got != "Low (42%)" {
		t.Errorf("Confidence: want %q, got %q", "Low (42%)", got)
	}
}

func TestEnrichResultsConfidenceProperty_StringValueResolvedViaPrecisionMap(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = gosarif.Properties{"confidence": "high"}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsConfidenceProperty()

	got, _ := result.Properties["Confidence"].(string)
	if got != "High (85%)" {
		t.Errorf("Confidence: want %q, got %q", "High (85%)", got)
	}
}

func TestEnrichResultsConfidenceProperty_PrecisionFallback(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{
		ID:         id,
		Properties: gosarif.Properties{"precision": "medium"},
	}
	result := resultFor(id)
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsConfidenceProperty()

	got, _ := result.Properties["Confidence"].(string)
	if got != "Medium (65%)" {
		t.Errorf("Confidence: want %q, got %q", "Medium (65%)", got)
	}
}

func TestEnrichResultsConfidenceProperty_SemgrepTag(t *testing.T) {
	id := "rule.test"
	rule := ruleWithTags(id, "LOW CONFIDENCE", "security")
	result := resultFor(id)
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsConfidenceProperty()

	got, _ := result.Properties["Confidence"].(string)
	if got != "Low (40%)" {
		t.Errorf("Confidence: want %q, got %q", "Low (40%)", got)
	}
}

// ── extractReferencesFromMarkdown ─────────────────────────────────────────────

func TestExtractReferences_SemgrepHTMLForm(t *testing.T) {
	md := "Some description.\n\n<b>References:</b>\n - [CWE-416](https://cwe.mitre.org/data/definitions/416.html)\n"
	urls := extractReferencesFromMarkdown(md)
	if len(urls) != 1 || urls[0] != "https://cwe.mitre.org/data/definitions/416.html" {
		t.Errorf("want one URL, got %v", urls)
	}
}

func TestExtractReferences_DedupAndCap(t *testing.T) {
	md := "## References\nhttps://a.example/\nhttps://b.example/\nhttps://a.example/\nhttps://c.example/\nhttps://d.example/\n"
	result := resultFor("r")
	result.Properties = nil
	rule := &gosarif.ReportingDescriptor{
		ID: "r",
		Help: &gosarif.MultiformatMessageString{
			Markdown: strPtr(md),
		},
	}
	refs := extractReferences(result, rule, 3)
	if len(refs) != 3 {
		t.Errorf("want 3 refs, got %d: %v", len(refs), refs)
	}
	seen := map[string]bool{}
	for _, r := range refs {
		if seen[r] {
			t.Errorf("duplicate ref: %q", r)
		}
		seen[r] = true
	}
}

func TestExtractReferences_HelpUriFallback(t *testing.T) {
	result := resultFor("r")
	result.Properties = nil
	uri := "https://help.example/rule"
	rule := &gosarif.ReportingDescriptor{
		ID:      "r",
		HelpURI: &uri,
	}
	refs := extractReferences(result, rule, 3)
	if len(refs) != 1 || refs[0] != uri {
		t.Errorf("want helpUri %q, got %v", uri, refs)
	}
}

// ── extractFix ────────────────────────────────────────────────────────────────

func TestExtractFix_MarkdownSection(t *testing.T) {
	md := "Description.\n\n## Fix\nUse malloc before reuse.\n\n## References\nhttps://example.com/\n"
	fix := extractFixFromMarkdown(md)
	if fix != "Use malloc before reuse." {
		t.Errorf("Fix: got %q", fix)
	}
}

func TestExtractFix_RecommendationPropertyWins(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{
		ID: id,
		Help: &gosarif.MultiformatMessageString{
			Markdown: strPtr("## Fix\nFrom markdown."),
		},
	}
	result := resultFor(id)
	result.Properties = gosarif.Properties{"recommendation": "From property."}
	report := makeSimpleReport(id, rule, result)
	_ = report

	fix := extractFix(result, rule)
	if fix != "From property." {
		t.Errorf("Fix: want %q, got %q", "From property.", fix)
	}
}

func strPtr(s string) *string { return &s }

func TestHumanizeRuleID(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"python.lang.security.audit.dangerous-system-call", "Dangerous system call"},
		{"my_rule_id", "My rule id"},
		{"simple", "Simple"},
		{"x.y.z", "Z"},
		{"", ""},
		{"only-dots.", ""},
	}
	for _, tc := range cases {
		got := humanizeRuleID(tc.in)
		if got != tc.want {
			t.Errorf("humanizeRuleID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFirstSentence(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"SQL injection here. More details follow.", "SQL injection here."},
		{"No boundary here", ""},
		{"First. second not capital.", ""},
		{"One. Two. Three.", "One."},
		{"", ""},
	}
	for _, tc := range cases {
		got := firstSentence(tc.in)
		if got != tc.want {
			t.Errorf("firstSentence(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestResolveFindingTitle(t *testing.T) {
	makeResult := func(msg string) *gosarif.Result {
		if msg == "" {
			return &gosarif.Result{Message: gosarif.Message{}}
		}
		return &gosarif.Result{Message: *gosarif.NewMessage().WithText(msg)}
	}
	makeRule := func(id, shortDesc, name string) *gosarif.ReportingDescriptor {
		r := &gosarif.ReportingDescriptor{ID: id}
		if shortDesc != "" {
			r.ShortDescription = &gosarif.MultiformatMessageString{Text: strPtr(shortDesc)}
		}
		if name != "" {
			r.Name = strPtr(name)
		}
		return r
	}

	cases := []struct {
		name      string
		ruleID    string
		shortDesc string
		ruleName  string
		msg       string
		want      string
	}{
		{
			name:      "shortDesc clean",
			ruleID:    "weak-hash",
			shortDesc: "Use of weak hash",
			want:      "Use of weak hash",
		},
		{
			name:      "shortDesc contains ruleID short-circuits to humanize",
			ruleID:    "x.y.bad-call",
			shortDesc: "Semgrep rule: x.y.bad-call",
			want:      "Bad call",
		},
		{
			name:      "shortDesc empty falls through to name",
			ruleID:    "some-rule",
			ruleName:  "Dangerous System Call",
			want:      "Dangerous System Call",
		},
		{
			name:     "name contains ruleID short-circuits to humanize",
			ruleID:   "x.y.bad-call",
			ruleName: "x.y.bad-call",
			want:     "Bad call",
		},
		{
			name:   "message first sentence used when rule fields empty",
			ruleID: "ZZZQ-9999",
			msg:    "SQL injection here. More details follow.",
			want:   "SQL injection here.",
		},
		{
			name:   "message first line when no sentence boundary",
			ruleID: "ZZZQ-9999",
			msg:    "Just one line\nplus more",
			want:   "Just one line",
		},
		{
			name:   "all candidates empty falls back to humanize",
			ruleID: "X.y_test-thing",
			want:   "Y test thing",
		},
		{
			name: "ruleID and all candidates empty returns Finding",
			want: "Finding",
		},
		{
			name:      "120-char cap",
			ruleID:    "ZZZQ-9999",
			shortDesc: "A very long title that goes well beyond the hundred and twenty character limit imposed by the resolver to prevent pathological titles",
			want:      "A very long title that goes well beyond the hundred and twenty character limit imposed by the resolver to prevent pathol",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rule := makeRule(tc.ruleID, tc.shortDesc, tc.ruleName)
			result := makeResult(tc.msg)
			got := resolveFindingTitle(rule, result)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCap120MultibyteChars(t *testing.T) {
	// Build a string of 130 two-byte runes (é)
	runes := make([]rune, 130)
	for i := range runes {
		runes[i] = 'é'
	}
	input := string(runes)
	got := cap120(input)
	if utf8.RuneCountInString(got) != 120 {
		t.Errorf("cap120: want 120 runes, got %d", utf8.RuneCountInString(got))
	}
	// Verify no partial byte sequences
	if !utf8.ValidString(got) {
		t.Error("cap120: result is not valid UTF-8")
	}
}
