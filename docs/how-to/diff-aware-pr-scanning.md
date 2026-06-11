# How to Run Diff-aware PR Scanning

This guide shows you how to configure Scanio to report only findings introduced by a pull request, rather than pre-existing issues in the codebase.

The approach works by fetching the PR's fork point into the local git store and passing it to the scanner as a baseline. The scanner compares its results against that baseline and suppresses findings that already existed.

## Prerequisites

- Scanio with the Semgrep plugin installed
- A pull request URL from GitHub, GitLab, or Bitbucket
- For CI mode only: `SEMGREP_APP_TOKEN` set in the environment

## Fetch the PR with `--fetch-base`

Add `--fetch-base` to your fetch command alongside the PR URL:

```bash
scanio fetch \
  --vcs bitbucket \
  --auth-type ssh-agent \
  --pr-mode branch \
  --fetch-base \
  --single-branch \
  --depth 1 \
  --no-tags \
  "https://bitbucket.example.com/projects/PROJ/repos/my-repo/pull-requests/42"
```

The fetch response extras will include two SHA values. Read both from the result JSON:

| Field | Value | Use for |
|-------|-------|---------|
| `merge_base_sha` | Fork point — where the PR diverged from the target | Pass to `--baseline-commit` |
| `base_sha` | Target-branch tip at fetch time | Drift detection only |

Your pipeline should extract `$MERGE_BASE_SHA` and pass it to the scanner step.

> [!IMPORTANT]
> Always pass `merge_base_sha` — not `base_sha` — to `--baseline-commit`. When the PR is behind the target branch, `base_sha` is a commit the PR author never touched, which causes semgrep to suppress findings it should report. See [Why `merge_base_sha` and `base_sha` differ](../explanations/diff-aware-baseline.md) for the full explanation.

> [!NOTE]
> `--fetch-base` is only valid when the URL contains a pull request ID. Using it without a PR URL returns an error. `merge_base_sha` is computed by querying the VCS provider API first (Bitbucket `fromHash`, GitHub compare endpoint, GitLab `merge_base` endpoint). This works for fork PRs and stale branches on all three providers. If the API call fails, a git-based fallback is tried. If both fail, the field is absent and the fetch still succeeds with `base_sha` only.

## Run the scanner with `--baseline-commit`

Pass `$MERGE_BASE_SHA` to the scanner via the `--` separator:

### OSS mode (local rules)

```bash
cd "$CODE_DIR" && \
scanio analyse \
  --scanner semgrep \
  --format sarif \
  --config /scanio/rules/semgrep/ci/ \
  --output "$REPORTS_DIR/report.sarif" \
  . \
  -- --baseline-commit "$MERGE_BASE_SHA" \
     --exclude '*vendor/*'
```

Semgrep scans the full repository and reports only findings on lines changed since the fork point.

> [!NOTE]
> Run the command from within the repo root (`cd "$CODE_DIR"`). Semgrep runs `git cat-file` from the current working directory — if that is not the cloned repository, it will fail to find the baseline commit even though it is present on disk.

### Platform mode (Semgrep AppSec Platform)

If you use the [Semgrep AppSec Platform](https://semgrep.dev/products/semgrep-appsec-platform/), use `--command ci` instead. Rules are fetched from the platform; no `--config` is needed.

```bash
cd "$CODE_DIR" && \
SEMGREP_APP_TOKEN=<token> \
scanio analyse \
  --scanner semgrep \
  --command ci \
  --format sarif \
  --output "$REPORTS_DIR/report.sarif" \
  . \
  -- --baseline-commit "$MERGE_BASE_SHA"
```

## If `merge_base_sha` is absent

`merge_base_sha` is best-effort. If it is missing from the fetch extras (the `git` binary was unavailable, the base branch name could not be resolved, or the deepen step failed), do not pass an empty value to `--baseline-commit` — that produces incorrect results. Either fail the pipeline or fall back to a full scan:

```bash
if [ -z "$MERGE_BASE_SHA" ]; then
  echo "merge_base_sha unavailable — running full scan"
  scanio analyse --scanner semgrep ...
else
  scanio analyse --scanner semgrep ... -- --baseline-commit "$MERGE_BASE_SHA"
fi
```

## Combining with `--diff-files`

If other scanners in the same pipeline need `$DIFF_FILES_DIR`, combine `--fetch-base` and `--diff-files` in a single fetch:

```bash
scanio fetch \
  --vcs bitbucket \
  --auth-type ssh-agent \
  --pr-mode branch \
  --fetch-base \
  --diff-files \
  --single-branch \
  --depth 1 \
  --no-tags \
  "https://bitbucket.example.com/projects/PROJ/repos/my-repo/pull-requests/42"
```

Both `diff_files_root` and `merge_base_sha` will be present in the fetch response extras.

## How the SHA values are surfaced

The fetch response JSON includes both fields in `result.extras`:

```json
{
  "launches": [{
    "result": {
      "path": "/path/to/repo",
      "extras": {
        "repo_root": "/path/to/repo",
        "base_sha": "c3d4e5f6...",
        "merge_base_sha": "a1b2c3d4..."
      }
    }
  }]
}
```

`base_sha` is always present when `--fetch-base` is set; if the VCS API returns an empty SHA the fetch fails explicitly. `merge_base_sha` is omitted without error when it cannot be determined — handle that case explicitly as shown above.

## Reference

- [`scanio fetch` options](../reference/cmd-fetch.md#options)
- [`scanio analyse` options](../reference/cmd-analyse.md#options)
- [Semgrep plugin — CI mode](../reference/plugin-semgrep.md#ci-mode---command-ci)
- [Why `merge_base_sha` and `base_sha` differ](../explanations/diff-aware-baseline.md)
