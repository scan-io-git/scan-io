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
- Severity pills: Critical 2, High 2, Medium 8, Low 2, Info 1.
- TOC "By severity" groups: Critical (settings.py, db.py), High (views.py ×2), Medium (views.py, nginx.conf ×3 with chip strip, Dockerfile, utils.py, auth.py), Low (auth.py, views.py), Info (views.py).
- SQL injection (#2) shows "Data flow:" label with 3 numbered locations across db.py.
- OS command injection (#3) shows "Data flow:" with 2 locations; `os.system(cmd)` is column-highlighted.
- Path traversal (#5) shows a multi-line code block with first-line partial highlight.
- Hardcoded AWS key (#1) and insecure cookie (#13) show single-line column highlights.
- Findings with Suggested fix show the green fix block; findings without it omit it.
- Findings with References show the link list; findings without it omit it.
- Missing HTTP security header (nginx.conf) clusters as one TOC entry with chip strip `#6 #7 #8`; Dockerfile finding is separate.
- Suppressed section is collapsed with count 5. Expand it to confirm:
  - `Suppressed` (green banner, inSource) on #16 and #17.
  - `Suppression under review` (amber banner, external) on #18 and #19.
  - `Suppression rejected` (red banner, inSource) on #20.
- Search: type `injection` → title and Rule field highlight, TOC filters, Suppressed section hides when 0 match and updates count when some match.
- Re-run `make example-report` a second time; `git diff templates/tohtml/example/example.html` should be empty (determinism check).

# Commit Messages and Pull Requests
- Follow convential commits instructions when write commit messages
- Every pull request should answer:
  - **What changed?**
  - **Why?**
  - **Breaking changes?**
