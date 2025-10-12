# SARIF Issues Path Handling Analysis

## Reproduction Context
- Command sequence:
  1. `scanio analyse --scanner semgrep /home/jekos/ghq/github.com/scan-io-git/scanio-test/apps/demo --format sarif --output outside-project.sarif`
  2. `scanio sarif-issues --namespace scan-io-git --repository scanio-test --ref aec0b795c350ff53fe9ab01adf862408aa34c3fd --sarif from-subfolder.sarif --source-folder /home/jekos/ghq/github.com/scan-io-git/scanio-test/apps/demo`
- Expected permalink: `.../blob/aec0b795c350ff53fe9ab01adf862408aa34c3fd/apps/demo/main.py#L11-L29`
- Actual permalink (incorrect): `.../blob/aec0b795c350ff53fe9ab01adf862408aa34c3fd/main.py#L11-L29`

## Key Observations
- `data/outside-project.sarif` contains absolute URIs such as `/home/.../scanio-test/apps/demo/main.py`.
- `data/from-subfolder.sarif` contains relative URIs (`main.py`) because Semgrep ran from the subfolder.
- In both cases the SARIF report points to the file under `apps/demo/main.py`, yet the CLI emits `main.py` in issue bodies and permalinks.

## Code Flow Review
- `cmd/sarif-issues/issue_processing.go` calls `extractFileURIFromResult` to determine the file path recorded in `NewIssueData` (`buildNewIssuesFromSARIF`, line references around `fileURI` usage).
- `extractFileURIFromResult` (`cmd/sarif-issues/utils.go:173-212`) trims the `--source-folder` prefix from absolute URIs and returns the remainder; for relative URIs it simply returns the raw value.
  - When `--source-folder` is `/.../scanio-test/apps/demo`, absolute URIs reduce to `main.py`, losing the repository subpath.
- `buildGitHubPermalink` (`utils.go:125-170`) expects `fileURI` to be repository-relative when constructing `https://github.com/{namespace}/{repo}/blob/{ref}/{fileURI}#L...`.
- `computeSnippetHash` (`utils.go:104-121`) relies on joining `sourceFolder` with the same `fileURI` to re-read the local file. If we change `fileURI` to be repo-relative (`apps/demo/main.py`), the current join logic will point at `/.../apps/demo/apps/demo/main.py` and fail.
- `internal/sarif.Report.EnrichResultsLocationProperty` and `EnrichResultsLocationURIProperty` perform similar prefix stripping using `sourceFolder`, so the HTML report path logic (`cmd/to-html.go`) inherits the same limitation.
- `internal/git.CollectRepositoryMetadata` already derives `RepoRootFolder` and the `Subfolder` path segment when `--source-folder` is nested within the repo.

## Root Cause
The CLI assumes `--source-folder` equals the repository root. When the user points it to a subdirectory, the helper trims that prefix and drops intermediate path segments. Consequently:
- Issue metadata (`File` field) loses the directory context.
- GitHub permalinks omit the subfolder and land on the wrong file.
- Correlation metadata (`Metadata.Filename`) no longer matches the path stored in GitHub issues, risking mismatches if/when we fix the permalink logic without updating correlation.

## Fix Considerations
1. **Determine repository root & subfolder once.** `internal/git.CollectRepositoryMetadata` gives us both `RepoRootFolder` and `Subfolder` for any path inside the repo. Reusing this keeps CLI logic consistent with the HTML report command.
2. **Produce dual path representations.**
   - Repo-relative path (e.g. `apps/demo/main.py`) for GitHub URLs and issue bodies.
   - Source-folder-relative path (e.g. `main.py`) or absolute path for reading files/snippet hashing.
3. **Avoid regressions in existing flows.** After changing `fileURI`, ensure:
   - `computeSnippetHash` receives the correct on-disk path.
   - Issue correlation (`Metadata.Filename`) uses the same representation that is stored in GitHub issue bodies to preserve matching.
4. **Consider harmonising SARIF helpers.** Updating `internal/sarif` enrichment to use repo metadata would fix both CLI commands (`sarif-issues`, `to-html`) and reduce duplicated path trimming logic.

## Proposed Fix Plan
1. Enhance the `sarif-issues` command to collect repository metadata:
   - Call `git.CollectRepositoryMetadata(opts.SourceFolder)` early (guard for errors).
   - Derive helper closures that can translate between repo-relative and local paths.
2. Update `extractFileURIFromResult` (or an adjacent helper) to:
   - Resolve the SARIF URI to an absolute path (using `uriBaseId` and `sourceFolder` when necessary).
   - Emit the repo-relative path (using metadata.RepoRootFolder) for issue content and permalinks.
   - Return both repo-relative and local paths, or store them in a small struct to avoid repeated conversions.
3. Adjust `computeSnippetHash` and correlation metadata to consume the correct local path while storing repo-relative filenames in issue metadata.
4. Reuse the new path helper in `buildGitHubPermalink` so the permalink path stays in sync.
5. Add regression tests:
   - Extend `cmd/sarif-issues/utils_test.go` (or introduce new tests) covering absolute and relative SARIF URIs when `sourceFolder` points to a subdirectory.
   - Include permalink assertions using `data/from-subfolder.sarif` / `data/outside-project.sarif`.
6. Evaluate whether `internal/sarif`’s enrichment should adopt the same metadata-aware logic; if so, share the helper to keep `to-html` and future commands consistent.

# Manual testing
## Scans from root, subfolder, outside, with abs and relative paths
### Semgrep scan of subfolder (monorepo like use case)
```sh
# 1. Outside folder absolute paths
cd /home/jekos/ghq/github.com/scan-io-git/scan-io
scanio analyse --scanner semgrep /home/jekos/ghq/github.com/scan-io-git/scanio-test/apps/demo --format sarif --output /home/jekos/ghq/github.com/scan-io-git/scan-io/data/outside-project-abs.sarif
scanio sarif-issues --namespace scan-io-git --repository scanio-test --ref aec0b795c350ff53fe9ab01adf862408aa34c3fd --sarif data/outside-project-abs.sarif --source-folder /home/jekos/ghq/github.com/scan-io-git/scanio-test/apps/demo
# validate here: 2 issues with correct permalinks
# correct: https://github.com/scan-io-git/scanio-test/blob/aec0b795c350ff53fe9ab01adf862408aa34c3fd/apps/demo/main.py

# 2. Outside folder relative paths
cd /home/jekos/ghq/github.com/scan-io-git/scan-io
scanio analyse --scanner semgrep ../scanio-test/apps/demo --format sarif --output data/outside-project-rel.sarif
scanio sarif-issues --namespace scan-io-git --repository scanio-test --ref aec0b795c350ff53fe9ab01adf862408aa34c3fd --sarif data/outside-project-rel.sarif --source-folder ../scanio-test/apps/demo
# validate here: 2 issues with correct permalinks
# correct: https://github.com/scan-io-git/scanio-test/blob/aec0b795c350ff53fe9ab01adf862408aa34c3fd/apps/demo/main.py

# 3. From root absolute path
cd /home/jekos/ghq/github.com/scan-io-git/scanio-test
scanio analyse --scanner semgrep /home/jekos/ghq/github.com/scan-io-git/scanio-test/apps/demo --format sarif --output /home/jekos/ghq/github.com/scan-io-git/scan-io/data/from-root-asb.sarif
scanio sarif-issues --namespace scan-io-git --repository scanio-test --ref aec0b795c350ff53fe9ab01adf862408aa34c3fd --sarif /home/jekos/ghq/github.com/scan-io-git/scan-io/data/from-root-asb.sarif --source-folder /home/jekos/ghq/github.com/scan-io-git/scanio-test/apps/demo
# validate here: 2 issues with correct permalinks
# correct: https://github.com/scan-io-git/scanio-test/blob/aec0b795c350ff53fe9ab01adf862408aa34c3fd/apps/demo/main.py

# 4. From root relative paths
cd /home/jekos/ghq/github.com/scan-io-git/scanio-test
scanio analyse --scanner semgrep apps/demo --format sarif --output /home/jekos/ghq/github.com/scan-io-git/scan-io/data/from-root-rel.sarif
scanio sarif-issues --namespace scan-io-git --repository scanio-test --ref aec0b795c350ff53fe9ab01adf862408aa34c3fd --sarif /home/jekos/ghq/github.com/scan-io-git/scan-io/data/from-root-rel.sarif --source-folder apps/demo
# validate here: 2 issues with correct permalinks
# correct https://github.com/scan-io-git/scanio-test/blob/aec0b795c350ff53fe9ab01adf862408aa34c3fd/apps/demo/main.py
# correct even when .git folder is not there

# 5. From subfolder absolute paths
cd /home/jekos/ghq/github.com/scan-io-git/scanio-test/apps/demo
scanio analyse --scanner semgrep /home/jekos/ghq/github.com/scan-io-git/scanio-test/apps/demo --format sarif --output /home/jekos/ghq/github.com/scan-io-git/scan-io/data/from-subfolder-abs.sarif
scanio sarif-issues --namespace scan-io-git --repository scanio-test --ref aec0b795c350ff53fe9ab01adf862408aa34c3fd --sarif /home/jekos/ghq/github.com/scan-io-git/scan-io/data/from-subfolder-abs.sarif --source-folder /home/jekos/ghq/github.com/scan-io-git/scanio-test/apps/demo
# validate here: 2 issues with correct permalinks
# correct: https://github.com/scan-io-git/scanio-test/blob/aec0b795c350ff53fe9ab01adf862408aa34c3fd/apps/demo/main.py

# 6. From subfolder relative paths
cd /home/jekos/ghq/github.com/scan-io-git/scanio-test/apps/demo
scanio analyse --scanner semgrep . --format sarif --output /home/jekos/ghq/github.com/scan-io-git/scan-io/data/from-subfolder-rel.sarif
scanio sarif-issues --namespace scan-io-git --repository scanio-test --ref aec0b795c350ff53fe9ab01adf862408aa34c3fd --sarif /home/jekos/ghq/github.com/scan-io-git/scan-io/data/from-subfolder-rel.sarif --source-folder .
# validate here: 2 issues with correct permalinks
# correct: https://github.com/scan-io-git/scanio-test/blob/aec0b795c350ff53fe9ab01adf862408aa34c3fd/apps/demo/main.py
```
### snyk
```sh
cd /home/jekos/ghq/github.com/scan-io-git/scanio-test

# 1. scan root
snyk code test --sarif-file-output=/home/jekos/ghq/github.com/scan-io-git/scan-io/data/snyk-root.sarif
scanio sarif-issues --namespace scan-io-git --repository scanio-test --ref aec0b795c350ff53fe9ab01adf862408aa34c3fd --sarif /home/jekos/ghq/github.com/scan-io-git/scan-io/data/snyk-root.sarif --source-folder .

# 2. scan subfolder from root
snyk code test --sarif-file-output=/home/jekos/ghq/github.com/scan-io-git/scan-io/data/snyk-subfolder-from-root.sarif apps/demo
scanio sarif-issues --namespace scan-io-git --repository scanio-test --ref aec0b795c350ff53fe9ab01adf862408aa34c3fd --sarif /home/jekos/ghq/github.com/scan-io-git/scan-io/data/snyk-subfolder-from-root.sarif --source-folder apps/demo
```
### codeql
```sh
cd /home/jekos/ghq/github.com/scan-io-git/scanio-test

# 1. scan root
/tmp/codeql/codeql database create /home/jekos/ghq/github.com/scan-io-git/scan-io/data/codeql-database-root --language=python --source-root=.
/tmp/codeql/codeql database analyze /home/jekos/ghq/github.com/scan-io-git/scan-io/data/codeql-database-root --format=sarif-latest --output=/home/jekos/ghq/github.com/scan-io-git/scan-io/data/codeql-root.sarif
scanio sarif-issues --namespace scan-io-git --repository scanio-test --ref aec0b795c350ff53fe9ab01adf862408aa34c3fd --sarif /home/jekos/ghq/github.com/scan-io-git/scan-io/data/codeql-root.sarif --source-folder .

# 1. scan subfolder
/tmp/codeql/codeql database create /home/jekos/ghq/github.com/scan-io-git/scan-io/data/codeql-database-subfolder --language=python --source-root=apps/demo
/tmp/codeql/codeql database analyze /home/jekos/ghq/github.com/scan-io-git/scan-io/data/codeql-database-subfolder --format=sarif-latest --output=/home/jekos/ghq/github.com/scan-io-git/scan-io/data/codeql-subfolder.sarif
scanio sarif-issues --namespace scan-io-git --repository scanio-test --ref aec0b795c350ff53fe9ab01adf862408aa34c3fd --sarif /home/jekos/ghq/github.com/scan-io-git/scan-io/data/codeql-subfolder.sarif --source-folder apps/demo
```
## How to handle 2 subfolders with 2 separate scans
```sh
cd /home/jekos/ghq/github.com/scan-io-git/scanio-test

# scan projects
scanio analyse --scanner semgrep apps/demo --format sarif --output semgrep-demo.sarif
snyk code test --sarif-file-output=snyk-another.sarif apps/another

# create issues
scanio sarif-issues --sarif semgrep-demo.sarif --source-folder apps/demo
scanio sarif-issues --sarif snyk-another.sarif --source-folder apps/another
```

**Solution Implemented**: The `sarif-issues` command now filters open issues by source folder scope before correlation. Issues are scoped based on their file path metadata matching the normalized subfolder path. This enables independent issue management for different subfolders in monorepo CI workflows.

**Key Changes**:
- Added `filterIssuesBySourceFolder()` function that filters open issues to only those within the current `--source-folder` scope
- Issues are filtered before correlation, ensuring each subfolder's issues are managed independently
- When `--source-folder` points to a subfolder, only issues whose file paths start with that subfolder are considered
- When scanning from root (no subfolder), all issues are included as before

**Expected Behavior**: Both sets of issues remain open and are managed independently. Issues from `apps/demo` won't be closed when running the second command for `apps/another`.

## empty source-folder test
```sh
cd /home/jekos/ghq/github.com/scan-io-git/scanio-test

snyk code test --sarif-file-output=snyk.sarif

# create issues
scanio sarif-issues --sarif snyk.sarif --source-folder .
scanio sarif-issues --sarif snyk.sarif
```