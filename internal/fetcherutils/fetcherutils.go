package fetcherutils

import "fmt"

// FetchScope defines the amount of repository data materialised during fetch.
type FetchScope string

const (
	// ScopeFull fetches full repository contents (default behaviour).
	ScopeFull FetchScope = "full"
	// ScopeDiff derive changes from a reference needed for diff-based scans and place it to a separated folder.
	ScopeDiff FetchScope = "diff"
)

func (s FetchScope) String() string {
	return string(s)
}

func ResolveFetchScope(diff bool) FetchScope {
	if diff {
		return ScopeDiff
	}
	return ScopeFull
}

// FetchMode defines the strategy used when fetching a repository.
type FetchMode int

const (
	// ModeDefault is used for non-PR fetching: direct commit or branch reference.
	ModeDefault FetchMode = iota

	// PRBranchMode fetches from a feature branch in the same repository.
	PRBranchMode

	// PRRefMode fetches from a special PR reference (e.g., /refs/pull/%s/head).
	PRRefMode

	// PRCommitMode fetches using a detached commit from the PR.
	PRCommitMode
)

var fetchModeToString = map[FetchMode]string{
	ModeDefault:  "basic",
	PRBranchMode: "pull-branch",
	PRRefMode:    "pull-ref",
	PRCommitMode: "pull-commit",
}

func (m FetchMode) String() string {
	if s, ok := fetchModeToString[m]; ok {
		return s
	}
	return fmt.Sprintf("unknown(%d)", m)
}

// ParsePRFetchMode parses a string into a FetchMode for PR fetching.
// Returns PRBranchMode if input is empty or invalid.
func ParsePRFetchMode(input string) (FetchMode, error) {
	if input == "" {
		return PRBranchMode, nil
	}
	prFetchMode := fmt.Sprintf("pull-%s", input)
	for _, mode := range []FetchMode{PRBranchMode, PRRefMode, PRCommitMode} {
		if mode.String() == prFetchMode {
			return mode, nil
		}
	}
	return PRBranchMode, fmt.Errorf("invalid pr fetch mode: %q", input)
}

// GetFetchMode determines the appropriate FetchMode for a repository fetch.
// If no PR ID is provided, it defaults to ModeDefault.
// If the mode string is invalid, it defaults to PRBranchMode and returns an error.
func GetFetchMode(prID, fetchMode string) (FetchMode, error) {
	if prID == "" {
		return ModeDefault, nil
	}

	mode, err := ParsePRFetchMode(fetchMode)
	if err != nil {
		return mode, fmt.Errorf("%w. Defaulting to \"pull-branch\" for pr fetch mode", err)
	}

	return mode, nil
}
