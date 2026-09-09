package tohtml

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"
)

// reportStateID returns a stable identifier for this report, or "" when the
// report cannot be identified.
//
// The browser files a reader's severity and scanner filters under this id, so it
// has to change exactly when the findings change. The commit hash carries that:
// re-rendering the same scan yields the same id and the reader's filters come
// back, while a new commit yields a new id and an unfiltered report — correct,
// because the findings are no longer the same ones the filters were chosen for.
// Nothing per-render (time, CSP nonce) may enter it, or every render would look
// like a different report.
//
// An empty result means "do not persist". Without a commit nothing pins the
// report to a code state, and inventing a shared fallback id would restore one
// report's filters onto another — the leak this identifier exists to prevent.
func reportStateID(repoFullName, prID, commitHash string, scanners []string, suppressionsRemoved bool) string {
	if strings.TrimSpace(commitHash) == "" {
		return ""
	}
	// The scanner set is part of the identity: one commit can produce several
	// reports (a merged one, a single-scanner one), and those hold different
	// findings, so a severity chosen against one must not be reapplied to another.
	// Sorted, because the report is the same whatever order the inputs arrived in.
	sorted := make([]string, len(scanners))
	copy(sorted, scanners)
	sort.Strings(sorted)

	// --no-supressions strips results before rendering, so the two variants of one
	// commit hold different finding sets and must not share filter state. Only
	// inputs that change *which findings exist* belong here: --required and
	// --never-demote annotate and reorder without removing anything, so a filter
	// stays equally valid across those variants and they share an id by design.
	suppressions := "suppressions:kept"
	if suppressionsRemoved {
		suppressions = "suppressions:removed"
	}

	// NUL-separated so ("ab","c") and ("a","bc") cannot collide.
	fields := append([]string{repoFullName, prID, commitHash, suppressions}, sorted...)
	sum := sha256.Sum256([]byte(strings.Join(fields, "\x00")))
	return hex.EncodeToString(sum[:8])
}

// resolveReportStateID pulls the identifying fields out of the report's metadata,
// tolerating the nils that appear when a scan runs without a source folder.
// Separate from reportStateID so the hashing stays a pure function of strings and
// can be tested without building git or VCS metadata.
func resolveReportStateID(
	repoMeta *git.RepositoryMetadata,
	parsedURL *vcsurl.VCSURL,
	commitHash string,
	scanners []string,
	suppressionsRemoved bool,
) string {
	repoFullName := ""
	if repoMeta != nil && repoMeta.RepositoryFullName != nil {
		repoFullName = *repoMeta.RepositoryFullName
	}
	prID := ""
	if parsedURL != nil {
		prID = parsedURL.PullRequestId
	}
	return reportStateID(repoFullName, prID, commitHash, scanners, suppressionsRemoved)
}
