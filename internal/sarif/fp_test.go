package sarif

import (
	"testing"

	gosarif "github.com/owenrumney/go-sarif/v2/sarif"
)

func TestEnrichResultsFPProperty_RecognizedVerdict(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"fp": map[string]any{
			"verdict":   "NEEDS_VERIFICATION",
			"p_real":    0.5,
			"reasoning": "Could not confirm exploitability from the available context.",
		},
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsFPProperty()

	if got, _ := result.Properties["FPVerdict"].(string); got != FPVerdictNeedsVerification {
		t.Errorf("FPVerdict = %q, want %q", got, FPVerdictNeedsVerification)
	}
	if got, _ := result.Properties["FPLabel"].(string); got != "Needs verification" {
		t.Errorf("FPLabel = %q, want %q", got, "Needs verification")
	}
	wantHover := "Automated false-positive review couldn't confirm whether this is real. Verify it by hand before merging."
	if got, _ := result.Properties["FPHover"].(string); got != wantHover {
		t.Errorf("FPHover = %q, want %q", got, wantHover)
	}
	wantReasoning := "Could not confirm exploitability from the available context."
	if got, _ := result.Properties["FPReasoning"].(string); got != wantReasoning {
		t.Errorf("FPReasoning = %q, want %q", got, wantReasoning)
	}
}

func TestEnrichResultsFPProperty_TruePositiveHasNoHover(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"fp": map[string]any{"verdict": "TRUE_POSITIVE", "reasoning": "Confirmed via the dataflow."},
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsFPProperty()

	if got, _ := result.Properties["FPLabel"].(string); got != "Confirmed as a real issue" {
		t.Errorf("FPLabel = %q, want %q", got, "Confirmed as a real issue")
	}
	if _, ok := result.Properties["FPHover"]; ok {
		t.Error("FPHover must be omitted for TRUE_POSITIVE")
	}
}

func TestEnrichResultsFPProperty_NoBag_NoProjection(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{"Severity": "high"}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsFPProperty()

	for _, key := range []string{"FPVerdict", "FPLabel", "FPHover", "FPReasoning"} {
		if _, ok := result.Properties[key]; ok {
			t.Errorf("Properties[%q] must be unset with no fp bag, got %v", key, result.Properties[key])
		}
	}
}

func TestEnrichResultsFPProperty_EvidenceNeverProjected(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"fp": map[string]any{
			"verdict":  "FALSE_POSITIVE",
			"evidence": []any{"grep hit at line 12", "no sink reachable"},
		},
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsFPProperty()

	for key := range result.Properties {
		if key == "Evidence" || key == "FPEvidence" {
			t.Errorf("evidence must never be projected into a template-facing property, found %q", key)
		}
	}
}

func TestFPVerdict_UnrecognizedValue(t *testing.T) {
	id := "rule.test"
	result := resultFor(id)
	result.Properties = map[string]any{"fp": map[string]any{"verdict": "MAYBE"}}

	if _, ok := fpVerdict(result); ok {
		t.Error("fpVerdict() ok = true, want false for an unrecognized verdict string")
	}
}

func TestFPVerdict_NoBag(t *testing.T) {
	id := "rule.test"
	result := resultFor(id)
	result.Properties = map[string]any{}

	if _, ok := fpVerdict(result); ok {
		t.Error("fpVerdict() ok = true, want false with no fp bag")
	}
}
