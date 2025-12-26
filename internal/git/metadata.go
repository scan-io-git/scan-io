package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

// struct with repository metadata
type RepositoryMetadata struct {
	BranchName         *string
	CommitHash         *string
	RepositoryFullName *string
	Subfolder          string
	RepoRootFolder     string
}

// CollectRepositoryMetadata function collects repository metadata
// that includes branch name, commit hash, repository full name, subfolder and repository root folder
func CollectRepositoryMetadata(sourceFolder string) (*RepositoryMetadata, error) {
	if sourceFolder == "" {
		return &RepositoryMetadata{}, fmt.Errorf("source folder is not set")
	}

	if absSource, err := filepath.Abs(sourceFolder); err == nil {
		sourceFolder = absSource
	}

	md := &RepositoryMetadata{
		RepoRootFolder: filepath.Clean(sourceFolder),
	}

	repoRootFolder, err := findGitRepositoryPath(sourceFolder)
	if err != nil {
		return md, err
	}

	md.RepoRootFolder = filepath.Clean(repoRootFolder)

	repo, err := git.PlainOpen(repoRootFolder)
	if err != nil {
		return md, fmt.Errorf("failed to open repository: %w", err)
	}

	if rel, err := filepath.Rel(repoRootFolder, sourceFolder); err == nil && rel != "." {
		md.Subfolder = filepath.ToSlash(rel)
	}

	if head, err := repo.Head(); err == nil {
		if head.Name().IsBranch() {
			branchName := head.Name().Short()
			md.BranchName = &branchName
		}

		hash := head.Hash().String()
		md.CommitHash = &hash
	}

	if remote, err := repo.Remote("origin"); err == nil {
		if cfg := remote.Config(); cfg != nil && len(cfg.URLs) > 0 {
			repositoryFullName := strings.TrimSuffix(cfg.URLs[0], ".git")
			md.RepositoryFullName = &repositoryFullName
		}
	}

	return md, nil
}
