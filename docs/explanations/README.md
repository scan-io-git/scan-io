# Explanations

This section dives into the concepts and reasoning behind Scanio’s design and functionality. It explains how things work and why certain decisions were made. Explanations are for users who want to understand the "why" behind Scanio and gain deeper insights into its functionality.


## Articles

- [Why `merge_base_sha` and `base_sha` differ](diff-aware-baseline.md) — explains the fork-point concept, when the two SHA values diverge, and why the distinction matters for correct diff-aware scanning with semgrep.
- [Why the HTML report embeds a Content-Security-Policy](html-report-security.md) — explains the nonce-based CSP, why a `<meta>` tag is used instead of a response header, and when to disable it with `--no-csp`.
