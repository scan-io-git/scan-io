package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/sourcegraph/go-diff/diff"

	sharedfiles "github.com/scan-io-git/scan-io/pkg/shared/files"
	log "github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// AddedLines returns, for every file touched between baseHash and headHash, a map of
// new-file line numbers to the textual content that was added. Returned line
// numbers are 1-based and only include additions; deletions and context lines are
// ignored. Paths that are deleted or outside the optional filter list are skipped.
func AddedLines(gitClient *Client, repoPath, baseHash, headHash string, filters []string) (map[string]map[int]string, error) {
	if baseHash == "" {
		return nil, fmt.Errorf("base hash is required to compute diff")
	}
	if headHash == "" {
		return nil, fmt.Errorf("head hash is required to compute diff")
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository %q: %w", repoPath, err)
	}

	baseHashObj := plumbing.NewHash(baseHash)
	headHashObj := plumbing.NewHash(headHash)

	if err := ensureCommitPresent(gitClient, repo, baseHashObj); err != nil {
		return nil, fmt.Errorf("failed to resolve base commit %q: %w", baseHash, err)
	}
	if err := ensureCommitPresent(gitClient, repo, headHashObj); err != nil {
		return nil, fmt.Errorf("failed to resolve head commit %q: %w", headHash, err)
	}

	baseCommit, err := repo.CommitObject(baseHashObj)
	if err != nil {
		return nil, err
	}
	headCommit, err := repo.CommitObject(headHashObj)
	if err != nil {
		return nil, err
	}

	baseTree, err := baseCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to load base tree: %w", err)
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to load head tree: %w", err)
	}

	patch, err := baseTree.Patch(headTree)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	parsed, err := diff.ParseMultiFileDiff([]byte(patch.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	allowed := buildFilterSet(filters)
	result := make(map[string]map[int]string)

	for _, fd := range parsed {
		// checking deleted files and with no changes
		if fd == nil || fd.NewName == "/dev/null" || len(fd.Hunks) == 0 {
			continue
		}

		path := strings.TrimPrefix(fd.NewName, "b/")
		if len(allowed) > 0 && !allowed[path] {
			continue
		}

		added := make(map[int]string)

		for _, h := range fd.Hunks {
			if h == nil {
				continue
			}
			lineNo := int(h.NewStartLine)
			if lineNo <= 0 {
				lineNo = 1
			}
			for _, bodyLine := range bytes.Split(h.Body, []byte("\n")) {
				if len(bodyLine) == 0 {
					continue
				}

				switch bodyLine[0] {
				case '+':
					added[lineNo] = string(bodyLine[1:])
					lineNo++
				case '-':
					// deletion; do not advance new file line counter
					continue
				default:
					lineNo++
				}
			}
		}

		if len(added) > 0 {
			result[path] = added
		}
	}

	return result, nil
}

// MaterializeDiff writes diff-focused copies of provided files into diffRoot. Every
// output file mirrors the repository structure but contains only the newly added
// lines (other positions remain blank), allowing scanners to operate on diff
// hunks without re-running git diff. When no additions are detected the function
// exits early without writing anything.
func MaterializeDiff(gitClient *Client, repoRoot, diffRoot, baseSHA, headSHA string, files []string) error {
	if err := sharedfiles.CreateFolderIfNotExists(diffRoot); err != nil {
		return fmt.Errorf("prepare diff folder: %w", err)
	}

	paths := uniqueNonEmpty(files)
	addedLines, err := AddedLines(gitClient, repoRoot, baseSHA, headSHA, paths)
	if err != nil {
		return err
	}

	if len(addedLines) == 0 {
		gitClient.logger.Info("no additions detected between commits", "base", baseSHA, "head", headSHA)
		return nil
	}

	if len(paths) == 0 {
		paths = sortedKeys(addedLines)
	}

	for _, relPath := range paths {
		relPath = strings.TrimSpace(relPath)
		if relPath == "" {
			continue
		}

		lines := addedLines[relPath]
		if len(lines) == 0 {
			gitClient.logger.Debug("skipping file with no additions", "path", relPath)
			continue
		}

		if err := writeSparseFile(repoRoot, diffRoot, relPath, lines); err != nil {
			return err
		}
	}

	return nil
}

// writeSparseFile writes a copy of relPath into diffRoot, keeping only the line
// numbers present in the supplied map and leaving other positions empty. The file
// retains trailing newline semantics from the source to minimise surprises for
// downstream tools.
func writeSparseFile(repoRoot, diffRoot, relPath string, lines map[int]string) error {
	srcPath := filepath.Join(repoRoot, relPath)
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read %q for diff materialisation: %w", relPath, err)
	}

	content := string(data)
	headLines := strings.Split(content, "\n")
	diffLines := make([]string, len(headLines))

	for lineNumber, value := range lines {
		if lineNumber <= 0 {
			continue
		}
		index := lineNumber - 1
		if index >= len(diffLines) {
			diffLines = append(diffLines, make([]string, index-len(diffLines)+1)...)
		}
		diffLines[index] = value
	}

	output := strings.Join(diffLines, "\n")
	if strings.HasSuffix(content, "\n") && !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	dstPath := filepath.Join(diffRoot, relPath)
	if err := sharedfiles.CreateFolderIfNotExists(filepath.Dir(dstPath)); err != nil {
		return fmt.Errorf("failed to prepare folder for %q: %w", relPath, err)
	}

	if err := os.WriteFile(dstPath, []byte(output), 0600); err != nil {
		return fmt.Errorf("failed to write diff file %q: %w", dstPath, err)
	}

	return nil
}

// buildFilterSet returns an O(1) lookup table for the provided filter slice.
// Nil is returned when no filters are supplied to avoid extra map checks downstream.
func buildFilterSet(filters []string) map[string]bool {
	if len(filters) == 0 {
		return nil
	}
	set := make(map[string]bool, len(filters))
	for _, f := range filters {
		set[f] = true
	}
	return set
}

// uniqueNonEmpty strips empty entries and duplicates from the path list while
// preserving the original order. Paths are kept verbatim (no trimming) so values
// containing leading/trailing spaces remain intact.
func uniqueNonEmpty(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, exists := set[item]; exists {
			continue
		}
		set[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

// sortedKeys returns the map keys in ascending order to provide deterministic
// iteration when no explicit filter list was provided.
func sortedKeys(m map[string]map[int]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func ensureCommitPresent(gitClient *Client, repo *git.Repository, hash plumbing.Hash) error {
	if _, err := repo.CommitObject(hash); err != nil {
		gitClient.logger.Debug("commit missing locally, attempting fetch", "hash", hash.String())
		if err := fetchCommit(gitClient, repo, hash); err != nil {
			return err
		}
	}
	return nil
}

func fetchCommit(gitClient *Client, repo *git.Repository, hash plumbing.Hash) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitClient.timeout)
	defer cancel()

	gitLog := log.GetLoggerOutput(gitClient.logger)
	output := io.MultiWriter(
		gitLog,
		os.Stderr,
	)

	insecure := InsecureFromCfg(gitClient.globalConfig)

	remoteName := origin
	if _, err := repo.Remote(remoteName); err != nil {
		remotes, rErr := repo.Remotes()
		if rErr != nil || len(remotes) == 0 {
			return fmt.Errorf("no remotes available to fetch commit %s", hash.String())
		}
		remoteName = remotes[0].Config().Name
	}

	tmpRef := plumbing.ReferenceName(fmt.Sprintf(tmpRefPrefix+"%s", hash.String()))
	refspec := config.RefSpec(fmt.Sprintf("+%s:%s", hash.String(), tmpRef.String()))

	if gitClient.logger != nil {
		gitClient.logger.Debug("fetching commit", "remote", remoteName, "hash", hash.String())
	}

	fetchErr := repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName:      remoteName,
		Auth:            gitClient.auth,
		InsecureSkipTLS: insecure,
		Progress:        output,
		Depth:           1,
		RefSpecs:        []config.RefSpec{refspec},
		Tags:            git.NoTags,
	})

	if fetchErr != nil && fetchErr != git.NoErrAlreadyUpToDate {
		if fetchErr != nil {
			if fetchErr == git.NoErrAlreadyUpToDate {
				gitClient.logger.Debug("commit already available", "hash", hash.String())
			} else {
				gitClient.logger.Warn("fetch commit failed", "hash", hash.String(), "error", fetchErr)
				return fetchErr
			}
		}
	}

	defer func() {
		_ = repo.Storer.RemoveReference(tmpRef)
	}()

	if _, err := repo.CommitObject(hash); err != nil {
		return err
	}
	gitClient.logger.Debug("commit fetched", "hash", hash.String())
	return nil
}
