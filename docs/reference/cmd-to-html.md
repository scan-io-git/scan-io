# To HTML Command
The `to-html` command converts sarif, standard sast output format, to a human-friendly html file.

## Table of Contents

- [Syntax](#syntax)
- [Options](#options)
- [Usage Examples](#usage-examples)
- [Report features](#report-features)
  - [Security](#security)
  - [Filtering](#filtering)
  - [Suppressed findings](#suppressed-findings)

## Syntax
```
scanio to-html --input/-i PATH --output/-o PATH [--source/-s PATH] [--templates-path/-t PATH] [--pull-request ID] [--no-supressions] [--no-csp]
```

### Options
| Option | Type | Required | Default Value | Description |
|--------|------|----------|---------------|-------------|
| `--input`, `-i` | string | Yes | `none` | Path to input file, sarif report |
| `--output`, `-o` | string | Yes | `none` | Path to output file, html report |
| `--source`, `-s` | string | No | `none` | Path to source code folder |
| `--templates-path`, `-t` | string | No | `none` | Path to templates folder |
| `--pull-request` | string | No | `none` | Pull request ID. Enables PR-aware links: the header pill links to the PR and each finding's "Location in PR" links to the PR diff at the exact line, with a secondary commit-permalink link. When omitted, auto-detected from CI env vars: GITHUB_REF (refs/pull/N/merge), CI_MERGE_REQUEST_IID, BITBUCKET_PR_ID |
| `--no-supressions` | bool | No | `false` | Enable removing results with suppressions properties |
| `--no-csp` | bool | No | `false` | Disable the Content-Security-Policy meta tag in the generated report |

## Usage Examples
The following examples demonstrate how to use the `to-html` command.

**Basic**  
Convert sarif output to html report, without code snippets.
```bash
scanio to-html -i /path/to/project/results.sarif -o /path/to/project/results.html
```

**With code snippets**  
Convert sarif output to html report with code snippets. Add a source code folder argument, so the tool can extract code snippets for corresponding code flows and locations in a report.
```bash
scanio to-html -i /path/to/project/results.sarif -o /path/to/project/results.html -s /path/to/project
```

**If no template path specified**  
If template path is not specified, the tool will look for templates in home folder: `SCANIO_HOME/templates/tohtml`. `SCANIO_HOME` can be configured in an AppConfig with `scanio.home_folder` key.

**Use custom template path**  
Use a custom path to a template file, in case it is located in non standard location or you would like to use customized verion of html template. The target folder should contain only a template with filename `report.html`.
```bash
scanio to-html -i /path/to/project/results.sarif -o /path/to/project/results.html -t /path/to/templates/tohtml
```

**Ignore Suppressed Findings**
The suppressions property in a SARIF result indicates that the finding was intentionally ignored, either in the source code or through external configuration. 
For example, Semgrep includes rules that were ignored using `// nosemgrep` in the SARIF results and marks them with a [suppressions property](https://docs.oasis-open.org/sarif/sarif/v2.0/csprd02/sarif-v2.0-csprd02.html#_Toc10127852). However, these are still listed as findings, which can be confusing compared to other output formats (e.g., JSON), where such suppressed issues are omitted entirely.

If you want to exclude suppressed results from the HTML report, use the `--no-supressions` flag.
```bash
scanio to-html -i /tmp/juice-shop/semgrep_results.sarif -o /tmp/juice-shop/semgrep_results.html -s /tmp/juice-shop/ -t ./templates/tohtml --no-supressions
```

**Disable CSP (for legacy viewers)**
By default, the report includes a strict Content-Security-Policy (see [Security](#security) below). Some email clients and legacy document viewers strip or reject `<meta>` CSP tags, which can prevent the report from rendering correctly. Use `--no-csp` to omit the policy for those environments.
```bash
scanio to-html -i /path/to/results.sarif -o /path/to/results.html --no-csp
```

**PR mode**
When `--pull-request` is set (or detected from CI env vars), the report renders in PR mode:
- The header shows a PR pill linking to the pull/merge request.
- Each finding card shows "Location in PR" linking to the PR diff at the exact line (GitHub: `#diff-<sha256>R<line>`, GitLab: `#<sha1>_<line>_<line>`, Bitbucket: `#<path>?t=<line>`).
- A secondary "at commit" link is shown beneath, preserving the exact-line commit permalink.
- Inline data-flow step links are unaffected (always commit links).

## Report features

The generated HTML file is fully self-contained and works offline.

### Security

Each report is hardened at render time:

- A 16-byte random nonce is generated with `crypto/rand` and embedded into every inline `<script>` and `<style>` tag.
- A `<meta http-equiv="Content-Security-Policy">` tag is injected as the first element in `<head>` with this policy:
  ```
  default-src 'none'; script-src 'nonce-{random}'; style-src 'nonce-{random}'; img-src data:; base-uri 'none'; form-action 'none'
  ```
- All external links use `target="_blank" rel="noopener noreferrer"` to prevent reverse tabnabbing.
- When `--source` is set, file reads for code snippets are confined to that directory. A SARIF with artifact URIs containing `../` traversal or absolute paths outside the folder will produce findings without code snippets rather than reading arbitrary files.

The CSP is defense-in-depth: Go's `html/template` already context-escapes all SARIF-derived values, but the nonce policy blocks injected `<script>` tags, inline event handlers, and `javascript:` URIs if escaping is ever bypassed. This matters because reports are often shared as email attachments or CI artifacts and opened by people other than the person who ran the scan.

Use `--no-csp` to omit the policy if the viewing environment does not support `<meta>` CSP tags.

### Filtering

- **Severity pills** -- click a severity label in the toolbar to show only findings of that level.
- **Free-text search** -- the search box filters findings by any combination of words. Matching text is highlighted in amber wherever it appears: title, file path, description, and metadata fields (Category, Confidence, Rule, Scanner). The findings panel (TOC) updates in sync.

The search index covers: title, description, file path, severity, rule ID, and all metadata field values. Typing `semgrep` finds all Semgrep findings; typing `low confidence` finds findings where both words appear anywhere in the finding.

Both filters combine with AND logic -- active severity pill plus a search term shows only findings that satisfy both.

### Suppressed findings

Suppressions are shown in a collapsed section at the bottom. If the active filter matches no suppressed findings, the section is hidden entirely.

