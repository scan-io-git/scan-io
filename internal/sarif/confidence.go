package sarif

import (
	"fmt"
	"math"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

var precisionToConfidence = map[string]float64{
	"very-high": 0.95,
	"high":      0.85,
	"medium":    0.65,
	"low":       0.40,
}

// resolveConfidence returns the confidence value for a result using a three-tier chain.
// Returns (0, false) when no signal is present — caller should omit the property.
//
// Priority (first non-nil wins):
//  1. result.properties.confidence  — explicit override (float, json.Number, or string)
//  2. Semgrep confidence tag        — rule.properties.tags[] "HIGH/MEDIUM/LOW CONFIDENCE"
//  3. precision string              — rule.properties.precision via precisionToConfidence
func resolveConfidence(result *sarif.Result, rule *sarif.ReportingDescriptor) (float64, bool) {
	if result.Properties != nil {
		if raw, ok := result.Properties["confidence"]; ok {
			if f, ok := toFloat64(raw); ok {
				return clamp(f), true
			}
			// String like "high" / "medium" — try the precision table.
			if s, ok := raw.(string); ok {
				if f, ok := precisionToConfidence[strings.ToLower(strings.TrimSpace(s))]; ok {
					return f, true
				}
			}
		}
	}

	if rule != nil {
		tags, _ := rule.Properties["tags"].([]any)
		for _, t := range tags {
			tag, ok := t.(string)
			if !ok {
				continue
			}
			upper := strings.ToUpper(strings.TrimSpace(tag))
			switch upper {
			case "HIGH CONFIDENCE":
				return precisionToConfidence["high"], true
			case "MEDIUM CONFIDENCE":
				return precisionToConfidence["medium"], true
			case "LOW CONFIDENCE":
				return precisionToConfidence["low"], true
			}
		}

		if prec, ok := rule.Properties["precision"].(string); ok {
			if f, ok := precisionToConfidence[strings.ToLower(strings.TrimSpace(prec))]; ok {
				return f, true
			}
		}
	}

	return 0, false
}

// resolvePreFPConfidence returns the scanner's original confidence value that a
// false-positive review overwrote, read from properties.pre_fp_confidence — a
// sibling of properties.fp, not one of its keys. Returns (0, false) when
// absent, which is the common case for scanners with no native confidence
// (e.g. secrets scanners) and for any result that was never assessed.
func resolvePreFPConfidence(result *sarif.Result) (float64, bool) {
	if result.Properties == nil {
		return 0, false
	}
	raw, ok := result.Properties["pre_fp_confidence"]
	if !ok {
		return 0, false
	}
	if f, ok := toFloat64(raw); ok {
		return clamp(f), true
	}
	if s, ok := raw.(string); ok {
		if f, ok := precisionToConfidence[strings.ToLower(strings.TrimSpace(s))]; ok {
			return f, true
		}
	}
	return 0, false
}

// EnrichResultsPreFPConfidenceProperty writes Properties["PreFPConfidence"]
// (e.g. "High (85%)") for every result whose properties.pre_fp_confidence
// resolves to a value, so the meta list can render "High (85%) → Low (12%)"
// alongside Properties["Confidence"]. The key is omitted when no pre-FP value
// is present, and no uninformative prior is substituted in its place: a report
// reads as a record, so showing an invented "before" number would be worse
// than showing only the value that was actually measured.
func (r Report) EnrichResultsPreFPConfidenceProperty() {
	for _, result := range r.Runs[0].Results {
		conf, ok := resolvePreFPConfidence(result)
		if !ok {
			continue
		}
		if result.Properties == nil {
			result.Properties = make(map[string]any)
		}
		result.Properties["PreFPConfidence"] = formatConfidence(conf)
	}
}

// formatConfidence converts a float confidence value to a display string, e.g. "High (85%)".
func formatConfidence(c float64) string {
	pct := int(math.Round(c * 100))
	var label string
	switch {
	case c >= 0.90:
		label = "Very high"
	case c >= 0.75:
		label = "High"
	case c >= 0.50:
		label = "Medium"
	default:
		label = "Low"
	}
	return fmt.Sprintf("%s (%d%%)", label, pct)
}

func clamp(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}
