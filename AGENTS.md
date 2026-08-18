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

Confirm each of the following:

- Header chips show `scan-io-git/scanio-d...`, branch `main`, a commit short-hash, and `Semgrep OSS 1.95.0`.
- Severity pills: Critical 2, High 3, Medium 9, Low 2, Info 1 (All 17 active, 5 suppressed).
- TOC "By severity" groups: Critical (db.py, settings.py), High (views.py ×2, zip_extractor_and_decompression_handler.py), Medium (nginx.conf ×3 with chip strip, views.py, tests/utils.py, Dockerfile, profile.js, app/utils.py, auth.py), Low (views.py, auth.py), Info (views.py).
- SQL injection (#2) shows "Data flow:" label with 3 numbered locations across db.py.
- OS command injection (#3) shows "Data flow:" with 2 locations; `os.system(cmd)` is column-highlighted.
- Path traversal (#5) shows a multi-line code block with first-line partial highlight.
- Hardcoded AWS key (#1) and insecure cookie (#13) show single-line column highlights.
- Findings with Suggested fix show the green fix block; findings without it omit it.
- Findings with References show the link list; findings without it omit it.
- Missing HTTP security header (nginx.conf) clusters as one TOC entry with chip strip `#6 #9 #12`; Dockerfile finding is separate.
- No chip strip in the example exceeds `CHIP_CAP` (48), so none shows a `+N more` button. To exercise the cap, render a SARIF with 50+ findings sharing one rule and file: the strip must show 48 chips plus `+N more`, and expanding it must scroll to the last chip.
- Suppressed section is collapsed with count 5. Expand it to confirm:
  - `Suppressed` (green banner, inSource) on #16 and #17.
  - `Suppression under review` (amber banner, external) on #18 and #19.
  - `Suppression rejected` (red banner, inSource) on #20.
- Search: type `injection` → title and Rule field highlight, TOC filters, Suppressed section hides when 0 match and updates count when some match.
- Re-run `make example-report` a second time and diff: only the CSP nonce and the "Generated ..." timestamp may differ. Both are non-deterministic by design (`crypto/rand` per render, render clock), so the raw diff is never empty. Normalize before comparing:

  ```bash
  diff <(sed -E 's/nonce[-="]+[A-Za-z0-9_-]+/nonceX/g' before.html) \
       <(sed -E 's/nonce[-="]+[A-Za-z0-9_-]+/nonceX/g' templates/tohtml/example/example.html)
  ```

# Commit Messages and Pull Requests
- Follow convential commits instructions when write commit messages
- Every pull request should answer:
  - **What changed?**
  - **Why?**
  - **Breaking changes?**
