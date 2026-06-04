# Scanio Report Template

## Files

- `report.html` -- Go html/template. Rendered by `scanio to-html`.
- `design-kit.html` -- Static design reference. Open directly in a browser.

## Generating a report

```
scanio to-html \
  --input results.sarif \
  --output report.html \
  --source /path/to/source \
  --templates-path ./templates/tohtml
```

## Design tokens

All tokens live in the `:root` block in `report.html`. The design kit mirrors these tokens exactly. If you change a token in one, update the other.

| Group | Prefix | Range |
|-------|--------|-------|
| Brand | `--brand-green` | `#0f7a51` light / `#3ddc8e` dark |
| Severity | `--sev-{level}-{fg/bg}` | per severity, per theme |
| Type | `--text-xs` .. `--text-xl` | 11px .. 20px |
| Shadow | `--shadow-xs` .. `--shadow-lg` | |
| Z-index | `--z-sticky` .. `--z-dialog` | 20 .. 1000 |
| Spacing | `--space-1` .. `--space-6` | 4px .. 32px |
| Syntax | `--syntax-keyword` etc. | Prism token colors |

## Search

The search input filters findings by tokenised full-text match (all tokens must appear, case-insensitive). The index is built once on `DOMContentLoaded` from each finding's:

- `.finding__title`
- `.finding__description`
- `.finding__path`
- `data-severity` attribute
- `data-rule-id` attribute
- All `<dd>` values inside `.finding__meta-dl` (Category, Confidence, Rule, Scanner)

Matching tokens are highlighted in-place using `<mark class="search-mark">` elements. Highlights are cleared and reapplied on every filter change. The TOC rebuilds from the visible set on every change. The suppressed section is hidden when no suppressed findings are visible.

CSS: `mark.search-mark` -- amber `#fff3b0` (light) / `#5c4000` (dark).

## Constraints

- Single offline file. No CDN. No build step.
- Uses `[data-theme="dark"]` on `<html>` with a pre-paint boot script.
- Go template variables: `{{.Branch}}`, `{{.TotalFindings}}`, etc.
- `@media print` collapses the TOC and expands all findings.
