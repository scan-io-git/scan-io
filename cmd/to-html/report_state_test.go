package tohtml

import "testing"

func TestReportStateID(t *testing.T) {
	const (
		repo   = "github.com/acme/widgets"
		pr     = "42"
		commit = "daaee1979c3ee11cb403567a256ad250d7a34986"
	)

	scanners := []string{"Semgrep OSS", "AI Security Scanner"}

	t.Run("stable across renders of the same scan", func(t *testing.T) {
		// Re-rendering the same report must return the reader's filters, so the id
		// cannot depend on anything that changes per render (time, nonce).
		if a, b := reportStateID(repo, pr, commit, scanners, false), reportStateID(repo, pr, commit, scanners, false); a != b {
			t.Errorf("id not stable: %q vs %q", a, b)
		}
	})

	t.Run("empty without a commit", func(t *testing.T) {
		// No commit means nothing pins the report to a code state. Empty means
		// "do not persist" -- a shared fallback id would leak one report's
		// filters onto another.
		for _, commit := range []string{"", "   "} {
			if got := reportStateID(repo, pr, commit, scanners, false); got != "" {
				t.Errorf("reportStateID(commit=%q) = %q, want \"\"", commit, got)
			}
		}
	})

	t.Run("commit alone is enough", func(t *testing.T) {
		if got := reportStateID("", "", commit, scanners, false); got == "" {
			t.Error("a commit hash alone should identify the report")
		}
	})

	t.Run("differs when any input differs", func(t *testing.T) {
		base := reportStateID(repo, pr, commit, scanners, false)
		cases := map[string]string{
			"other commit": reportStateID(repo, pr, "0000000000000000000000000000000000000000", scanners, false),
			"other pr":     reportStateID(repo, "43", commit, scanners, false),
			"other repo":   reportStateID("github.com/acme/other", pr, commit, scanners, false),
		}
		for name, got := range cases {
			if got == base {
				t.Errorf("%s: id matched the base id %q; distinct reports must not share filter state", name, base)
			}
		}
	})

	t.Run("field boundaries are unambiguous", func(t *testing.T) {
		// Naive concatenation would make ("ab","c") and ("a","bc") collide.
		if reportStateID("ab", "c", commit, scanners, false) == reportStateID("a", "bc", commit, scanners, false) {
			t.Error("field values run together; separate them when hashing")
		}
	})

	t.Run("safe as an HTML attribute and a storage key", func(t *testing.T) {
		id := reportStateID(repo, pr, commit, scanners, false)
		for _, r := range id {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				t.Fatalf("id %q contains %q; keep it hex so it needs no escaping", id, r)
			}
		}
		if len(id) != 16 {
			t.Errorf("len(id) = %d, want 16", len(id))
		}
	})

	t.Run("differs when the scanner set differs", func(t *testing.T) {
		// Same commit, different inputs, is a different report: a merged
		// two-scanner report and a single-scanner one must not share filter state,
		// or a severity chosen against one finding set is reapplied to another.
		merged := reportStateID(repo, pr, commit, scanners, false)
		single := reportStateID(repo, pr, commit, []string{"Semgrep OSS"}, false)
		if merged == single {
			t.Error("reports with different scanner sets shared an id")
		}
	})

	t.Run("scanner order does not matter", func(t *testing.T) {
		// Inputs arrive in whatever order the caller passed them; the report is the
		// same report either way.
		a := reportStateID(repo, pr, commit, []string{"Semgrep OSS", "AI Security Scanner"}, false)
		b := reportStateID(repo, pr, commit, []string{"AI Security Scanner", "Semgrep OSS"}, false)
		if a != b {
			t.Errorf("id depends on scanner order: %q vs %q", a, b)
		}
	})

	t.Run("differs when suppressed findings were removed", func(t *testing.T) {
		// --no-supressions strips results before the report is rendered, so the two
		// variants hold different finding sets from the same commit. Sharing an id
		// would reapply a filter chosen against one to the other -- the leak this
		// identifier exists to prevent, reached via a flag instead of a new commit.
		kept := reportStateID(repo, pr, commit, scanners, false)
		removed := reportStateID(repo, pr, commit, scanners, true)
		if kept == removed {
			t.Error("reports with and without suppressed findings shared an id")
		}
	})
}
