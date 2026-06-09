# Why `merge_base_sha` and `base_sha` Differ

When `--fetch-base` is set, Scanio returns two commit SHAs in the fetch extras. They look similar but mean different things, and using the wrong one as a semgrep baseline silently produces incorrect results.

## What each field represents

**`base_sha`** is the tip of the target branch at the moment the repository was cloned. It is whatever commit the VCS provider API reports as the target's `latestCommit` — in other words, where `main` (or whatever the target branch is) currently points.

**`merge_base_sha`** is the fork point: the most recent common ancestor between the PR head and the target branch tip. It is the commit where the PR's history diverged from the target.

## When they differ

Consider a PR that was created some time ago. While the PR was in review, other work landed on main:

```
main:   C0 ── C1 ── C2 ── C3    ← base_sha = C3
               \
feature:        A ── B           ← head (PR head = B)
```

Here:
- `base_sha` = C3 (where main is now)
- `merge_base_sha` = C1 (where the PR branched off)

These are different commits. C2 and C3 were written by other people after the PR was opened.

## Why the wrong SHA produces wrong results

Semgrep's `--baseline-commit` flag tells it: "only report findings on lines that changed since this commit." Internally it runs `git diff --merge-base <sha>` between the baseline and the current HEAD.

If you pass `base_sha` (C3) as the baseline:
- Semgrep computes `git diff C3 --merge-base B`
- The merge base of C3 and B is C1
- The diff includes C1→B (the PR's changes) **and also C1→C3** (C2 and C3, the changes from main)
- Semgrep suppresses findings on lines introduced by C2 and C3 — commits the PR author never wrote
- You miss real findings in the PR

If you pass `merge_base_sha` (C1) as the baseline:
- Semgrep computes `git diff C1 --merge-base B`
- The merge base of C1 and B is C1 (C1 is already the ancestor)
- The diff covers exactly the PR's own changes: A and B
- Findings are correct

## When they are the same

If the PR is fully up to date with the target branch (branched from the current tip), both SHAs are identical:

```
main:   C0 ── C1 ── C2 ── C3    ← base_sha = C3
                            \
feature:                     A ── B   ← head
```

Here `merge_base_sha` = C3 = `base_sha`. Either field gives the same result for semgrep, but using `merge_base_sha` consistently means your pipeline is always correct regardless of PR state.

## Why a shallow clone makes this harder

Scanio fetches with `--depth 1 --single-branch --no-tags` by default, which downloads only the latest snapshot without history. That is efficient — it avoids transferring the entire repo — but it also means neither C1 nor any other ancestor is in the local git store.

To compute `merge_base_sha`, Scanio uses the git CLI's `--shallow-exclude` protocol: it asks the server to send only the feature branch's commits up to the fork point, then deepens by one commit to include C1 itself. This fetches exactly `N_feature + 2` commits regardless of how far behind main the PR is — the server computes the boundary rather than the client walking the graph.

Without this, passing `base_sha` to semgrep would fail with `fatal: no merge base found` on a shallow clone, because C3 and B have no graph connection in the local store.

## Fallback behaviour

`merge_base_sha` is best-effort. If the git binary is absent, the base branch name is unavailable, or the deepen fetch fails for any reason, the field is omitted from the response. `base_sha` is always returned when the VCS API provides it. Pipelines should check whether `merge_base_sha` is present before using it and fall back to a full scan if it is not.
