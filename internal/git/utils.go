package git

import (
	"fmt"
	"path/filepath"

	"github.com/go-git/go-git/v5"
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
