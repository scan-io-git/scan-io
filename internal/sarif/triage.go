package sarif

import (
	"net/url"
	"strings"
	"time"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

// Triage decisions and statuses recognized in result.Properties["triage"]. A
// producer that tracks reviewer decisions on a pull request annotates the
// results those decisions concern; anything outside these values is treated as
// "no triage" so a newer producer cannot silently change classification.
const (
	TriageFalsePositive = "false_positive"
	TriageAcceptedRisk  = "accepted_risk"
	TriageTruePositive  = "true_positive"

	TriageApplied = "applied"
	TriagePending = "pending"
)

// Decision kinds exposed as Properties["DecisionKind"].
const (
	DecisionExcluded  = "excluded"
	DecisionConfirmed = "confirmed"
	DecisionPending   = "pending"
)

// Result classes, as returned by ResultClass.
const (
	ClassRequired    = "required"
	ClassRecommended = "recommended"
	ClassExcluded    = "excluded"
	ClassNone        = "none"
)

var triageDecisionLabels = map[string]string{
	TriageFalsePositive: "False positive",
	TriageAcceptedRisk:  "Accepted risk",
	TriageTruePositive:  "Real issue",
}

// triage is the parsed properties.triage bag.
type triage struct {
	Decision      string
	Status        string
	Author        string
	Reason        string
	Approver      string
	Ticket        string
	Source        string
	RequestSource string
	DecidedAt     string
}

func isExclusion(decision string) bool {
	return decision == TriageFalsePositive || decision == TriageAcceptedRisk
}

// triageOf returns the result's triage bag, or nil when it is absent or carries
// an unrecognized decision or status.
func triageOf(result *sarif.Result) *triage {
	if result.Properties == nil {
		return nil
	}
	bag, ok := result.Properties["triage"].(map[string]any)
	if !ok {
		return nil
	}
	str := func(key string) string {
		s, _ := bag[key].(string)
		return strings.TrimSpace(s)
	}
	t := &triage{
		Decision:      str("decision"),
		Status:        str("status"),
		Author:        str("author"),
		Reason:        str("reason"),
		Approver:      str("approver"),
		Ticket:        str("ticket"),
		Source:        webURL(str("source")),
		RequestSource: webURL(str("requestSource")),
		DecidedAt:     str("decidedAt"),
	}
	switch t.Decision {
	case TriageFalsePositive, TriageAcceptedRisk, TriageTruePositive:
	default:
		return nil
	}
	if t.Status != TriageApplied && t.Status != TriagePending {
		return nil
	}
	return t
}

// webURL keeps only absolute http(s) URLs; the values come from an external
// producer and end up in href attributes.
func webURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return raw
}

// kind maps a triage bag to its decision kind: an applied exclusion, an applied
// confirmation, or a pending exclusion request. A pending confirmation is not
// something a reviewer can request, so it carries no kind.
func (t *triage) kind() string {
	switch {
	case t == nil:
		return ""
	case t.Status == TriageApplied && isExclusion(t.Decision):
		return DecisionExcluded
	case t.Status == TriageApplied && t.Decision == TriageTruePositive:
		return DecisionConfirmed
	case t.Status == TriagePending && isExclusion(t.Decision):
		return DecisionPending
	}
	return ""
}

func isSuppressed(result *sarif.Result) bool {
	if len(result.Suppressions) > 0 {
		return true
	}
	s, _ := result.Properties["Suppressed"].(string)
	return s == "true"
}

// EnrichResultsTriageProperty projects properties.triage into template-facing
// properties on every non-suppressed result that carries a recognized triage
// bag: DecisionKind, Excluded ("true" for an applied exclusion) and PRDecision,
// a map holding the display values of the PR decision panel. It runs whether or
// not Required/Recommended classification is on, because an applied exclusion
// applies in both modes. Suppressed results are left untouched.
func (r Report) EnrichResultsTriageProperty() {
	for _, run := range r.Runs {
		for _, result := range run.Results {
			if result == nil || isSuppressed(result) {
				continue
			}
			t := triageOf(result)
			kind := t.kind()
			if kind == "" {
				continue
			}
			result.Properties["DecisionKind"] = kind
			if kind == DecisionExcluded {
				result.Properties["Excluded"] = "true"
			}
			result.Properties["PRDecision"] = map[string]any{
				"Kind":          kind,
				"Decision":      t.Decision,
				"DecisionLabel": triageDecisionLabels[t.Decision],
				"Author":        t.Author,
				"Approver":      t.Approver,
				"Ticket":        t.Ticket,
				"Reason":        t.Reason,
				"SourceURL":     t.Source,
				"RequestURL":    t.RequestSource,
				"Date":          formatDecisionDate(t.DecidedAt),
			}
		}
	}
}

func formatDecisionDate(raw string) string {
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return ""
	}
	return ts.UTC().Format("2 Jan 2006")
}

// ResultClass returns the class assigned by EnrichResultsTriageProperty and
// EnrichResultsRequiredProperty: excluded, required, recommended, or none when
// the report has no Required/Recommended split and the result is not excluded.
func ResultClass(result *sarif.Result) string {
	if v, _ := result.Properties["Excluded"].(string); v == "true" {
		return ClassExcluded
	}
	switch v, _ := result.Properties["Required"].(string); v {
	case "true":
		return ClassRequired
	case "false":
		return ClassRecommended
	}
	return ClassNone
}

// ResultDecisionKind returns "excluded", "confirmed", "pending" or "".
func ResultDecisionKind(result *sarif.Result) string {
	k, _ := result.Properties["DecisionKind"].(string)
	return k
}

// CollectDecisionInfo returns how many non-suppressed results carry each PR
// decision kind ("excluded", "confirmed", "pending"). Kinds nobody used are
// absent from the map. Reads Properties["DecisionKind"] set by
// EnrichResultsTriageProperty.
func (r Report) CollectDecisionInfo() map[string]int {
	counts := map[string]int{}
	for _, run := range r.Runs {
		for _, result := range run.Results {
			if k := ResultDecisionKind(result); k != "" {
				counts[k]++
			}
		}
	}
	return counts
}
