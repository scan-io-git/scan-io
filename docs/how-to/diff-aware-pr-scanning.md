# How to Run Diff-aware PR Scanning

This guide shows you how to configure Scanio to report only findings introduced by a pull request, rather than pre-existing issues in the codebase.

The approach works by fetching the PR base commit into the local git store and passing it to the scanner as a baseline. The scanner compares its results against that baseline and suppresses findings that already existed.

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

The fetch response extras will include `base_sha` — the commit SHA of the PR target branch tip. Your pipeline reads this and makes it available as `$BASE_SHA`.

> [!NOTE]
> `--fetch-base` is only valid when the URL contains a pull request ID. Using it without a PR URL returns an error.

## Run the scanner with `--baseline-commit`

Pass `$BASE_SHA` to the scanner via the `--` separator:

### OSS mode (local rules)

```bash
scanio analyse \
  --scanner semgrep \
  --format sarif \
  --config /scanio/rules/semgrep/ci/ \
  --output "$REPORTS_DIR/report.sarif" \
  "$CODE_DIR/" \
  -- --baseline-commit "$BASE_SHA" \
     --exclude '*vendor/*'
```

Semgrep scans the full repository at `$CODE_DIR` and reports only findings on lines changed since `$BASE_SHA`.

### Platform mode (Semgrep AppSec Platform)

If you use the [Semgrep AppSec Platform](https://semgrep.dev/products/semgrep-appsec-platform/) for some repositories, use `--command ci` instead. Rules are fetched from the platform; no `--config` is needed.

```bash
SEMGREP_APP_TOKEN=<token> \
scanio analyse \
  --scanner semgrep \
  --command ci \
  --format sarif \
  --output "$REPORTS_DIR/report.sarif" \
  "$CODE_DIR/" \
  -- --baseline-commit "$BASE_SHA"
```

## Combining with `--diff-files`

If other scanners in the same pipeline run need `$DIFF_FILES_DIR`, you can combine `--fetch-base` and `--diff-files` in a single fetch:

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

Both `diff_files_root` and `base_sha` will be present in the fetch response extras.

## How `base_sha` is surfaced

The fetch response JSON includes `base_sha` in the `result.extras` map:

```json
{
  "launches": [{
    "result": {
      "path": "/path/to/repo",
      "extras": {
        "repo_root": "/path/to/repo",
        "base_sha": "a1b2c3d4e5f6..."
      }
    }
  }]
}
```

If the base commit could not be resolved (e.g. the VCS API returned an empty SHA), the fetch fails with an explicit error rather than silently omitting the field.

## Reference

- [`scanio fetch` options](../reference/cmd-fetch.md#options)
- [`scanio analyse` options](../reference/cmd-analyse.md#options)
- [Semgrep plugin — CI mode](../reference/plugin-semgrep.md#ci-mode---command-ci)
