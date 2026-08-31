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
| Shell | `--shell-max` | 1920px page-shell cap |
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

## Page shell

Four bands share one horizontal axis: `.report-header__inner`, `.summary-bar__inner`, `.outline-layout` (the TOC + findings row) and `.report-footer__inner`. Each is capped at `--shell-max` (1920px), centered with `margin: 0 auto`, and — importantly — set to `box-sizing: border-box` so their 16px padding counts *inside* the cap. Without `border-box` the padding sits outside it and the bands land 16px apart.

`.main-body` deliberately has no cap of its own: it fills the column left over beside the TOC. Capping it there was the original defect — it centered the findings inside `.main-container`, which starts after the sidebar, so the cards were centered on a different axis than the header. At 2560px that left a 453px void between the TOC and the cards and another 453px to their right, and the header aligned with neither. The drift began at 1440px, not just on ultrawide displays.

Invariant to preserve when editing any of these rules: above 900px the header brand, the first filter pill and the TOC drawer must all share the same left edge, at every viewport width.

Findings below 900px switch to the overlay TOC and are unaffected by the cap.

## TOC chip strips

When two or more findings share a rule and a file, the TOC collapses them into one row plus a strip of `#N` chips. Both trees render the strip through `renderChipStrip` in `report.html`, marker-wrapped so `cmd/to-html/toc_chips_test.go` runs the shipped code under node.

The strip renders at most `CHIP_CAP` (48) chips. The rest are emitted with `tv-chip--overflow` (hidden by CSS) behind a `+N more` button that adds `is-expanded` to the strip, revealing them in a 300px scroll pane. Expansion state resets whenever a filter or search rebuilds the trees.

The cap exists because the strip has a bounded `max-height` to drive the collapse animation: a rule with hundreds of findings in one file (a secret scanner over a lockfile, say) rendered a strip several thousand pixels tall, of which everything past ~240px was clipped, invisible, and unclickable. Two rules keep that from recurring:

- `max-height: 480px` on the strip clears a full 48-chip strip in the worst case measured (240px drawer, 4-digit finding numbers, ~410px).
- `overflow-y: auto` on any strip that is not collapsed, as a backstop: if a strip ever exceeds the bound anyway, it scrolls instead of stranding chips.

Raising `CHIP_CAP` means re-checking both, since a taller strip has to keep fitting under `max-height`.

## Scanner tab strip

Rendered when the report merges more than one input (`Metadata.Scanners` has 2+ entries). `initScannerTabs` in `report.html` owns it. `activeScanner` (`''` = all) scopes `applyVisibility`, and every count on screen is recomputed against it: the tab's own `.scanner-tab__n`, the severity pills, and the Required/Recommended pills. A number next to a filter always describes what the active tab shows, never the whole report.

Overflow is measured, not guessed. `measureOverflow` reflows every tab back into the list, then walks them summing `offsetWidth + GAP` against `container.clientWidth`; the first tab that would not fit becomes the cut, and the remainder move into the `+N more` menu. It re-runs on resize. Two invariants in that function:

- `if (cut < 1) cut = 1` — "All scanners" always stays visible, even in a container too narrow for it. Losing the way back to the unfiltered view would be worse than one clipped tab.
- Measurement requires the tabs to be in the DOM and unhidden, which is why it appends them all and un-hides `+N more` before reading widths. Measuring a hidden or clamped element reports the wrong size — the same class of bug as measuring a collapsed element's `scrollHeight`.

Driver names come from the SARIF, so the strip cannot assume a fixed set or short names. It was checked at 12 scanners. Deliberately rejected: per-scanner colours and initials. `AI Security Scanner` is 19 characters and word-initials give "ASS", and neither scales to arbitrary names, so the row carries no scanner marker at all — the tab is the only affordance.

## False-positive review panel

`.finding__fp`, emitted inside `.finding__body` when a result carries `properties.fp` with a recognized verdict. Structure is a titled panel over a `.finding__fp-dl` key/value list: verdict, then reasoning. Evidence is deliberately not rendered.

The panel's colour is independent of the verdict — a fixed slate (`--fp-border`, `--fp-bg`, `--fp-fg`), not the Required/Recommended palette. That was a decision, not an oversight: tinting the panel by outcome makes the panel itself look like a verdict badge and collides with the `.req-notice` banner directly above it, which is already colour-coded. Slate was picked against the Low-severity grey so the two do not read as the same signal.

Verdict wording is owned by `fpVerdictLabels` in `internal/sarif/fp.go`, not the template: `Confirmed as a real issue`, `Likely false positive`, `Needs verification`. `NEEDS_VERIFICATION` also carries a hover hint (`fpVerdictHover`) rather than printing its full explanation inline, which otherwise dominated the panel.

Confidence is not shown here. It stays in the finding's meta list, rendered as a pre-FP to post-FP arrow, because the FP agent overwrites `properties.confidence` with `p_real` and the reader needs both numbers to make sense of the change.

## Required / Recommended classification

When `--required` is passed to `scanio to-html`, findings are classified as Required to fix or Recommended.

**Basic usage** — list blocker severities; all findings of those severities become Required regardless of confidence score:

```
scanio to-html --required "critical,high,medium"
```

**With confidence filtering** — append `:threshold` (0-1) to a severity to demote findings whose confidence falls below it:

```
scanio to-html --required "critical,high:0.60,medium:0.70"
```

Severities listed without a threshold (`critical` above) are always Required. Confidence filtering is only applied for severities that have an explicit `sev:N` threshold in the flag or a `SCANIO_CONFIDENCE_THRESHOLD_<SEV>` env var.

**Pinning a severity** — `--never-demote critical` keeps a severity Required whatever the FP verdict or threshold says:

```
scanio to-html --required "critical,high" --never-demote "critical"
```

It is checked before both, so a pinned severity never reaches the verdict or threshold branch. It does not promote: a severity absent from `--required` stays Recommended. `RequiredReason` then reads `"Critical severity, always required"`, or `"Critical severity, always required despite false-positive review"` when the verdict disagreed — the FP panel still renders that verdict, so the banner has to acknowledge it or the card reads as self-contradictory.

**Template data:** `Metadata.RequiredEnabled` (bool) gates all classification output. When `false` the report is byte-identical to the baseline. `Metadata.RequiredInfo` carries `"required"` and `"recommended"` counts.

**Per-finding data:** `Properties["Required"]` (`"true"`/`"false"`) and `Properties["RequiredReason"]` (human-readable rationale, e.g. `"High severity, confidence 85% >= 60% threshold"` or `"High severity (blocker, no confidence threshold configured)"`). Set by `EnrichResultsRequiredProperty` in `internal/sarif/required.go`; absent when classification is off.

**DOM attributes:** each active `.finding` element carries `data-classification="required"` or `data-classification="recommended"` (empty string when off). The JS filter and TOC read from this attribute.

**UI surfaces:**
- `.findings-section` divs emitted on classification transition in the findings loop (Required group first, then Recommended). Hidden by `applyVisibility` when all their findings are filtered out.
- `.req-notice` banner inside each `.finding__body` (guarded by `RequiredEnabled`).
- `.pill--required` filter pill in the summary bar (template-gated). Toggling it sets `requiredOnly` and calls `applyVisibility`. Reset by the "All" pill, Escape, and the clear-filters button.
- TOC `buildSevTree` splits active items into Required/Recommended bands under `.tv-prio-hdr` banner headers when `requiredEnabled` is derived from the dataset.

**Design tokens:** `--req-fg/bg/border` (amber) and `--rec-fg/bg/border` (green), both themes.

## References

`.finding__refs` renders `Properties["References"]` as a plain `<ul>` of full URLs. Past the third item, entries carry `.finding__refs-more` (hidden by CSS) and a `.finding__refs-toggle` button reveals them by adding `is-expanded` to the section. The button is emitted only when there are more than three, so the common case — one reference — has no control at all.

This replaced a `max-height: 160px; overflow-y: auto` scroll box. That box turned the list into a nested scroll container, so the wheel drove the list instead of the page whenever the pointer was over it, and it only ever engaged on the rare finding carrying many links: across the 22 findings in the consolidated example exactly one overflowed, by 19px. The toggle is bounded the same way but without capturing scroll — 8 references render at 118px instead of 184px, and that height no longer grows with the reference count.

Two rules to preserve when editing:

- The print block must reveal every reference (`.finding__refs-more { display: list-item !important; }`) and hide the toggle. Printing is the one context where a control cannot be operated, so hidden content would be lost outright.
- Keep the hidden items in the DOM rather than dropping them. Browser find-in-page and the report's own search both read `textContent`, which includes `display: none` items, so a search hit still resolves inside a collapsed list.

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
