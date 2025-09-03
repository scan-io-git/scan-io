package git

import "errors"

// Repo errors
var (
	ErrDifferentRepo  = errors.New("target folder contains a different repo")
	ErrRecloneConsent = errors.New("reclone requires user consent")
)

// Target resolution errors
var (
	ErrDefaultBranchHead     = errors.New("failed to resolve default branch from HEAD")
	ErrShortCommitSHA        = errors.New("short commit SHA not supported")
	ErrUnsupportedTargetKind = errors.New("unsupported target kind")
)
