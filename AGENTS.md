# Agents Contribution Guide

## HTML report template

`templates/tohtml/report.html` is the only template for the `to-html` command. After changing it, regenerate the example report and verify it before committing:

```bash
make example-report
```

Then commit the updated `templates/tohtml/example/example.html` alongside the template change.

### Verification checklist after regeneration

Open `templates/tohtml/example/example.html` via a local HTTP server (file:// is blocked by Playwright):

```bash
python3 -m http.server 9999 --directory templates/tohtml/example
# open http://localhost:9999/example.html
```

Clear the site's localStorage and sessionStorage before walking the checklist (DevTools → Application → Storage, or run `localStorage.clear(); sessionStorage.clear();` in the console): the report persists TOC state across visits in `localStorage`, and the active severities and scanner tab for the current tab in `sessionStorage`, so state left over from a previous pass makes an otherwise-correct report look broken.

Note the two are scoped differently, which matters when checking filter behaviour. TOC preferences are shared by every report. Severity and scanner selections are keyed by `data-report-state` on `<html>`, so they apply only to the report they were set on and only in that tab — see "Persisted state" in `templates/tohtml/README.md`. The search query is not persisted at all.

Confirm each of the following on `example.html` (single scanner):

- Header chips show `scan-io-git/scanio-d...`, branch `main`, a commit short-hash, and `Semgrep OSS 1.95.0`.
- Severity pills: Critical 2, High 3, Medium 9, Low 2, Info 1 (All 17 active, 5 suppressed).
- TOC "By severity" groups: Critical (db.py, settings.py), High (views.py ×2, zip_extractor_and_decompression_handler.py), Medium (nginx.conf ×3 with chip strip, views.py, tests/utils.py, Dockerfile, profile.js, app/utils.py, auth.py), Low (views.py, auth.py), Info (views.py).
- SQL injection (#1) shows "Data flow:" label with 3 numbered locations across db.py.
- OS command injection (#3) shows "Data flow:" with 2 locations; `os.system(cmd)` is column-highlighted.
- Path traversal (#5) shows a multi-line code block with first-line partial highlight.
- Hardcoded AWS key (#2) and insecure cookie (#16) show single-line column highlights.
- Findings with Suggested fix show the green fix block; findings without it omit it.
- Findings with References show the link list; findings without it omit it.
- Missing HTTP security header (nginx.conf) clusters as one TOC entry with chip strip `#6 #9 #12`; Dockerfile finding is separate.
- No chip strip in the example exceeds `CHIP_CAP` (48), so none shows a `+N more` button. To exercise the cap, render a SARIF with 50+ findings sharing one rule and file: the strip must show 48 chips plus `+N more`, and expanding it must scroll to the last chip.
- Suppressed section is collapsed with count 5. Expand it to confirm:
  - `Suppressed` (green banner, inSource) on #18 and #19.
  - `Suppression under review` (amber banner, external) on #20 and #21.
  - `Suppression rejected` (red banner, inSource) on #22.
- Widen the window past 1440px: the header brand, the `All` filter pill and the TOC drawer keep the same left edge, the cards sit ~16px right of the TOC with no void between them, and past 1920px the whole shell centers with symmetric margins.
- Search: type `injection` → title and Rule field highlight, TOC filters, Suppressed section hides when 0 match and updates count when some match.
- No scanner tab strip renders, and the TOC has no `By scanner` mode: `example.html`, `example-pr.html`, and `example-required.html` each carry exactly one scanner (`Semgrep OSS`).

### Multi-scanner and false-positive review checklist (`example-consolidated.html`)

`make example-report` also renders `example-consolidated.html` from `example.sarif` plus `example-ai-scan.sarif` (a second, distinct-scanner fixture) with `--required critical,high,medium,low`. Confirm:

- The header tool chip shows both driver name/version pairs: `Semgrep OSS 1.95.0, AI Security Scanner 1.4.0`.
- A scanner tab strip renders above the findings list: `All scanners` 22, `Semgrep OSS` 17, `AI Security Scanner` 5; the two scanner counts sum to the `All scanners` count.
- Clicking the `AI Security Scanner` tab narrows the card list to its 5 findings and recomputes both the severity pills (Critical 1, High 2, Medium 2, Low 0, Info 0) and the Required/Recommended pills (Required 3, Recommended 2) against just that tab. Clicking back to `All scanners` restores the full counts (Required 19, Recommended 3).
- At a narrow viewport (around 320px) the tab strip collapses `Semgrep OSS` and `AI Security Scanner` into a `+2 more` button; opening it and picking a scanner from the menu filters exactly as clicking its tab would.
- The `By scanner` TOC mode groups findings by scanner, then severity, then file, one level deeper than `By severity`.
- Typing in the search box narrows within the active scanner tab rather than resetting it back to `All scanners`.
- Finding numbering is continuous across both scanners, with no restart at the scanner boundary: #1-#22 active (interleaved by Required-then-severity, not by scanner), #23-#27 suppressed (all from `Semgrep OSS`; the AI Security Scanner fixture has no suppressions).
- The false-positive review panel renders for every result carrying a `properties.fp` verdict, with the exact labels `Confirmed as a real issue` (TRUE_POSITIVE), `Likely false positive` (FALSE_POSITIVE), and `Needs verification` (NEEDS_VERIFICATION).
- `Needs verification` carries a dotted underline and shows a tooltip on hover.
- `Mass assignment via unvalidated model binding` (Medium, AI Security Scanner) has no `properties.fp` bag at all: it shows no false-positive panel, and its Required banner reads "... not FP-assessed".
- `Privilege escalation via missing authorization check` (Critical, AI Security Scanner, `FALSE_POSITIVE`) reads `Recommended — Critical severity, false positive per FP review`: no severity is special-cased, so a false-positive verdict demotes a Critical finding exactly as it demotes any other. The slate panel below it keeps the same fixed color it has for every verdict.
- The confidence arrow (e.g. `High (88%) → Low (5%)`) renders only on findings that also carry `properties.pre_fp_confidence`; findings with only `properties.confidence` show a single value with no arrow and no fabricated "before" number. The arrow carries no strikethrough and no bold weight.
- Check both light and dark themes for the tab strip and the false-positive panel; the panel header background (`#e9eef5` light / `#1a2432` dark) is fixed regardless of verdict, independent of the Required/Recommended banner color above it.

### PR decisions, finding IDs and provenance links checklist (`example-pr-decisions.html`)

`make example-report` also renders `example-pr-decisions.html` from `example-pr-decisions.sarif` (scanner `Example SAST`) plus `example-pr-decisions-ai.sarif` (scanner `Example AI Reviewer`) with `--required critical,high,medium --never-demote critical --vcs github --pull-request 7`. The source folder is an empty temp folder without `.git`, so the repository, branch, commit and every link must come from the SARIF's `versionControlProvenance` (`https://github.com/scan-io-git/scanio-demo`); code snippets come from the SARIF. All names are synthetic (`alice`, `bob`). Confirm:

- Header: repo pill `scan-io-git / scanio-demo`, branch `feature/example`, commit `#0123456`, PR pill `#7`. The branch pill links to `.../tree/feature/example`, the commit pill to `.../commit/0123456789abcdef0123456789abcdef01234567` and the PR pill to `.../pull/7`. A finding's "Location in PR" link has the same shape as in `example-pr.html` (`.../pull/7/files#diff-<sha256 of path>R<line>`) and the "at commit" link is `.../blob/0123456789abcdef0123456789abcdef01234567/<path>#L<line>`. The `comment`, `request` and `approval` links in PR decision panels are `.../pull/7#issuecomment-<n>`.
- Pills: All 19, Critical 1, High 6, Medium 10, Low 2, Info 0, then Required 11, Recommended 8, Excluded 9 (the Excluded pill follows Recommended). All and the severity pills do not count the 9 excluded findings or the 2 suppressed ones.
- Scanner tabs: `All scanners` 19, `Example SAST` 16, `Example AI Reviewer` 3. The `Example AI Reviewer` tab shows All 3, Critical 0, High 2, Medium 1, Low 0, Info 0, Required 3, Recommended 0, Excluded 2.
- Section order: `Required` 11 (open), `Recommended` 8 (open), `Excluded in this PR` 9 (collapsed, `circle-off` icon, teal), `Suppressed` 2 (collapsed). Card headers carry no decision label of any kind.
- Finding IDs: cards show `F-xxxxxxxx` instead of `#n`; the duplicate shows `F-356a192b.81d0`; `Finding without a stable ID` shows `#8`. The ID sits on the same line as the severity badge and the first line of the title, also for titles that wrap to two lines, such as the `Order endpoint returns records ...` finding (check at a width around 700px). An ID longer than 15 characters (none in the example; temporarily edit `findingId` to try) is cut with an ellipsis and keeps its full value in a hover title.
- Notices, exact text:
  - Excluded card: `Excluded in this PR — false positive, does not block the merge` (or `accepted risk`).
  - `Missing authorization check`: `Required — confirmed real by a reviewer`.
  - `Predictable session identifier`: `Required — confirmed real by a reviewer, overrides the false-positive review`.
  - `Path traversal in upload handler` (pending on Required): `Required — High severity, confirmed by FP review. An exclusion is waiting for approval`.
  - `Weak random number generator` (pending on Recommended): `Recommended — Medium severity, false positive per FP review`, with nothing appended.
- PR decision panels are collapsed by default and open on click. Rows per kind:
  - `Open redirect` (approved request, Excluded colors): Excluded as `False positive`; Requested by `alice` + `request` link; Approved by `bob` + `approval` link; Reason; Date `18 Nov 2026`.
  - `Insecure temporary file` (single decision): Excluded as `False positive`; Decided by `alice` + `comment` link; Reason; Date `17 Nov 2026`.
  - `XML external entity expansion` (accepted risk): Excluded as `Accepted risk`; Decided by `bob` + `comment`; Ticket `TKT-4821`; Reason; Date.
  - `Excluded without links`: no link after the user name and no Date row.
  - Confirmed cards (Required colors, header `PR decision`): Marked as `Real issue`; Decided by `bob` + `comment`; Reason; Date. The false-positive review panel follows the PR decision panel.
  - Pending cards (amber, header `PR decision: exclusion requested`): Requested as; Requested by `alice` + `request`; Status `Waiting for approval; still blocks the merge` on the Required card and `Waiting for approval` on the Recommended one; Reason; Date.
- Findings panel, `By severity`: after Recommended comes `Excluded in this PR` 9 with Critical, High, Medium, Low groups, then file rows, then finding rows or an ID chip strip (`Debug flag enabled` 3 shows chips `F-0ade7c2c F-b1d57811 F-17ba0791`). `Suppressed` 2 follows with Medium and Low groups, then file, then rows (expand it in the panel; it starts collapsed).
- Findings panel, `By finding`: `Excluded in this PR` 9 then `Suppressed` 2, each listing rule rows, then files.
- Findings panel, `By scanner`: under `Excluded in this PR` the levels are scanner (`Example SAST` 7, `Example AI Reviewer` 2), severity, file, finding rows or chips, each one indent deeper. `Suppressed` has the same four levels.
- Clicking a finding in the panel that sits in a collapsed section (Excluded, Suppressed) opens the section and scrolls to the card.
- Filters: the `Excluded` pill shows only the 9 excluded cards and opens their section; Escape clears it. `Required` and `Recommended` hide excluded cards. Searching `F-356a` finds both the finding and its duplicate, and the matching part of the ID is highlighted.
- A 60-finding strip of IDs: render a SARIF with 60+ findings that share one rule and file and all carry IDs. The strip shows 20 chips plus `+N more` (not 48), the collapsed strip does not scroll, and expanding scrolls to the last chip.
- Colors, light and dark: Excluded section header, pill, notice and panel header use fg `#0f5c4f`, bg `#e6f6f2`, border `#1a7f6b` (dark `#56d4bc`, `#04241e`, `#1a7f6b`); the confirmed panel uses the Required colors; the pending panel uses the amber suppression-under-review colors (`#92400e`, `#fef3c7`, `#f59e0b`; dark `#e0a020`, `#1c1200`, `#9e6c00`).
- At 320px and 400px nothing forces horizontal page scroll: the summary bar pills scroll inside their own row, the card header ID, badge and title wrap without overlap, the PR decision panel keeps a readable label column, and the findings panel overlay fits the screen.
- Console: no errors, no Content-Security-Policy violations.

### Comparing regenerated output

A raw `diff` between two renders is never empty, for two reasons: the CSP nonce and the "Generated ..." timestamp are non-deterministic by design (`crypto/rand` per render, render clock), and every render embeds one shared `<style>`/`<script>` block, so a template change anywhere (even to a feature the current render never exercises) changes those bytes in every generated file. Strip both before comparing:

```bash
bodyonly() { python3 -c "
import re,sys
s=open(sys.argv[1]).read()
s=re.sub(r'<style[^>]*>.*?</style>','<STYLE/>',s,flags=re.S)
s=re.sub(r'<script[^>]*>.*?</script>','<SCRIPT/>',s,flags=re.S)
s=re.sub(r'nonce[-=\"]+[A-Za-z0-9_-]+','nonceX',s)
s=re.sub(r'Generated [^<]*','GeneratedX',s)
print(s)" "$1"; }
diff <(bodyonly before.html) <(bodyonly templates/tohtml/example/example.html)
```

An empty diff means the visible markup is byte-identical; anything left over is a real regression to investigate, not noise.

# Commit Messages and Pull Requests
- Follow convential commits instructions when write commit messages
- Every pull request should answer:
  - **What changed?**
  - **Why?**
  - **Breaking changes?**
