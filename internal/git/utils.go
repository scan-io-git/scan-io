package git

import (
	"fmt"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type RefType struct {
	Branch   plumbing.ReferenceName
	Hash     plumbing.Hash
	IsCommit bool
}

// determineBranch returns the appropriate branch reference.
func determineBranch(branch, defaultBranch string) RefType {
	var result RefType
	var ref plumbing.ReferenceName
	if branch == "" {
		branch = defaultBranch
	}

	if plumbing.IsHash(branch) {
		result.Hash = plumbing.NewHash(branch)
		result.IsCommit = true
	}

	if result.IsCommit {
		ref = plumbing.ReferenceName(defaultBranch)
	} else {
		ref = plumbing.ReferenceName(branch)
	}

	if !ref.IsBranch() && !ref.IsRemote() && !ref.IsTag() && !ref.IsNote() {
		result.Branch = plumbing.NewBranchReferenceName(ref.String())
		return result
	}
	result.Branch = ref
	return result
}

// findGitRepositoryPath function finds a git repository path for a given source folder
func findGitRepositoryPath(sourceFolder string) (string, error) {
	if sourceFolder == "" {
		return "", fmt.Errorf("source folder is not set")
	}

	// check if source folder is a subfolder of a git repository
	for {
		_, err := git.PlainOpen(sourceFolder)
		if err == nil {
			return sourceFolder, nil
		}

		// move up one level
		sourceFolder = filepath.Dir(sourceFolder)

		// check if reached the root folder
		if sourceFolder == filepath.Dir(sourceFolder) {
			break
		}
	}

	return "", fmt.Errorf("source folder is not a git repository")
}
