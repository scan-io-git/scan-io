package sarif

import (
	"encoding/json"
	"os"
	"testing"

	gosarif "github.com/owenrumney/go-sarif/v2/sarif"
)

type triageCase struct {
	Name      string          `json:"name"`
	Required  bool            `json:"required"`
	Severity  string          `json:"severity"`
	FPVerdict *string         `json:"fpVerdict"`
	Triage    map[string]any  `json:"triage"`
	Expected  triageCaseWants `json:"expected"`
}

type triageCaseWants struct {
	Class    string  `json:"class"`
	Decision *string `json:"decision"`
}

type triageCaseFile struct {
	BlockerSeverities     []string     `json:"blockerSeverities"`
	NeverDemoteSeverities []string     `json:"neverDemoteSeverities"`
	Cases                 []triageCase `json:"cases"`
}

func toSet(items []string) map[string]bool {
	m := map[string]bool{}
	for _, s := range items {
		m[s] = true
	}
	return m
}

// classify runs the same enrichment chain to-html runs, on one result.
func classify(result *gosarif.Result, policy *RequiredPolicy) {
	rule := &gosarif.ReportingDescriptor{ID: "rule.test"}
	report := makeSimpleReport("rule.test", rule, result)
	report.EnrichResultsTriageProperty()
	if policy != nil {
		report.EnrichResultsRequiredProperty(*policy)
	}
}

func TestTriageClassificationCases(t *testing.T) {
	raw, err := os.ReadFile("testdata/triage-classification-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var file triageCaseFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Cases) == 0 {
		t.Fatal("case table is empty")
	}
	policy := RequiredPolicy{
		BlockerSeverities:     toSet(file.BlockerSeverities),
		NeverDemoteSeverities: toSet(file.NeverDemoteSeverities),
	}

	for _, tc := range file.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			result := resultFor("rule.test")
			result.Properties = map[string]any{"Severity": tc.Severity}
			if tc.FPVerdict != nil {
				result.Properties["fp"] = map[string]any{"verdict": *tc.FPVerdict}
			}
			if tc.Triage != nil {
				result.Properties["triage"] = tc.Triage
			}

			var p *RequiredPolicy
			if tc.Required {
				p = &policy
			}
			classify(result, p)

			if got := ResultClass(result); got != tc.Expected.Class {
				t.Errorf("class = %q, want %q", got, tc.Expected.Class)
			}
			wantDecision := ""
			if tc.Expected.Decision != nil {
				wantDecision = *tc.Expected.Decision
			}
			if got := ResultDecisionKind(result); got != wantDecision {
				t.Errorf("decision = %q, want %q", got, wantDecision)
			}
		})
	}
}

func TestTriage_UnknownValuesAreIgnored(t *testing.T) {
	for name, bag := range map[string]map[string]any{
		"unknown decision": {"decision": "wontfix", "status": "applied"},
		"unknown status":   {"decision": "false_positive", "status": "revoked"},
		"empty bag":        {},
	} {
		t.Run(name, func(t *testing.T) {
			result := resultFor("rule.test")
			result.Properties = map[string]any{"Severity": "high", "triage": bag}
			classify(result, &RequiredPolicy{BlockerSeverities: map[string]bool{"high": true}})

			if got := ResultClass(result); got != ClassRequired {
				t.Errorf("class = %q, want required (triage ignored)", got)
			}
			if got := ResultDecisionKind(result); got != "" {
				t.Errorf("decision = %q, want none", got)
			}
		})
	}
}

func TestTriage_SuppressedResultsAreUntouched(t *testing.T) {
	result := resultFor("rule.test")
	result.Properties = map[string]any{
		"Severity":   "high",
		"Suppressed": "true",
		"triage":     map[string]any{"decision": "false_positive", "status": "applied"},
	}
	classify(result, &RequiredPolicy{BlockerSeverities: map[string]bool{"high": true}})

	if got := ResultClass(result); got != ClassNone {
		t.Errorf("class = %q, want none for a suppressed result", got)
	}
}

func TestTriage_RequiredReasonForConfirmedFindings(t *testing.T) {
	cases := []struct {
		name    string
		verdict string
		want    string
	}{
		{"no verdict", "", "confirmed real by a reviewer"},
		{"false positive verdict", FPVerdictFalsePositive, "confirmed real by a reviewer, overrides the false-positive review"},
		{"true positive verdict", FPVerdictTruePositive, "confirmed real by a reviewer"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := resultFor("rule.test")
			result.Properties = map[string]any{
				"Severity": "low",
				"triage":   map[string]any{"decision": "true_positive", "status": "applied"},
			}
			if tc.verdict != "" {
				result.Properties["fp"] = map[string]any{"verdict": tc.verdict}
			}
			classify(result, &RequiredPolicy{BlockerSeverities: map[string]bool{"high": true}})

			if got, _ := result.Properties["RequiredReason"].(string); got != tc.want {
				t.Errorf("RequiredReason = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTriage_PRDecisionDisplayValues(t *testing.T) {
	result := resultFor("rule.test")
	result.Properties = map[string]any{
		"Severity": "high",
		"triage": map[string]any{
			"decision":      "accepted_risk",
			"status":        "applied",
			"author":        "alice",
			"approver":      "bob",
			"ticket":        "TKT-1",
			"reason":        "legacy endpoint",
			"source":        "https://example.com/pr/1#approval",
			"requestSource": "javascript:alert(1)",
			"decidedAt":     "2026-11-17T04:10:16Z",
		},
	}
	classify(result, nil)

	d, ok := result.Properties["PRDecision"].(map[string]any)
	if !ok {
		t.Fatal("PRDecision is missing")
	}
	want := map[string]string{
		"DecisionLabel": "Accepted risk",
		"Author":        "alice",
		"Approver":      "bob",
		"Ticket":        "TKT-1",
		"Reason":        "legacy endpoint",
		"SourceURL":     "https://example.com/pr/1#approval",
		"RequestURL":    "",
		"Date":          "17 Nov 2026",
	}
	for k, v := range want {
		if got, _ := d[k].(string); got != v {
			t.Errorf("PRDecision[%s] = %q, want %q", k, got, v)
		}
	}
}

func TestTriage_CountsAndOrdering(t *testing.T) {
	mk := func(sev string, bag map[string]any) *gosarif.Result {
		r := resultFor("rule.test")
		r.Properties = map[string]any{"Severity": sev}
		if bag != nil {
			r.Properties["triage"] = bag
		}
		return r
	}
	excluded := map[string]any{"decision": "false_positive", "status": "applied"}
	results := []*gosarif.Result{mk("critical", excluded), mk("low", nil), mk("high", nil), mk("high", excluded)}
	report := makeSimpleReport("rule.test", &gosarif.ReportingDescriptor{ID: "rule.test"}, results[0])
	report.Runs[0].Results = results
	policy := RequiredPolicy{BlockerSeverities: map[string]bool{"high": true}}

	report.EnrichResultsTriageProperty()
	report.EnrichResultsRequiredProperty(policy)

	if got := report.CollectSeverityInfo(); got["total"] != 2 || got["critical"] != 0 || got["high"] != 1 {
		t.Errorf("severity info must count active findings only, got %v", got)
	}
	if got := report.CollectRequiredInfo(); got["required"] != 1 || got["recommended"] != 1 {
		t.Errorf("required info = %v, want required 1 recommended 1", got)
	}
	if got := report.CollectExcludedInfo(); got != 2 {
		t.Errorf("excluded count = %d, want 2", got)
	}

	report.SortResultsByRequiredThenSeverity()
	var order []string
	for _, r := range report.Runs[0].Results {
		order = append(order, ResultClass(r)+"/"+r.Properties["Severity"].(string))
	}
	want := []string{"required/high", "recommended/low", "excluded/critical", "excluded/high"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %d results", order, len(want))
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}

func TestTriage_SortBySeverityPutsExcludedLast(t *testing.T) {
	mk := func(sev string, excluded bool) *gosarif.Result {
		r := resultFor("rule.test")
		r.Properties = map[string]any{"Severity": sev}
		if excluded {
			r.Properties["Excluded"] = "true"
		}
		return r
	}
	report := makeSimpleReport("rule.test", &gosarif.ReportingDescriptor{ID: "rule.test"}, mk("low", false))
	report.Runs[0].Results = []*gosarif.Result{mk("critical", true), mk("low", false), mk("high", false)}

	report.SortResultsBySeverity()

	got := []string{}
	for _, r := range report.Runs[0].Results {
		got = append(got, r.Properties["Severity"].(string))
	}
	if len(got) != 3 {
		t.Fatalf("order = %v, want 3 results", got)
	}
	if got[0] != "high" || got[1] != "low" || got[2] != "critical" {
		t.Errorf("order = %v, want [high low critical]", got)
	}
}
