package fetcherutils

import "fmt"

type FetchMode int

const (
	ModeDefault  FetchMode = iota // non-pr fetching: direct provided commit or ref
	PRBranchMode                  // pr fetching: via feature branch
	PRRefMode                     // pr fetching: via pr special branch /refs/pull/%s/head
	PRCommitMode                  // pr fetching: via detached latest commit
)

func (m FetchMode) String() string {
	switch m {
	case ModeDefault:
		return "basic"
	case PRBranchMode:
		return "pull-branch"
	case PRRefMode:
		return "pull-ref"
	case PRCommitMode:
		return "pull-commit"
	default:
		return "unknown"
	}
}

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

// GetFetchMode determines the mode for the fetch request.
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
