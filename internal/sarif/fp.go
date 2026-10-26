package sarif

import (
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

// FP verdicts recognized in result.Properties["fp"]["verdict"]. A producer that
// runs a false-positive review over a SARIF annotates each assessed result with
// one of these values; anything else is treated as unassessed.
const (
	FPVerdictTruePositive      = "TRUE_POSITIVE"
	FPVerdictFalsePositive     = "FALSE_POSITIVE"
	FPVerdictNeedsVerification = "NEEDS_VERIFICATION"
)

// fpVerdictLabels are the display labels for the FP review panel. The wording
// for TRUE_POSITIVE is deliberate: phrasings built on "passed" read as a
// boolean about whether the review ran, and carry the wrong valence for a
// finding that is real and still blocks a merge.
var fpVerdictLabels = map[string]string{
	FPVerdictTruePositive:      "Confirmed as a real issue",
	FPVerdictFalsePositive:     "Likely false positive",
	FPVerdictNeedsVerification: "Needs verification",
}

// fpVerdictHover carries the hover/tooltip text shown only for
// NEEDS_VERIFICATION, whose meaning isn't obvious from the label alone.
var fpVerdictHover = map[string]string{
	FPVerdictNeedsVerification: "Automated false-positive review couldn't confirm whether this is real. Verify it by hand before merging.",
}

// fpBag returns result.Properties["fp"] as a map, and whether it is present.
func fpBag(result *sarif.Result) (map[string]any, bool) {
	if result.Properties == nil {
		return nil, false
	}
	bag, ok := result.Properties["fp"].(map[string]any)
	return bag, ok
}

// fpVerdict returns the result's FP verdict and true, only when the fp bag is
// present and its verdict is one of the three recognized values. A missing fp
// bag, a missing verdict key, or an unrecognized value (e.g. a bag carrying
// only a probability score and no verdict) all return ("", false)
// — callers must treat that as "not FP-assessed", never guess a verdict.
func fpVerdict(result *sarif.Result) (string, bool) {
	bag, ok := fpBag(result)
	if !ok {
		return "", false
	}
	v, _ := bag["verdict"].(string)
	v = strings.TrimSpace(v)
	switch v {
	case FPVerdictTruePositive, FPVerdictFalsePositive, FPVerdictNeedsVerification:
		return v, true
	default:
		return "", false
	}
}

// EnrichResultsFPProperty projects properties.fp into template-facing
// properties: FPVerdict (raw), FPLabel (display text), FPHover
// (NEEDS_VERIFICATION only), and FPReasoning. Properties are set only when the
// result carries a recognized verdict, so the template's FP review panel
// renders for exactly the results EnrichResultsRequiredProperty's verdict gate
// also recognizes — a result with no fp bag, or an unrecognized verdict, gets
// no panel at all. properties.fp.evidence is never projected; it stays hidden
// by design.
func (r Report) EnrichResultsFPProperty() {
	for _, result := range r.Runs[0].Results {
		verdict, ok := fpVerdict(result)
		if !ok {
			continue
		}
		if result.Properties == nil {
			result.Properties = make(map[string]any)
		}
		result.Properties["FPVerdict"] = verdict
		result.Properties["FPLabel"] = fpVerdictLabels[verdict]
		if hover, ok := fpVerdictHover[verdict]; ok {
			result.Properties["FPHover"] = hover
		}
		if bag, ok := fpBag(result); ok {
			if reasoning, ok := bag["reasoning"].(string); ok && strings.TrimSpace(reasoning) != "" {
				result.Properties["FPReasoning"] = reasoning
			}
		}
	}
}
