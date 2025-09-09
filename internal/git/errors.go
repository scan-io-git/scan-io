package git

import "errors"

// Repo errors
var (
	ErrDifferentRepo  = errors.New("target folder contains a different repo")
	ErrRecloneConsent = errors.New("corrupted/shallow repository detected. Repair requires user consent. Re-run the command with '--auto-repair' to allow automatic recovery.")
)

// Target resolution errors
var (
	ErrDefaultBranchHead     = errors.New("failed to resolve default branch from HEAD")
	ErrShortCommitSHA        = errors.New("short commit SHA not supported")
	ErrUnsupportedTargetKind = errors.New("unsupported target kind")
)
