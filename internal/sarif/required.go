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
}

// EnrichResultsRequiredProperty classifies every non-suppressed result as
// Required or Recommended per the policy, writing Properties["Required"]
// ("true"/"false") and Properties["RequiredReason"] (human-readable rationale).
// Suppressed results are left untouched. In-memory only; never written to disk.
//
// Gate order. Every severity is treated the same way: what you list in
// BlockerSeverities is what can block, and nothing is special-cased.
//  1. Severity not in BlockerSeverities → Recommended.
//  2. A recognized FP verdict takes precedence over Thresholds: FALSE_POSITIVE →
//     Recommended; TRUE_POSITIVE and NEEDS_VERIFICATION → Required.
//  3. No fp bag, or an unrecognized verdict: falls back to the
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
		case !policy.BlockerSeverities[sev]:
			reason = fmt.Sprintf("%s severity is not required", capSev)
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

func pct(f float64) int { return int(math.Round(f * 100)) }

// CollectRequiredInfo returns counts of required and recommended active findings.
// Suppressed results are excluded. Reads Properties["Required"] set by
// EnrichResultsRequiredProperty.
func (r Report) CollectRequiredInfo() map[string]int {
	counts := map[string]int{"required": 0, "recommended": 0}
	for _, run := range r.Runs {
		for _, result := range run.Results {
			if s, _ := result.Properties["Suppressed"].(string); s == "true" {
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

// SortResultsByRequiredThenSeverity orders results Required-first, then by the
// canonical severity order within each group. Use instead of SortResultsBySeverity
// when classification is active so the two content sections render contiguously.
func (r Report) SortResultsByRequiredThenSeverity() {
	severityOrder := map[string]int{
		"critical": 0, "high": 1, "medium": 2, "low": 3, "info": 4, "unknown": 5,
	}
	rank := func(res *sarif.Result) (int, int) {
		reqRank := 1
		if v, _ := res.Properties["Required"].(string); v == "true" {
			reqRank = 0
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
