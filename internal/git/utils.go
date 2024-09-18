package git

import (
	"fmt"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
)

type RefType struct {
	Branch   plumbing.ReferenceName
	Hash     plumbing.Hash
	IsCommit bool
}

// determineBranch returns the appropriate branch or commit reference.
func determineBranch(branch string, cloneURL string, auth *transport.AuthMethod) (RefType, error) {
	var result RefType

	// If the branch is explicitly provided, return it as the reference
	if branch != "" {
		if plumbing.IsHash(branch) {
			result.Hash = plumbing.NewHash(branch)
			result.IsCommit = true
			return result, nil
		}
		result.Branch = plumbing.ReferenceName(branch)
		// Ensure we avoid double concatenation of refs if it already looks like a ref
		if !result.Branch.IsBranch() && !result.Branch.IsRemote() && !result.Branch.IsTag() && !result.Branch.IsNote() {
			result.Branch = plumbing.NewBranchReferenceName(result.Branch.String())
		}
		return result, nil
	}

	// No branch provided, resolve the default branch by fetching the remote's HEAD
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{cloneURL},
	})

	refs, err := remote.List(&git.ListOptions{
		Auth: *auth,
	})
	if err != nil {
		return result, fmt.Errorf("failed to list remote references: %w", err)
	}

	// Find the reference that HEAD points to (default branch)
	for _, ref := range refs {
		if ref.Name() == plumbing.HEAD {
			result.Branch = ref.Target() // This is the default branch (refs/heads/main or refs/heads/master)
			return result, nil
		}
	}

	return result, fmt.Errorf("failed to resolve default branch from HEAD")
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
