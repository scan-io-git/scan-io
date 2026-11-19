package sarif

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

// RequiredPolicy controls Required/Recommended classification.
// Both maps are keyed by lowercase severity bucket (critical/high/medium/low/info).
// An empty or nil Thresholds map disables confidence filtering: every finding whose
// severity is in BlockerSeverities is Required regardless of its confidence score.
// A threshold is only applied for severities that have an explicit entry in Thresholds.
type RequiredPolicy struct {
	BlockerSeverities map[string]bool
	Thresholds        map[string]float64
	// NeverDemoteSeverities pins a blocker severity to Required regardless of any
	// FP verdict or confidence threshold. Empty means nothing is pinned: every
	// severity is then subject to the same gates. It exists so a caller that
	// treats one severity as non-negotiable can say so explicitly, instead of the
	// rule being hardcoded here where a second consumer cannot see or share it.
	NeverDemoteSeverities map[string]bool
}

// EnrichResultsRequiredProperty classifies every non-suppressed result as
// Required or Recommended per the policy, writing Properties["Required"]
// ("true"/"false") and Properties["RequiredReason"] (human-readable rationale).
// Suppressed results are left untouched, and so are results excluded by an
// applied PR decision: EnrichResultsTriageProperty must run first so both are
// known. In-memory only; never written to disk.
//
// Gate order. Every severity is treated the same way: what you list in
// BlockerSeverities is what can block, and nothing is special-cased.
//  0. An applied reviewer confirmation (triage true_positive) → Required, whatever
//     the severity or verdict. An applied exclusion never reaches this function.
//  1. Severity not in BlockerSeverities → Recommended. NeverDemoteSeverities does
//     not promote: a severity nobody asked to block stays Recommended.
//  2. Severity in NeverDemoteSeverities → Required, skipping both the verdict and
//     the threshold.
//  3. A recognized FP verdict takes precedence over Thresholds: FALSE_POSITIVE →
//     Recommended; TRUE_POSITIVE and NEEDS_VERIFICATION → Required.
//  4. No fp bag, or an unrecognized verdict: falls back to the
//     confidence-threshold path, so callers that pass Thresholds see the same
//     behavior as before verdicts were read at all. With no threshold
//     configured for the severity, that path fails closed to Required.
func (r Report) EnrichResultsRequiredProperty(policy RequiredPolicy) {
	rulesMap := map[string]*sarif.ReportingDescriptor{}
	for _, rule := range r.Runs[0].Tool.Driver.Rules {
		rulesMap[rule.ID] = rule
	}

	for _, result := range r.Runs[0].Results {
		if result.Properties == nil {
			result.Properties = make(map[string]any)
		}
		if s, _ := result.Properties["Suppressed"].(string); s == "true" {
			continue
		}
		if ResultClass(result) == ClassExcluded {
			continue
		}

		sev, _ := result.Properties["Severity"].(string)
		sev = strings.ToLower(strings.TrimSpace(sev))

		var capSev string
		if sev != "" {
			capSev = strings.ToUpper(sev[:1]) + sev[1:]
		}

		required := false
		reason := ""

		verdict, hasVerdict := fpVerdict(result)

		switch {
		case ResultDecisionKind(result) == DecisionConfirmed:
			required = true
			reason = confirmedReason(verdict, hasVerdict)
		case !policy.BlockerSeverities[sev]:
			reason = fmt.Sprintf("%s severity is not required", capSev)
		case policy.NeverDemoteSeverities[sev]:
			required = true
			reason = neverDemoteReason(capSev, verdict, hasVerdict)
		case hasVerdict:
			required = verdict != FPVerdictFalsePositive
			reason = fpReasonPhrase(capSev, verdict, hasVerdict)
		default:
			var rule *sarif.ReportingDescriptor
			if result.RuleID != nil {
				rule = rulesMap[*result.RuleID]
			}
			thr, hasThr := policy.Thresholds[sev]
			conf, ok := resolveConfidence(result, rule)
			switch {
			case !hasThr:
				required = true
				reason = fpReasonPhrase(capSev, verdict, hasVerdict)
			case !ok:
				required = true
				reason = fmt.Sprintf("%s severity, no confidence score (treated as fully confident)", capSev)
			case conf >= thr:
				required = true
				reason = fmt.Sprintf("%s severity, confidence %d%% ≥ %d%% threshold", capSev, pct(conf), pct(thr))
			default:
				required = false
				reason = fmt.Sprintf("%s severity, confidence %d%% < %d%% threshold", capSev, pct(conf), pct(thr))
			}
		}

		result.Properties["Required"] = strconv.FormatBool(required)
		result.Properties["RequiredReason"] = reason
	}
}

// fpReasonPhrase renders the FP-review clause of RequiredReason for a blocker
// severity: the verdict when recognized, "not FP-assessed" otherwise (no fp
// bag, or an unrecognized/legacy verdict).
func fpReasonPhrase(capSev, verdict string, hasVerdict bool) string {
	if hasVerdict {
		switch verdict {
		case FPVerdictFalsePositive:
			return fmt.Sprintf("%s severity, false positive per FP review", capSev)
		case FPVerdictTruePositive:
			return fmt.Sprintf("%s severity, confirmed by FP review", capSev)
		case FPVerdictNeedsVerification:
			return fmt.Sprintf("%s severity, needs verification", capSev)
		}
	}
	return fmt.Sprintf("%s severity, not FP-assessed", capSev)
}

// confirmedReason explains a Required class that comes from a reviewer's
// decision. The extra clause is for the verdict that the decision overrides, so
// the banner does not read as self-contradictory above a false-positive panel.
func confirmedReason(verdict string, hasVerdict bool) string {
	if hasVerdict && verdict == FPVerdictFalsePositive {
		return "confirmed real by a reviewer, overrides the false-positive review"
	}
	return "confirmed real by a reviewer"
}

// neverDemoteReason states policy as the reason, because policy is what decided
// it: a pinned severity is Required whether or not a verdict exists and whatever
// the verdict says. The extra clause is for the one case where the verdict
// disagrees, so the banner does not read as though no review happened while the
// FP panel below it shows a false-positive verdict.
func neverDemoteReason(capSev, verdict string, hasVerdict bool) string {
	if hasVerdict && verdict == FPVerdictFalsePositive {
		return fmt.Sprintf("%s severity, always required despite false-positive review", capSev)
	}
	return fmt.Sprintf("%s severity, always required", capSev)
}

func pct(f float64) int { return int(math.Round(f * 100)) }

// CollectRequiredInfo returns counts of required and recommended active findings.
// Suppressed and PR-excluded results are not counted. Reads Properties["Required"] set by
// EnrichResultsRequiredProperty.
func (r Report) CollectRequiredInfo() map[string]int {
	counts := map[string]int{"required": 0, "recommended": 0}
	for _, run := range r.Runs {
		for _, result := range run.Results {
			if s, _ := result.Properties["Suppressed"].(string); s == "true" {
				continue
			}
			if ResultClass(result) == ClassExcluded {
				continue
			}
			if v, _ := result.Properties["Required"].(string); v == "true" {
				counts["required"]++
			} else {
				counts["recommended"]++
			}
		}
	}
	return counts
}

// SortResultsByRequiredThenSeverity orders results Required-first, then
// Recommended, then PR-excluded, with the canonical severity order within each
// group. Use instead of SortResultsBySeverity when classification is active so
// the content sections render contiguously.
func (r Report) SortResultsByRequiredThenSeverity() {
	severityOrder := map[string]int{
		"critical": 0, "high": 1, "medium": 2, "low": 3, "info": 4, "unknown": 5,
	}
	rank := func(res *sarif.Result) (int, int) {
		reqRank := 1
		switch ResultClass(res) {
		case ClassRequired:
			reqRank = 0
		case ClassExcluded:
			reqRank = 2
		}
		sev, _ := res.Properties["Severity"].(string)
		sevRank, ok := severityOrder[sev]
		if !ok {
			sevRank = 5
		}
		return reqRank, sevRank
	}
	for _, run := range r.Runs {
		sort.SliceStable(run.Results, func(i, j int) bool {
			ri, si := rank(run.Results[i])
			rj, sj := rank(run.Results[j])
			if ri != rj {
				return ri < rj
			}
			return si < sj
		})
	}
}
