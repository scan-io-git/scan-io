package git

import (
	"fmt"
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
	defaultRepositoryMetadata := &RepositoryMetadata{
		RepoRootFolder: sourceFolder,
		Subfolder:      "",
	}

	if sourceFolder == "" {
		return defaultRepositoryMetadata, fmt.Errorf("source folder is not set")
	}

	repoRootFolder, err := findGitRepositoryPath(sourceFolder)
	if err != nil {
		return defaultRepositoryMetadata, err
	}

	repo, err := git.PlainOpen(repoRootFolder)
	if err != nil {
		return defaultRepositoryMetadata, fmt.Errorf("failed to open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return defaultRepositoryMetadata, fmt.Errorf("failed to get HEAD: %w", err)
	}
	branchName := head.Name().Short()
	commitHash := head.Hash().String()

	remote, err := repo.Remote("origin")
	if err != nil {
		return defaultRepositoryMetadata, fmt.Errorf("failed to get remote: %w", err)
	}

	repositoryFullName := strings.TrimSuffix(remote.Config().URLs[0], ".git")

	return &RepositoryMetadata{
		BranchName:         &branchName,
		CommitHash:         &commitHash,
		RepositoryFullName: &repositoryFullName,
		Subfolder:          strings.TrimPrefix(sourceFolder, repoRootFolder),
		RepoRootFolder:     repoRootFolder,
	}, nil
}
