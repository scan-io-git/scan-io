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
| Required | `--req-fg/bg/border` | amber palette, per theme |
| Recommended | `--rec-fg/bg/border` | green palette, per theme |
| Type | `--text-xs` .. `--text-2xl` | 11px .. 20px |
| Shadow | `--shadow-xs` .. `--shadow-lg` | |
| Z-index | `--z-sticky` .. `--z-dialog` | 20 .. 1000 |
| Spacing | `--space-1` .. `--space-6` | 4px .. 32px |
| Syntax | `--syntax-keyword` etc. | Prism token colors |
| Semantic | `--search-mark-bg`, `--on-accent` | per theme |

Type steps are distinct (no aliases): `xs` 11, `sm` 12, `base` 14, `lg` 16, `xl` 18 (finding title), `2xl` 20 (report title).

## Example report

`templates/tohtml/example/` contains a synthetic SARIF (`example.sarif`) and three rendered HTML reports:

- `example.html` — baseline report (no classification)
- `example-pr.html` — PR mode with `--pull-request 42`
- `example-required.html` — Required/Recommended mode with `--required "critical,high"`

The SARIF embeds `region.snippet.text` on every code location, so no source checkout is needed to render it.

The example covers: all five severity levels, all three suppression statuses (accepted / underReview / rejected), single-line column highlights, multi-line highlights, multi-step data flows, affected-code-only findings, findings with and without fix/references, and same-rule deduplication across multiple files (TOC chip clustering).

Regenerate after changing the template:

```
make example-report
```

Then commit the updated HTML files alongside the template change. See `AGENTS.md` for the full verification checklist.

## Required / Recommended classification

When `--required` is passed to `scanio to-html`, findings are classified as Required to fix or Recommended based on severity and confidence.

**Template data:** `Metadata.RequiredEnabled` (bool) gates all classification output. When `false` the report is byte-identical to the baseline. `Metadata.RequiredInfo` carries `"required"` and `"recommended"` counts.

**Per-finding data:** `Properties["Required"]` (`"true"`/`"false"`) and `Properties["RequiredReason"]` (human-readable rationale, e.g. `"High severity, confidence 85% >= 60% threshold"`). Set by `EnrichResultsRequiredProperty` in `internal/sarif/required.go`; absent when classification is off.

**DOM attributes:** each active `.finding` element carries `data-classification="required"` or `data-classification="recommended"` (empty string when off). The JS filter and TOC read from this attribute.

**UI surfaces:**
- `.findings-section` divs emitted on classification transition in the findings loop (Required group first, then Recommended). Hidden by `applyVisibility` when all their findings are filtered out.
- `.req-notice` banner inside each `.finding__body` (guarded by `RequiredEnabled`).
- `.pill--required` filter pill in the summary bar (template-gated). Toggling it sets `requiredOnly` and calls `applyVisibility`. Reset by the "All" pill, Escape, and the clear-filters button.
- TOC `buildSevTree` splits active items into Required/Recommended bands under `.tv-prio-hdr` banner headers when `requiredEnabled` is derived from the dataset.

**Design tokens:** `--req-fg/bg/border` (amber) and `--rec-fg/bg/border` (green), both themes.

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

CSS: `mark.search-mark` uses the `--search-mark-bg` token -- amber `#fff3b0` (light) / `#5c4000` (dark).

## Security

Each render injects a `<meta http-equiv="Content-Security-Policy">` tag as the first element in `<head>` with a strict nonce-based policy:

```
default-src 'none'; script-src 'nonce-{random}'; style-src 'nonce-{random}'; img-src data:; base-uri 'none'; form-action 'none'
```

A fresh 16-byte nonce (`crypto/rand`, base64url-encoded) is generated per render and placed on every inline `<script>` and `<style>` tag. No `'unsafe-inline'` or `'unsafe-eval'`. All external links carry `rel="noopener noreferrer"`.

When `--source` is set, disk reads for code snippets are confined to that directory via `os.OpenRoot`. Artifact URIs with `../` traversal or absolute paths outside the source folder produce findings without code snippets rather than reading arbitrary files.

The policy complements Go's `html/template` context-escaping — if a future escaping bypass were discovered, the nonce policy would still refuse injected scripts. Reports are typically shared as email attachments or CI artifacts, so recipients may open files crafted from a malicious SARIF.

Pass `--no-csp` to `scanio to-html` to omit the policy (e.g., for viewers that do not support `<meta>` CSP).

## Accessibility

- Heading hierarchy: the report title is level 1, each finding title is level 2 (`role="heading" aria-level`), and the References list inside a finding is level 3 (`<h3>`). Preserve this order when editing.
- Touch targets: small icon buttons (line copy, copy-all, data-flow step circles, TOC close, scroll-to-top) keep their compact visual size but expand to a >=44px hit area via a `::before` pseudo-element. The data-flow step circles only fill their row gap, so the hit area never overlaps a neighbour.
- The search input is pinned to 16px to stop iOS Safari auto-zoom on focus.
- Every interactive element shows a `--focus-ring` on `:focus-visible`; motion respects `prefers-reduced-motion`.

## Constraints

- Single offline file. No CDN. No build step.
- Uses `[data-theme="dark"]` on `<html>` with a pre-paint boot script.
- Go template variables: `{{.Branch}}`, `{{.TotalFindings}}`, etc.
- `@media print` collapses the TOC and expands all findings.
