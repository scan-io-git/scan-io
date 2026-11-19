# How to Add Finding IDs and PR Decisions to the HTML Report

This guide shows a tool author how to make `scanio to-html` show stable finding IDs and the decisions people made about findings in a pull request: exclusions, approved requests, confirmations and pending requests. You write the extra fields into the SARIF; to-html does the rest.

For every field, type and invalid-value rule, see [SARIF Input Read by to-html](../reference/to-html-sarif-input.md). For a complete working input, use [`example-pr-decisions.sarif`](../../templates/tohtml/example/example-pr-decisions.sarif).

## Prerequisites

- A SARIF file with one run, rules in `tool.driver.rules`, and a `ruleId`, `message.text` and `locations[0]` with a region on every result.
- The HTML template folder (`-t`), for example `./templates/tohtml` in the Scanio repository.

## Write a finding ID per result

Add `properties.findingId`. Use a value that stays the same for the same finding across runs; to-html shows it instead of `#n`:

```json
"properties": { "findingId": "F-3f9a2c1d" }
```

For a duplicate of another finding, use a suffix, for example `F-3f9a2c1d.81d0`. Results without an ID keep their `#n`.

## Write triage for decided findings

Add `properties.triage` only to results someone decided about. Each shape below produces a different card.

An applied exclusion, decided in one comment. The finding moves to the Excluded section:

```json
"triage": {
  "decision": "false_positive", "status": "applied",
  "author": "alice", "reason": "Only used by the local demo script.",
  "source": "https://github.com/scan-io-git/scanio-demo/pull/7#issuecomment-101",
  "decidedAt": "2026-11-17T04:10:16Z"
}
```

Use `"decision": "accepted_risk"` for an accepted risk, and set `"ticket"` to show a tracking reference on the panel.

An exclusion that was requested and then approved. Set `approver`, put the approval comment in `source` and the request comment in `requestSource`:

```json
"triage": {
  "decision": "false_positive", "status": "applied",
  "author": "alice", "approver": "bob", "reason": "The target is checked against an allow list.",
  "source": "https://github.com/scan-io-git/scanio-demo/pull/7#issuecomment-102",
  "requestSource": "https://github.com/scan-io-git/scanio-demo/pull/7#issuecomment-101",
  "decidedAt": "2026-11-18T09:00:00Z"
}
```

A confirmation, where a reviewer says the finding is real. With `--required` it becomes Required whatever its severity or review verdict:

```json
"triage": {
  "decision": "true_positive", "status": "applied",
  "author": "bob", "reason": "Reproduced locally.",
  "source": "https://github.com/scan-io-git/scanio-demo/pull/7#issuecomment-120"
}
```

A pending request. The finding keeps its class; the card shows a waiting panel. Here `source` is the request comment, and `requestSource` stays null until an approver exists:

```json
"triage": {
  "decision": "accepted_risk", "status": "pending",
  "author": "alice", "reason": "Not used for secrets.",
  "source": "https://github.com/scan-io-git/scanio-demo/pull/7#issuecomment-103"
}
```

If a `decision` or `status` is outside the values above, to-html treats the result as having no triage, and a `true_positive` with status `pending` is ignored.

## Add provenance when there is no git checkout

When the machine that renders the report has no clone of the repository, add the repository identity to the run so file and PR links still work:

```json
"versionControlProvenance": [{
  "repositoryUri": "https://github.com/scan-io-git/scanio-demo",
  "revisionId": "0123456789abcdef0123456789abcdef01234567",
  "branch": "feature/example"
}]
```

If the `--source` folder is a git checkout with an `origin` remote, git wins and this block is not used. Pass `--vcs` when the host name does not reveal the provider.

## Render the report

Render from an empty folder so links come from the provenance. The command below uses the example input from the Scanio repository:

```bash
scanio to-html \
  -i templates/tohtml/example/example-pr-decisions.sarif \
  -o report.html \
  -s "$(mktemp -d)" \
  -t ./templates/tohtml \
  --vcs github --pull-request 7 \
  --required critical,high,medium --never-demote critical
```

Replace `-i` with your own SARIF. Without `--required` there is no Required and Recommended split, but applied exclusions still produce the Excluded section.

## Check the result

Open `report.html` and look for:

- Finding IDs (`F-...`) in the card headers and the findings panel where you set `findingId`.
- An `Excluded` pill and an `Excluded in this PR` section, collapsed, with the count of applied exclusions. These findings are not in the All or severity counts.
- On each decided card, a collapsed PR decision panel. Open it and check the rows and that the `comment`, `request` and `approval` links open your pull request comments.
- On a Required card with a pending request, the notice ends with `An exclusion is waiting for approval`, and the panel is titled `PR decision: exclusion requested`.
- Header pills for repository, branch, commit and PR, and a file link on each card, pointing at your repository.

If a panel or link is missing, compare the field with the tables in [SARIF Input Read by to-html](../reference/to-html-sarif-input.md): a misspelled value, a non-http URL or an unparsable `decidedAt` drops that row or link without an error.
