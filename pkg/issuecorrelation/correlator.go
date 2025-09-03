package issuecorrelation

// IssueMetadata describes the minimal metadata required to correlate issues.
// Fields:
//   - IssueID: optional external identifier, not used by correlation logic.
//   - Scanner, RuleID: identify the rule that produced the finding.
//   - Filename, StartLine, EndLine: location information inside a file.
//   - SnippetHash: optional code snippet/content fingerprint used for stronger matching.
type IssueMetadata struct {
	IssueID     string // issue id in external system or sequence number in report, just to know what issue it is outside of this module. Not used in correlation processing.
	Scanner     string
	RuleID      string
	Severity    string
	Filename    string
	StartLine   int
	EndLine     int
	SnippetHash string
}

// Match groups a known issue with the new issues that correlate to it.
// Match groups a single known issue with the list of new issues that were
// correlated to it. A new issue may appear in multiple Match.New slices if it
// correlates to multiple known issues.
type Match struct {
	Known IssueMetadata
	New   []IssueMetadata
}

// Correlator accepts slices of new and known issues and can compute correlations
// between them. Every known issue may match multiple new issues and vice versa.
// Correlator accepts slices of new and known issues and computes correlations
// between them. Use NewCorrelator to create an instance and call Process() to
// compute matches. After processing, use Matches(), UnmatchedNew() and
// UnmatchedKnown() to inspect results. The correlator preserves many-to-many
// relationships: a known issue may match multiple new issues and vice versa.
type Correlator struct {
	NewIssues   []IssueMetadata
	KnownIssues []IssueMetadata

	// internal indexes populated by Process()
	knownToNew map[int][]int // known index -> list of new indices
	newToKnown map[int][]int // new index -> list of known indices

	processed bool
}

// NewCorrelator creates a Correlator with the provided issues.
// NewCorrelator constructs a Correlator with the provided slices of new and
// known issues. The correlator is inert until Process() is called.
func NewCorrelator(newIssues, knownIssues []IssueMetadata) *Correlator {
	return &Correlator{
		NewIssues:   newIssues,
		KnownIssues: knownIssues,
	}
}

// Process computes correlations between every known and every new issue.
// Correlation strategy (in order):
// 1) If both issues have a non-empty SnippetHash and they are equal => match.
// 2) If Scanner, RuleID, Filename, StartLine and EndLine are all equal => match.
// 3) Fallback: Scanner, RuleID, Filename and StartLine equal => match.
// Process computes correlations between every known and every new issue using
// four ordered stages. Once a known or new issue has been matched in an
// earlier stage it is excluded from later stages. The stages are:
// 1) scanner+ruleid+filename+startline+endline+snippethash
// 2) scanner+ruleid+filename+snippethash
// 3) scanner+ruleid+filename+startline+endline
// 4) scanner+ruleid+filename+startline
// The results are stored internally and can be retrieved via Matches(),
// UnmatchedNew() and UnmatchedKnown(). Process is idempotent.
func (c *Correlator) Process() {
	if c.processed {
		return
	}
	c.knownToNew = make(map[int][]int)
	c.newToKnown = make(map[int][]int)

	// matchedBefore tracks indices already matched in earlier stages and
	// therefore excluded from later stages. matchedThisStage collects items
	// matched during the current stage so that multiple matches within the
	// same stage are allowed.
	matchedKnown := make(map[int]bool)
	matchedNew := make(map[int]bool)

	stages := []int{1, 2, 3, 4}
	for _, stage := range stages {
		matchedKnownThis := make(map[int]bool)
		matchedNewThis := make(map[int]bool)

		for ki, k := range c.KnownIssues {
			if matchedKnown[ki] {
				continue
			}
			for ni, n := range c.NewIssues {
				if matchedNew[ni] {
					continue
				}

				if matchStage(k, n, stage) {
					c.knownToNew[ki] = append(c.knownToNew[ki], ni)
					c.newToKnown[ni] = append(c.newToKnown[ni], ki)
					matchedKnownThis[ki] = true
					matchedNewThis[ni] = true
				}
			}
		}

		// promote this stage's matches to the global matched sets so they are
		// excluded from subsequent stages.
		for ki := range matchedKnownThis {
			matchedKnown[ki] = true
		}
		for ni := range matchedNewThis {
			matchedNew[ni] = true
		}
	}

	c.processed = true
}

// matchStage applies the specified stage matching rules. It returns true when
// the two IssueMetadata values should be considered a match for the given
// stage. The function enforces that Scanner and RuleID are present for all
// stages.
//
// Stage details:
// 1: scanner + ruleid + filename + startline + endline + snippethash
// 2: scanner + ruleid + filename + snippethash
// 3: scanner + ruleid + filename + startline + endline
// 4: scanner + ruleid + filename + startline
func matchStage(a, b IssueMetadata, stage int) bool {
	// require scanner and ruleid for all stages
	if a.Scanner == "" || b.Scanner == "" || a.RuleID == "" || b.RuleID == "" {
		return false
	}

	if a.Scanner != b.Scanner {
		return false
	}

	if a.RuleID != b.RuleID {
		return false
	}

	if a.Filename != b.Filename {
		return false
	}

	switch stage {
	case 1:
		return a.StartLine == b.StartLine && a.EndLine == b.EndLine && a.SnippetHash == b.SnippetHash
	case 2:
		return a.SnippetHash == b.SnippetHash
	case 3:
		return a.StartLine == b.StartLine && a.EndLine == b.EndLine
	case 4:
		return a.StartLine == b.StartLine
	default:
		return false
	}
}

// UnmatchedNew returns new issues that do not have any correlation to known issues.
// UnmatchedNew returns the subset of new issues that were not correlated to
// any known issue after Process() has been executed. If Process() has not
// yet been run it will be invoked.
func (c *Correlator) UnmatchedNew() []IssueMetadata {
	if !c.processed {
		c.Process()
	}

	var out []IssueMetadata
	for ni, n := range c.NewIssues {
		if len(c.newToKnown[ni]) == 0 {
			out = append(out, n)
		}
	}
	return out
}

// UnmatchedKnown returns known issues that do not have any correlation to new issues.
// UnmatchedKnown returns the subset of known issues that were not correlated
// to any new issue after Process() has been executed. If Process() has not
// yet been run it will be invoked.
func (c *Correlator) UnmatchedKnown() []IssueMetadata {
	if !c.processed {
		c.Process()
	}

	var out []IssueMetadata
	for ki, k := range c.KnownIssues {
		if len(c.knownToNew[ki]) == 0 {
			out = append(out, k)
		}
	}
	return out
}

// Matches returns a slice of Match entries. Each Match contains one known issue
// and the list of new issues that were correlated to it. A new issue that
// matches multiple known issues will appear under each matching known issue.
// Matches returns a slice of Match entries describing each known issue that
// had at least one correlated new issue. Each Match contains the known issue
// and the list of new issues correlated to it. If Process() has not been run
// it will be invoked.
func (c *Correlator) Matches() []Match {
	if !c.processed {
		c.Process()
	}

	var out []Match
	for ki, newIdxs := range c.knownToNew {
		if len(newIdxs) == 0 {
			continue
		}
		m := Match{Known: c.KnownIssues[ki], New: make([]IssueMetadata, 0, len(newIdxs))}
		for _, ni := range newIdxs {
			if ni >= 0 && ni < len(c.NewIssues) {
				m.New = append(m.New, c.NewIssues[ni])
			}
		}
		out = append(out, m)
	}
	return out
}
