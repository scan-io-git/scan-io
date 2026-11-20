package tohtml

import (
	"regexp"
	"strings"
	"testing"
)

const decisionPR = "https://github.com/scan-io-git/scanio-demo/pull/7"

func decisionHTML(t *testing.T, required string, results ...testResult) string {
	t.Helper()
	dir := t.TempDir()
	in := writeTestSARIF(t, dir, "in.sarif", nil, results...)
	return renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir, Required: required})
}

func triageBag(decision, status string, extra map[string]any) map[string]any {
	bag := map[string]any{"decision": decision, "status": status, "author": "alice", "reason": "synthetic reason", "decidedAt": "2026-11-17T04:10:16Z"}
	for k, v := range extra {
		bag[k] = v
	}
	return bag
}

// cardFor returns the markup from the card whose rule is ruleID up to the next
// card (or the end of the page for the last one).
func cardFor(t *testing.T, html, ruleID string) string {
	t.Helper()
	re := regexp.MustCompile(`<details id="issue-\d+"[^>]*data-rule-id="` + regexp.QuoteMeta(ruleID) + `"`)
	loc := re.FindStringIndex(html)
	if loc == nil {
		t.Fatalf("no card for rule %s", ruleID)
	}
	rest := html[loc[0]+1:]
	if next := strings.Index(rest, `<details id="issue-`); next >= 0 {
		rest = rest[:next]
	}
	return rest
}

func TestDecision_ExcludedFalsePositive(t *testing.T) {
	html := decisionHTML(t, "high", testResult{Rule: "r.fp", Level: "high", Props: map[string]any{
		"findingId": "F-0000000a",
		"triage":    triageBag("false_positive", "applied", map[string]any{"source": decisionPR + "#issuecomment-1"}),
	}})
	card := cardFor(t, html, "r.fp")

	for _, want := range []string{
		`<strong>Excluded in this PR</strong> &mdash; false positive, does not block the merge`,
		`finding__decision finding__decision--excluded`,
		`<dt class="finding__decision-key">Excluded as:</dt><dd>False positive</dd>`,
		`<dt class="finding__decision-key">Decided by:</dt>`,
		`<dt class="finding__decision-key">Decided by:</dt><dd>alice<a class="finding__decision-link"`,
		`href="` + decisionPR + `#issuecomment-1"`,
		`>comment</a>`,
		`"synthetic reason"`,
		`<dt class="finding__decision-key">Date:</dt><dd>17 Nov 2026</dd>`,
	} {
		if !strings.Contains(card, want) {
			t.Errorf("card is missing %q", want)
		}
	}
	for _, unwanted := range []string{`Ticket:`, `Approved by:`, `Requested by:`, `req-notice--recommended`, `req-notice--required`} {
		if strings.Contains(card, unwanted) {
			t.Errorf("card unexpectedly contains %q", unwanted)
		}
	}
	if strings.Contains(card, `<details class="finding__decision finding__decision--excluded" open`) {
		t.Error("PR decision panel must be collapsed by default")
	}
}

func TestDecision_ExcludedAcceptedRiskWithTicket(t *testing.T) {
	html := decisionHTML(t, "", testResult{Rule: "r.ar", Props: map[string]any{
		"triage": triageBag("accepted_risk", "applied", map[string]any{"ticket": "TKT-1"}),
	}})
	card := cardFor(t, html, "r.ar")

	mustContain(t, card, `&mdash; accepted risk, does not block the merge`)
	mustContain(t, card, `<dd>Accepted risk</dd>`)
	mustContain(t, card, `<dt class="finding__decision-key">Ticket:</dt><dd>TKT-1</dd>`)
	mustNotContain(t, card, `comment</a>`)
}

func TestDecision_ApprovedRequest(t *testing.T) {
	html := decisionHTML(t, "high", testResult{Rule: "r.apr", Props: map[string]any{
		"triage": triageBag("false_positive", "applied", map[string]any{
			"approver": "bob", "requestSource": decisionPR + "#issuecomment-1", "source": decisionPR + "#issuecomment-2",
		}),
	}})
	card := cardFor(t, html, "r.apr")

	mustContain(t, card, `<dt class="finding__decision-key">Requested by:</dt><dd>alice<a class="finding__decision-link" href="`+decisionPR+`#issuecomment-1" target="_blank" rel="noopener noreferrer">request</a>`)
	mustContain(t, card, `<dt class="finding__decision-key">Approved by:</dt><dd>bob<a class="finding__decision-link" href="`+decisionPR+`#issuecomment-2" target="_blank" rel="noopener noreferrer">approval</a>`)
	mustNotContain(t, card, `Decided by:`)
}

func TestDecision_LinksOmittedWhenURLMissingOrUnsafe(t *testing.T) {
	html := decisionHTML(t, "", testResult{Rule: "r.nolink", Props: map[string]any{
		"triage": triageBag("false_positive", "applied", map[string]any{"source": "javascript:alert(1)", "decidedAt": nil}),
	}})
	card := cardFor(t, html, "r.nolink")

	mustContain(t, card, `alice</dd>`)
	mustNotContain(t, card, `finding__decision-link`)
	mustNotContain(t, card, `javascript:`)
	mustNotContain(t, card, `Date:`)
}

func TestDecision_ConfirmedOverFalsePositiveVerdict(t *testing.T) {
	html := decisionHTML(t, "critical", testResult{Rule: "r.tp", Level: "low", Props: map[string]any{
		"fp":     map[string]any{"verdict": "FALSE_POSITIVE", "reasoning": "looked benign"},
		"triage": triageBag("true_positive", "applied", map[string]any{"source": decisionPR + "#issuecomment-3"}),
	}})
	card := cardFor(t, html, "r.tp")

	mustContain(t, card, `<strong>Required</strong> &mdash; confirmed real by a reviewer, overrides the false-positive review</span>`)
	mustContain(t, card, `finding__decision--confirmed`)
	mustContain(t, card, `<dt class="finding__decision-key">Marked as:</dt><dd>Real issue</dd>`)
	mustContain(t, card, `>comment</a>`)
	if strings.Index(card, "finding__decision--confirmed") > strings.Index(card, `class="finding__fp"`) {
		t.Error("the PR decision panel must come before the false-positive review panel")
	}
}

func TestDecision_ConfirmedWithoutVerdictHasNoOverrideClause(t *testing.T) {
	html := decisionHTML(t, "critical", testResult{Rule: "r.tp2", Level: "low", Props: map[string]any{
		"triage": triageBag("true_positive", "applied", nil),
	}})
	card := cardFor(t, html, "r.tp2")

	mustContain(t, card, `confirmed real by a reviewer</span>`)
	mustNotContain(t, card, `overrides the false-positive review`)
}

func TestDecision_PendingOnRequiredAndRecommended(t *testing.T) {
	html := decisionHTML(t, "high",
		testResult{Rule: "r.preq", Level: "high", Props: map[string]any{
			"triage": triageBag("false_positive", "pending", map[string]any{"requestSource": decisionPR + "#issuecomment-4"}),
		}},
		testResult{Rule: "r.prec", Level: "low", Props: map[string]any{
			"triage": triageBag("accepted_risk", "pending", map[string]any{"source": decisionPR + "#issuecomment-5"}),
		}},
	)
	req := cardFor(t, html, "r.preq")
	rec := cardFor(t, html, "r.prec")

	mustContain(t, req, `. An exclusion is waiting for approval</span>`)
	mustContain(t, req, `PR decision: exclusion requested`)
	mustContain(t, req, `finding__decision--pending`)
	mustContain(t, req, `<dt class="finding__decision-key">Status:</dt><dd>Waiting for approval; still blocks the merge</dd>`)
	mustContain(t, req, `#issuecomment-4"`)

	mustNotContain(t, rec, `An exclusion is waiting for approval`)
	mustContain(t, rec, `req-notice--recommended`)
	mustContain(t, rec, `<dt class="finding__decision-key">Status:</dt><dd>Waiting for approval</dd>`)
	mustContain(t, rec, `<dt class="finding__decision-key">Requested as:</dt><dd>Accepted risk</dd>`)
	mustContain(t, rec, `#issuecomment-5"`)
}

func TestDecision_WithoutRequiredShowsPanelsWithoutRequiredNotice(t *testing.T) {
	html := decisionHTML(t, "",
		testResult{Rule: "r.c", Props: map[string]any{"triage": triageBag("true_positive", "applied", nil)}},
		testResult{Rule: "r.p", Props: map[string]any{"triage": triageBag("false_positive", "pending", nil)}},
	)
	for _, rule := range []string{"r.c", "r.p"} {
		card := cardFor(t, html, rule)
		mustContain(t, card, `finding__decision finding__decision--`)
		mustNotContain(t, card, `class="req-notice`)
	}
	mustContain(t, cardFor(t, html, "r.p"), `<dd>Waiting for approval</dd>`)
}

func TestDecision_NoTriageNoPanel(t *testing.T) {
	html := decisionHTML(t, "high", testResult{Rule: "r.plain", Level: "high"})
	mustNotContain(t, cardFor(t, html, "r.plain"), `finding__decision`)
	mustNotContain(t, html, `<details class="finding__decision`)
	mustNotContain(t, html, `id="tb-clock"`)
}
