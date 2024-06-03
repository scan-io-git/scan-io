package git

import (
	"github.com/go-git/go-git/v5/plumbing"
)

// determineBranch returns the appropriate branch reference.
func determineBranch(branch, defaultBranch string) plumbing.ReferenceName {
	if branch == "" {
		branch = defaultBranch
	}
	ref := plumbing.ReferenceName(branch)
	if !ref.IsBranch() && !ref.IsRemote() && !ref.IsTag() && !ref.IsNote() {
		return plumbing.NewBranchReferenceName(branch)
	}
	return ref
}
