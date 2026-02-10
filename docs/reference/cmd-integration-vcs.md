# Integration-VCS Command

The `integration-vcs` command lets Scanio talk directly to a VCS provider to read or update pull requests without cloning code. It is primarily used in CI pipelines after scanners have finished and a policy engine needs to review PR metadata or push statuses/comments back to the platform.

## Table of Contents
- [Supported Actions](#supported-actions)
- [Command Output Format](#command-output-format)
  - [Fields Returned by `checkPR`](#fields-returned-by-checkpr)
  - [Reviewer Verdicts](#reviewer-verdicts)
- [Examples](#examples)

## Supported Actions

| Action          | Description                                                                                       | Typical Result                                  |
|-----------------|---------------------------------------------------------------------------------------------------|-------------------------------------------------|
| `checkPR`       | Retrieves metadata about a pull request/merge request and validates that Scanio can access it.    | A `result` object describing the PR (see below) |
| `addComment`    | Posts a comment to the pull request and optionally uploads attachments.                           | Boolean success flag                            |
| `addRoleToPR`   | Adds an assignee or reviewer to the pull request.                                                 | Boolean success flag                            |
| `setStatusOfPR` | Submits an approval/unapproval review optionally guarded by a required head SHA.                  | Boolean success flag                            |

`integration-vcs` shares the same generic JSON envelope as the rest of Scanio commands:

```json
{
  "launches": [
    {
      "args": { ... },
      "result": { ... },
      "status": "OK",
      "message": ""
    }
  ]
}
```

## Command Output Format

When `--action checkPR` is used, `result` is populated with the fields from [`PRParams`](../../pkg/shared/ivcs.go). This section documents each field and the new `reviewers` array so that CI scripts can rely on them.

```json
{
  "launches": [
    {
      "args": {
        "repo_param": {
          "domain": "github.com",
          "namespace": "scan-io-git",
          "repository": "scan-io",
          "pull_request_id": "15"
        },
        "action": "checkPR"
      },
      "result": {
        "id": 15,
        "title": "Add semantic diff lines mode",
        "description": "Keeps new scanners happy.",
        "state": "OPEN",
        "author": {
          "user_name": "alice",
          "email": "alice@example.com"
        },
        "self_link": "https://github.com/scan-io-git/scan-io/pull/15",
        "source": {
          "id": "refs/heads/feature/demo",
          "display_id": "feature/demo",
          "latest_commit": "0123456789abcdef0123456789abcdef01234567"
        },
        "destination": {
          "id": "refs/heads/main",
          "display_id": "main",
          "latest_commit": "89abcdef0123456789abcdef0123456789abcdef"
        },
        "created_date": 1730732643,
        "updated_date": 1730800021,
        "reviewers": [
          {
            "reviewer": {
              "user_name": "bob",
              "email": "bob@example.com"
            },
            "verdict": "APPROVED",
            "lastReviewedCommit": "0123456789abcdef0123456789abcdef01234567"
          },
          {
            "reviewer": {
              "user_name": "carol",
              "email": "carol@example.com"
            },
            "verdict": "PENDING"
          }
        ]
      },
      "status": "OK",
      "message": ""
    }
  ]
}
```

### Fields Returned by `checkPR`

| Field          | Description                                                                                                                                         |
|----------------|-----------------------------------------------------------------------------------------------------------------------------------------------------|
| `id`           | Numeric pull request identifier reported by the VCS.                                                                                                |
| `title`        | Current pull request title.                                                                                                                         |
| `description`  | Pull request description/body text.                                                                                                                 |
| `state`        | Provider-specific state (e.g., `OPEN`, `MERGED`, `DECLINED`, `CLOSED`).                                                                              |
| `author`       | Object with `user_name` and `email` of the author.                                                                                                   |
| `self_link`    | Canonical URL to the pull/merge request.                                                                                                            |
| `source`       | Object describing the source reference (`id`, `display_id`, `latest_commit`). Bitbucket/GitHub provide ref names; GitLab uses branch names.         |
| `destination`  | Object describing the target reference (`id`, `display_id`, `latest_commit`).                                                                       |
| `created_date` | Unix epoch (seconds) when the pull request was created.                                                                                             |
| `updated_date` | Unix epoch (seconds) when the pull request was last updated.                                                                                        |
| `reviewers`    | Array of reviewer verdicts (see next section). Present only when the VCS plugin is able to collect reviewer data.                                   |

### Reviewer Verdicts

Each entry in `reviewers` has the following structure:

| Field                  | Description                                                                                                                            |
|------------------------|----------------------------------------------------------------------------------------------------------------------------------------|
| `reviewer.user_name`   | Reviewer display name/login normalized by the VCS plugin.                                                                              |
| `reviewer.email`       | Reviewer e-mail if exposed by the API; GitHub normally omits it for privacy.                                                           |
| `verdict`              | Uppercase status per reviewer. Typical values are `APPROVED`, `CHANGES_REQUESTED`, `REJECTED`, or `PENDING`.                            |
| `lastReviewedCommit`   | (Optional) SHA of the commit that the reviewer last reviewed. Available on GitHub/Bitbucket; GitLab currently leaves it empty.         |

Reviewer verdicts are aggregated differently per platform:

- **Bitbucket** exposes the most recent vote per reviewer, so `verdict` tracks approvals, rejections, or pending states.
- **GitHub** merges submitted reviews with the current “requested reviewer” list. Pending reviewers show `PENDING`; reviewers with completed reviews include `lastReviewedCommit`.
- **GitLab** merges approvers, reviewers, and approval states. All entries include the highest-priority verdict known to the API.

Use this field to decide whether automation should block, warn, or proceed before changing PR status. Downstream scripts can iterate through `reviewers` and require unanimous approval, ensure at least one approval, or audit that certain teams responded.

## Examples

Check whether GitHub PR #42 is accessible and dump all metadata (including reviewers) to stdout:

```bash
scanio integration-vcs --vcs github --action checkPR https://github.com/scan-io-git/scan-io/pull/42
```

Export GitLab merge request #17 metadata to a file for later policy checks:

```bash
scanio integration-vcs --vcs gitlab --action checkPR \
  --domain gitlab.com --namespace scan-io-git --repository scan-io --pull-request-id 17 \
  -o /tmp/mr17.json
```

In CI mode Scanio also emits the JSON artifact automatically under `{home_folder}/artifacts/integration-vcs-checkPR/<plugin>.json`, which makes it easy to share reviewer verdicts across jobs.
