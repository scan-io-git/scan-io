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

Clear the site's localStorage and sessionStorage before walking the checklist (DevTools → Application → Storage, or run `localStorage.clear(); sessionStorage.clear();` in the console): the report persists filter, tab, and TOC state across visits, so a stale filter left over from a previous session makes an otherwise-correct report look broken.

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
- Clicking the `AI Security Scanner` tab narrows the card list to its 5 findings and recomputes both the severity pills (Critical 1, High 2, Medium 2, Low 0, Info 0) and the Required/Recommended pills (Required 4, Recommended 1) against just that tab. Clicking back to `All scanners` restores the full counts (Required 20, Recommended 2).
- At a narrow viewport (around 320px) the tab strip collapses `Semgrep OSS` and `AI Security Scanner` into a `+2 more` button; opening it and picking a scanner from the menu filters exactly as clicking its tab would.
- The `By scanner` TOC mode groups findings by scanner, then severity, then file, one level deeper than `By severity`.
- Typing in the search box narrows within the active scanner tab rather than resetting it back to `All scanners`.
- Finding numbering is continuous across both scanners, with no restart at the scanner boundary: #1-#22 active (interleaved by Required-then-severity, not by scanner), #23-#27 suppressed (all from `Semgrep OSS`; the AI Security Scanner fixture has no suppressions).
- The false-positive review panel renders for every result carrying a `properties.fp` verdict, with the exact labels `Confirmed as a real issue` (TRUE_POSITIVE), `Likely false positive` (FALSE_POSITIVE), and `Needs verification` (NEEDS_VERIFICATION).
- `Needs verification` carries a dotted underline and shows a tooltip on hover.
- `Mass assignment via unvalidated model binding` (Medium, AI Security Scanner) has no `properties.fp` bag at all: it shows no false-positive panel, and its Required banner reads "... not FP-assessed".
- `Privilege escalation via missing authorization check` (Critical, AI Security Scanner, `FALSE_POSITIVE`) shows a red `Required` banner directly above the neutral slate false-positive panel; the two never borrow each other's color, so a Critical finding the agent thinks is bogus never renders behind a "safe"-colored panel.
- The confidence arrow (e.g. `High (88%) → Low (5%)`) renders only on findings that also carry `properties.pre_fp_confidence`; findings with only `properties.confidence` show a single value with no arrow and no fabricated "before" number. The arrow carries no strikethrough and no bold weight.
- Check both light and dark themes for the tab strip and the false-positive panel; the panel header background (`#e9eef5` light / `#1a2432` dark) is fixed regardless of verdict, independent of the Required/Recommended banner color above it.

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
