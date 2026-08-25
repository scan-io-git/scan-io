package sarif

import (
	"testing"

	gosarif "github.com/owenrumney/go-sarif/v2/sarif"
)

// thresholdFixture is a per-severity confidence threshold set used only by these
// tests to exercise the threshold path. These are not recommended defaults: the
// production path injects no thresholds unless a caller asks for them.
func thresholdFixture() map[string]float64 {
	return map[string]float64{
		"critical": 0.5,
		"high":     0.6,
		"medium":   0.7,
		"low":      0.8,
		"info":     1.1,
	}
}

func TestEnrichRequired_BlockerHighConfidence(t *testing.T) {
	id := "rule.test"
	rule := ruleWithTags(id, "HIGH CONFIDENCE") // resolves to 0.85
	result := resultFor(id)
	result.Properties = map[string]any{"Severity": "high"}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"high": true},
		Thresholds:        thresholdFixture(),
	})

	if got, _ := result.Properties["Required"].(string); got != "true" {
		t.Errorf("Required = %q, want \"true\"", got)
	}
}

func TestEnrichRequired_DemotedBelowThreshold(t *testing.T) {
	id := "rule.test"
	rule := ruleWithTags(id, "LOW CONFIDENCE") // resolves to 0.40
	result := resultFor(id)
	result.Properties = map[string]any{"Severity": "high"}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"high": true},
		Thresholds:        thresholdFixture(),
	})

	if got, _ := result.Properties["Required"].(string); got != "false" {
		t.Errorf("Required = %q, want \"false\" (0.40 < 0.60)", got)
	}
}

func TestEnrichRequired_NoConfidenceTreatedAsConfident(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id} // no confidence signal
	result := resultFor(id)
	result.Properties = map[string]any{"Severity": "critical"}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"critical": true},
		Thresholds:        thresholdFixture(),
	})

	if got, _ := result.Properties["Required"].(string); got != "true" {
		t.Errorf("Required = %q, want \"true\" (no confidence => confident)", got)
	}
}

func TestEnrichRequired_NoThresholdAlwaysRequired(t *testing.T) {
	id := "rule.test"
	rule := ruleWithTags(id, "LOW CONFIDENCE") // resolves to 0.40 — would fail any default threshold
	result := resultFor(id)
	result.Properties = map[string]any{"Severity": "high"}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"high": true},
		Thresholds:        map[string]float64{}, // empty → no confidence filtering
	})

	if got, _ := result.Properties["Required"].(string); got != "true" {
		t.Errorf("Required = %q, want \"true\" (no threshold → skip confidence check)", got)
	}
}

func TestEnrichRequired_SeverityNotBlocker(t *testing.T) {
	id := "rule.test"
	rule := ruleWithTags(id, "HIGH CONFIDENCE")
	result := resultFor(id)
	result.Properties = map[string]any{"Severity": "medium"}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"critical": true, "high": true},
		Thresholds:        thresholdFixture(),
	})

	if got, _ := result.Properties["Required"].(string); got != "false" {
		t.Errorf("Required = %q, want \"false\" (medium not a blocker)", got)
	}
}

func TestEnrichRequired_SuppressedSkipped(t *testing.T) {
	id := "rule.test"
	rule := ruleWithTags(id, "HIGH CONFIDENCE")
	result := resultFor(id)
	result.Properties = map[string]any{"Severity": "high", "Suppressed": "true"}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"high": true},
		Thresholds:        thresholdFixture(),
	})

	if _, ok := result.Properties["Required"]; ok {
		t.Errorf("suppressed result must not be classified")
	}
}

func TestCollectRequiredInfo(t *testing.T) {
	id := "rule.test"
	rule := ruleWithTags(id, "HIGH CONFIDENCE")
	r1 := resultFor(id)
	r1.Properties = map[string]any{"Severity": "high", "Required": "true"}
	r2 := resultFor(id)
	r2.Properties = map[string]any{"Severity": "low", "Required": "false"}
	r3 := resultFor(id)
	r3.Properties = map[string]any{"Severity": "high", "Suppressed": "true"}
	report := makeSimpleReport(id, rule, r1)
	report.Runs[0].Results = append(report.Runs[0].Results, r2, r3)

	info := report.CollectRequiredInfo()
	if info["required"] != 1 || info["recommended"] != 1 {
		t.Errorf("info = %v, want required:1 recommended:1 (suppressed excluded)", info)
	}
}

// ── FP verdict gate ────────────────────────────────────────────────────────

func TestEnrichRequired_FPVerdict_FalsePositiveDemotes(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"Severity": "high",
		"fp":       map[string]any{"verdict": "FALSE_POSITIVE", "p_real": 0.1},
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"high": true},
	})

	if got, _ := result.Properties["Required"].(string); got != "false" {
		t.Errorf("Required = %q, want \"false\" (FALSE_POSITIVE demotes)", got)
	}
	want := "High severity, false positive per FP review"
	if got, _ := result.Properties["RequiredReason"].(string); got != want {
		t.Errorf("RequiredReason = %q, want %q", got, want)
	}
}

func TestEnrichRequired_FPVerdict_TruePositiveRequired(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"Severity": "high",
		"fp":       map[string]any{"verdict": "TRUE_POSITIVE", "p_real": 0.95},
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"high": true},
	})

	if got, _ := result.Properties["Required"].(string); got != "true" {
		t.Errorf("Required = %q, want \"true\" (TRUE_POSITIVE)", got)
	}
	want := "High severity, confirmed by FP review"
	if got, _ := result.Properties["RequiredReason"].(string); got != want {
		t.Errorf("RequiredReason = %q, want %q", got, want)
	}
}

func TestEnrichRequired_FPVerdict_NeedsVerificationRequired(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"Severity": "high",
		"fp":       map[string]any{"verdict": "NEEDS_VERIFICATION", "p_real": 0.5},
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"high": true},
	})

	if got, _ := result.Properties["Required"].(string); got != "true" {
		t.Errorf("Required = %q, want \"true\" (NEEDS_VERIFICATION fails closed)", got)
	}
	want := "High severity, needs verification"
	if got, _ := result.Properties["RequiredReason"].(string); got != want {
		t.Errorf("RequiredReason = %q, want %q", got, want)
	}
}

// Critical is not special-cased: a FALSE_POSITIVE verdict demotes it exactly as
// it demotes any other blocker severity.
func TestEnrichRequired_FPVerdict_CriticalHonoursVerdict(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"Severity": "critical",
		"fp":       map[string]any{"verdict": "FALSE_POSITIVE", "p_real": 0.02},
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"critical": true},
	})

	if got, _ := result.Properties["Required"].(string); got != "false" {
		t.Errorf("Required = %q, want \"false\" (critical honours FALSE_POSITIVE like any other severity)", got)
	}
	want := "Critical severity, false positive per FP review"
	if got, _ := result.Properties["RequiredReason"].(string); got != want {
		t.Errorf("RequiredReason = %q, want %q", got, want)
	}
}

func TestEnrichRequired_NoFPBag_FailsClosed(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{"Severity": "high"}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"high": true},
	})

	if got, _ := result.Properties["Required"].(string); got != "true" {
		t.Errorf("Required = %q, want \"true\" (no fp bag fails closed)", got)
	}
	want := "High severity, not FP-assessed"
	if got, _ := result.Properties["RequiredReason"].(string); got != want {
		t.Errorf("RequiredReason = %q, want %q", got, want)
	}
}

func TestEnrichRequired_LegacyPRealOnly_NoVerdict_FailsClosed(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"Severity": "high",
		// Legacy bag with no "verdict" key: confidence was overwritten with the
		// probability score, but nothing says what the review concluded.
		"fp": map[string]any{"p_real": 0.5},
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"high": true},
	})

	if got, _ := result.Properties["Required"].(string); got != "true" {
		t.Errorf("Required = %q, want \"true\" (legacy p_real-only bag fails closed)", got)
	}
	want := "High severity, not FP-assessed"
	if got, _ := result.Properties["RequiredReason"].(string); got != want {
		t.Errorf("RequiredReason = %q, want %q", got, want)
	}
}

func TestEnrichRequired_UnrecognizedVerdict_FailsClosed(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"Severity": "high",
		"fp":       map[string]any{"verdict": "SOMETHING_ELSE"},
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"high": true},
	})

	if got, _ := result.Properties["Required"].(string); got != "true" {
		t.Errorf("Required = %q, want \"true\" (unrecognized verdict fails closed)", got)
	}
}

func TestEnrichRequired_VerdictOverridesThreshold(t *testing.T) {
	id := "rule.test"
	// Confidence tag alone would demote this below the 0.6 threshold, but a
	// recognized TRUE_POSITIVE verdict must win regardless.
	rule := ruleWithTags(id, "LOW CONFIDENCE")
	result := resultFor(id)
	result.Properties = map[string]any{
		"Severity": "high",
		"fp":       map[string]any{"verdict": "TRUE_POSITIVE", "p_real": 0.95},
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"high": true},
		Thresholds:        thresholdFixture(),
	})

	if got, _ := result.Properties["Required"].(string); got != "true" {
		t.Errorf("Required = %q, want \"true\" (verdict overrides threshold)", got)
	}
}

func TestEnrichRequired_NoVerdict_ThresholdPathReasonUnchanged(t *testing.T) {
	// Regression guard: when no fp bag is present and a threshold IS configured,
	// the pre-ticket-09 confidence-based reason text must not change.
	id := "rule.test"
	rule := ruleWithTags(id, "LOW CONFIDENCE") // resolves to 0.40
	result := resultFor(id)
	result.Properties = map[string]any{"Severity": "high"}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities: map[string]bool{"high": true},
		Thresholds:        thresholdFixture(),
	})

	want := "High severity, confidence 40% < 60% threshold"
	if got, _ := result.Properties["RequiredReason"].(string); got != want {
		t.Errorf("RequiredReason = %q, want %q", got, want)
	}
}

func TestSortByRequiredThenSeverity(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	lowReq := resultFor(id)
	lowReq.Properties = map[string]any{"Severity": "low", "Required": "true"}
	critRec := resultFor(id)
	critRec.Properties = map[string]any{"Severity": "critical", "Required": "false"}
	report := makeSimpleReport(id, rule, critRec)
	report.Runs[0].Results = append(report.Runs[0].Results, lowReq)

	report.SortResultsByRequiredThenSeverity()

	first, _ := report.Runs[0].Results[0].Properties["Required"].(string)
	if first != "true" {
		t.Errorf("required-first sort failed: first Required = %q, want \"true\"", first)
	}
}

// NeverDemoteSeverities pins a severity to Required. The verdict is still read and
// still rendered by the report; it just no longer decides the classification.
func TestEnrichRequired_NeverDemote_PinsDespiteFalsePositive(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"Severity": "critical",
		"fp":       map[string]any{"verdict": "FALSE_POSITIVE", "p_real": 0.02},
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities:     map[string]bool{"critical": true},
		NeverDemoteSeverities: map[string]bool{"critical": true},
	})

	if got, _ := result.Properties["Required"].(string); got != "true" {
		t.Errorf("Required = %q, want \"true\" (pinned severity ignores FALSE_POSITIVE)", got)
	}
	want := "Critical severity, always required despite false-positive review"
	if got, _ := result.Properties["RequiredReason"].(string); got != want {
		t.Errorf("RequiredReason = %q, want %q", got, want)
	}
	// The verdict must survive on the result: the FP panel still renders it.
	if _, ok := result.Properties["fp"]; !ok {
		t.Error("fp bag was removed; the report needs it to render the review panel")
	}
}

// A pinned severity states policy as its reason even when no review ran, because
// policy is what decided it -- the outcome would be the same either way.
func TestEnrichRequired_NeverDemote_ReasonWithoutVerdict(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{"Severity": "critical"}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities:     map[string]bool{"critical": true},
		NeverDemoteSeverities: map[string]bool{"critical": true},
	})

	if got, _ := result.Properties["Required"].(string); got != "true" {
		t.Errorf("Required = %q, want \"true\"", got)
	}
	want := "Critical severity, always required"
	if got, _ := result.Properties["RequiredReason"].(string); got != want {
		t.Errorf("RequiredReason = %q, want %q", got, want)
	}
}

// Pinning outranks a confidence threshold too, not just the verdict.
func TestEnrichRequired_NeverDemote_OutranksThreshold(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{"Severity": "high", "confidence": 0.10}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities:     map[string]bool{"high": true},
		Thresholds:            map[string]float64{"high": 0.90},
		NeverDemoteSeverities: map[string]bool{"high": true},
	})

	if got, _ := result.Properties["Required"].(string); got != "true" {
		t.Errorf("Required = %q, want \"true\" (pin beats the threshold)", got)
	}
	if got, _ := result.Properties["RequiredReason"].(string); got != "High severity, always required" {
		t.Errorf("RequiredReason = %q, want the policy reason, not a confidence comparison", got)
	}
}

// Pinning must not promote: a severity absent from BlockerSeverities stays
// Recommended even when it is listed as never-demote.
func TestEnrichRequired_NeverDemote_DoesNotPromoteNonBlocker(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{"Severity": "low"}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities:     map[string]bool{"critical": true},
		NeverDemoteSeverities: map[string]bool{"low": true},
	})

	if got, _ := result.Properties["Required"].(string); got != "false" {
		t.Errorf("Required = %q, want \"false\" (never-demote must not promote)", got)
	}
	want := "Low severity is not required"
	if got, _ := result.Properties["RequiredReason"].(string); got != want {
		t.Errorf("RequiredReason = %q, want %q", got, want)
	}
}

// An empty pin set leaves every existing gate untouched.
func TestEnrichRequired_NeverDemote_EmptyChangesNothing(t *testing.T) {
	id := "rule.test"
	rule := &gosarif.ReportingDescriptor{ID: id}
	result := resultFor(id)
	result.Properties = map[string]any{
		"Severity": "critical",
		"fp":       map[string]any{"verdict": "FALSE_POSITIVE", "p_real": 0.02},
	}
	report := makeSimpleReport(id, rule, result)

	report.EnrichResultsRequiredProperty(RequiredPolicy{
		BlockerSeverities:     map[string]bool{"critical": true},
		NeverDemoteSeverities: map[string]bool{},
	})

	if got, _ := result.Properties["Required"].(string); got != "false" {
		t.Errorf("Required = %q, want \"false\" (empty pin set is a no-op)", got)
	}
}
