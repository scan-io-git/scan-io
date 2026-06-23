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

## How `merge_base_sha` is computed

Scanio tries three approaches in order, using whichever succeeds:

**1. VCS provider API (preferred).** Each provider exposes an authoritative merge-base endpoint that the server computes from the full repository graph. Fork PRs and stale branches are handled correctly regardless of local history depth.

- Bitbucket Server: `fromHash` in the PR changes response (`/rest/api/1.0/.../pull-requests/{id}/changes`)
- GitHub: `merge_base_commit` from the Repositories compare API (`GET /repos/{owner}/{repo}/compare/{base}...{head}`)
- GitLab: `Repositories.MergeBase` API (`GET /api/v4/projects/{id}/repository/merge_base`)

**2. go-git materialization.** Once the correct SHA is known from the API, Scanio deepens the PR head's ancestry in the local shallow clone via go-git's `FetchContext` at increasing absolute depths (10 → 50 → 200 commits) until the merge-base commit is reachable from HEAD. This is required because change-aware scanners call `git diff --merge-base <sha>` in the cloned directory and need `git merge-base HEAD <sha>` to succeed locally. After each deepen, stale `.git/shallow` entries are pruned (go-git adds new shallow boundaries but does not remove obsolete ones) so the git CLI sees the correct ancestry. Reachability is confirmed with the git CLI (`git merge-base`), which honors `.git/shallow` exactly as the scanners do — not go-git's `IsAncestor`, which walks raw parent links and ignores `.git/shallow`. The distinction matters for merge-commit heads: when the head's first parent is the base but its second parent is absent, the head stays a parentless root in `.git/shallow` that the CLI cannot cross, so go-git would falsely report reachable while `git merge-base` fails.

**3. Git-binary fallback.** If go-git deepening fails (server does not support the protocol, auth unavailable), the git binary performs the same 10 → 50 → 200 deepen via explicit SHA fetches and verifies with `git merge-base`.

If the API is unavailable, the git binary both computes and materializes the merge-base using the target-branch tip as the reference point.

If all approaches fail, `merge_base_sha` is omitted. The value is either the true fork point C1 or absent — it is never an approximation such as the immediate parent of the PR tip.

## Fallback behaviour

`merge_base_sha` is best-effort. If the provider API is unavailable and the git fallback also fails (git binary absent, network error, ancestor not found within 200 commits), the field is omitted from the response. `base_sha` is always returned when the VCS API provides it. Pipelines should check whether `merge_base_sha` is present before using it and fall back to a full scan if it is not.

### No merge base (initial commit / unrelated histories)

Some PRs genuinely have no merge base — an initial commit, or a fork whose branch was created with an independent history so it shares no commit with the target. Bitbucket Server signals this by returning git's empty-tree hash (`4b825dc642cb6eb9a060e54bf8d69288fbee4904`) as the changes `fromHash`. Scanio recognises that sentinel, skips the deepening/fallback, and omits `merge_base_sha`. There is no valid baseline to give a change-aware scanner in this case (the empty tree is not a commit, and the unrelated base has no common ancestor), so the entire head is new — consumers must run a full scan by omitting `--baseline-commit`, which for an all-new head is equivalent to a diff-aware scan.
