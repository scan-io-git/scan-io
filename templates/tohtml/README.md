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
| Type | `--text-xs` .. `--text-xl` | 12px .. 20px |
| Shadow | `--shadow-xs` .. `--shadow-lg` | |
| Z-index | `--z-sticky` .. `--z-dialog` | 20 .. 1000 |
| Spacing | `--space-1` .. `--space-6` | 4px .. 32px |
| Syntax | `--syntax-keyword` etc. | Prism token colors |

## Example report

`templates/tohtml/example/` contains a synthetic SARIF (`example.sarif`) and the rendered HTML (`example.html`). The SARIF embeds `region.snippet.text` on every code location, so no source checkout is needed to render it.

The example covers: all five severity levels, all three suppression statuses (accepted / underReview / rejected), single-line column highlights, multi-line highlights, multi-step data flows, affected-code-only findings, findings with and without fix/references, and same-rule deduplication across multiple files (TOC chip clustering).

Regenerate after changing the template:

```
make example-report
```

Then commit the updated `example.html` alongside the template change. See `AGENTS.md` for the full verification checklist.

## Suggested fix

The "Suggested fix" card appears inside each finding body when fix data is available. It is rendered as an Action Card: full brand-green border, neutral background (`--bg-subtle`), wrench icon header.

Fix content is parsed from markdown at report-generation time (Go side, `splitFixParts` in `internal/sarif/help_markdown.go`) into an ordered slice of prose and code parts. Prose renders as `<p class="finding__fix-prose">`. Fenced code blocks (e.g., ` ```python `) render as a `<pre class="finding__fix-pre"><code class="language-X">` block with a header row showing the language badge and a Copy button. Prism picks up the `language-X` class and syntax-highlights automatically.

**Source precedence** (highest first):
1. `result.properties.recommendation` — plain text, no fences expected
2. `rule.help.markdown` — the `## Fix` section, may contain fenced code blocks

The `fixes.artifactChanges` SARIF field (machine-executable patch format) is intentionally not rendered; it is not human-readable guidance.

**Copy button** uses the existing `data-copied` pattern: `fix-copy-btn[data-copied="1"]` turns green via `--success-fg`. The handler reads `code.innerText` from the nearest `.finding__fix-codeblock`.

## Search

The search input filters findings by tokenised full-text match (all tokens must appear, case-insensitive). The index is built once on `DOMContentLoaded` from each finding's:

- `.finding__title`
- `.finding__description`
- `.finding__path`
- `data-severity` attribute
- `data-rule-id` attribute
- All `<dd>` values inside `.finding__meta-dl` (Category, Confidence, Rule, Scanner)

Matching tokens are highlighted in-place using `<mark class="search-mark">` elements. Highlights are cleared and reapplied on every filter change. The TOC rebuilds from the visible set on every change. The suppressed section is hidden when no suppressed findings are visible.

When the active (non-suppressed) visible count reaches zero, `#no-results` (`.findings-empty`) is shown with a "Clear filters" button that resets both the severity filter and the search string. The toolbar `#search-count` span shows "N of M shown" whenever any filter is active.

CSS: `mark.search-mark` -- amber `#fff3b0` (light) / `#5c4000` (dark).

## Constraints

- Single offline file. No CDN. No build step.
- Uses `[data-theme="dark"]` on `<html>` with a pre-paint boot script.
- Go template variables: `{{.Branch}}`, `{{.TotalFindings}}`, etc.
- `@media print` collapses the TOC and expands all findings.
