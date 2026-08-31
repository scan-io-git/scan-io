package sarif

import (
	"testing"

	gosarif "github.com/owenrumney/go-sarif/v2/sarif"
)

func TestEnrichPreFPConfidence_Present(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"Severity":          "high",
		"confidence":        0.12,
		"pre_fp_confidence": 0.85,
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsConfidenceProperty()
	report.EnrichResultsPreFPConfidenceProperty()

	wantPre := "High (85%)"
	if got, _ := result.Properties["PreFPConfidence"].(string); got != wantPre {
		t.Errorf("PreFPConfidence = %q, want %q", got, wantPre)
	}
	wantCur := "Low (12%)"
	if got, _ := result.Properties["Confidence"].(string); got != wantCur {
		t.Errorf("Confidence = %q, want %q", got, wantCur)
	}
}

func TestEnrichPreFPConfidence_Absent(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"Severity":   "high",
		"confidence": 0.9,
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsPreFPConfidenceProperty()

	if _, ok := result.Properties["PreFPConfidence"]; ok {
		t.Errorf("PreFPConfidence must be omitted when pre_fp_confidence is absent, got %v", result.Properties["PreFPConfidence"])
	}
}

func TestEnrichPreFPConfidence_EqualToCurrent(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"Severity":          "high",
		"confidence":        0.85,
		"pre_fp_confidence": 0.85,
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsConfidenceProperty()
	report.EnrichResultsPreFPConfidenceProperty()

	want := "High (85%)"
	if got, _ := result.Properties["PreFPConfidence"].(string); got != want {
		t.Errorf("PreFPConfidence = %q, want %q", got, want)
	}
	if got, _ := result.Properties["Confidence"].(string); got != want {
		t.Errorf("Confidence = %q, want %q", got, want)
	}
}

func TestResolvePreFPConfidence_StringPrecision(t *testing.T) {
	id := "rule.test"
	result := resultFor(id)
	result.Properties = map[string]any{"pre_fp_confidence": "high"}

	conf, ok := resolvePreFPConfidence(result)
	if !ok {
		t.Fatal("resolvePreFPConfidence() ok = false, want true")
	}
	if conf != precisionToConfidence["high"] {
		t.Errorf("resolvePreFPConfidence() = %v, want %v", conf, precisionToConfidence["high"])
	}
}

func TestResolvePreFPConfidence_NoProperties(t *testing.T) {
	result := &gosarif.Result{}
	if _, ok := resolvePreFPConfidence(result); ok {
		t.Error("resolvePreFPConfidence() ok = true, want false when Properties is nil")
	}
}
