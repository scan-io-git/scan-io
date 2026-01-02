package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/hashicorp/go-hclog"

	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"
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

type AutoDetectedParams struct {
	Namespace  string
	Repository string
	Reference  string
}

// ApplyGitMetadataOptionsFallbacks applies git metadata fallbacks to the run options.
// It extracts namespace, repository, and ref from local git repository metadata
// when the corresponding flags are not already provided.
func ApplyGitMetadataOptionsFallbacks(logger hclog.Logger, sourceFolder, namespace, repository, VCSPluginName, ref string) (AutoDetectedParams, error) {
	detectedParams := AutoDetectedParams{}
	// Determine the base folder for git metadata extraction
	baseFolder := strings.TrimSpace(sourceFolder)
	if baseFolder == "" {
		// Use current working directory if source-folder is not provided
		if cwd, err := os.Getwd(); err == nil {
			baseFolder = cwd
		} else {
			return detectedParams, fmt.Errorf("failed to get current working directory for git metadata extraction: %w", err)
		}
	}

	repoMetadata, err := CollectRepositoryMetadata(baseFolder)
	if err != nil {
		return detectedParams, fmt.Errorf("unable to collect git repository metadata for baseFolder %v: %v", err)
	}

	// Extract namespace and repository from git remote URL if not already set
	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(repository) == "" {
		if repoMetadata.RepositoryFullName != nil && *repoMetadata.RepositoryFullName != "" {
			// Parse the repository URL to extract namespace and repository
			vcsType := vcsurl.StringToVCSType(VCSPluginName)
			vcsURL, err := vcsurl.ParseForVCSType(*repoMetadata.RepositoryFullName, vcsType)
			if err != nil {
				logger.Debug("failed to parse git repository URL", "error", err, "url", *repoMetadata.RepositoryFullName)
			} else {
				// Apply namespace if not already set
				if strings.TrimSpace(namespace) == "" && vcsURL.Namespace != "" {
					detectedParams.Namespace = vcsURL.Namespace
					logger.Debug("auto-detected namespace from git metadata", "namespace", vcsURL.Namespace)
				}

				// Apply repository if not already set
				if strings.TrimSpace(repository) == "" && vcsURL.Repository != "" {
					detectedParams.Repository = vcsURL.Repository
					logger.Debug("auto-detected repository from git metadata", "repository", vcsURL.Repository)
				}
			}
		}
	}

	// Extract commit hash for ref if not already set
	if strings.TrimSpace(ref) == "" {
		if repoMetadata.CommitHash != nil && *repoMetadata.CommitHash != "" {
			detectedParams.Reference = *repoMetadata.CommitHash
			logger.Debug("auto-detected ref from git metadata", "ref", *repoMetadata.CommitHash)
		}
	}
	return detectedParams, nil
}
