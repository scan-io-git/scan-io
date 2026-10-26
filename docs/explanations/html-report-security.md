# Why the HTML Report Embeds a Content-Security-Policy

Scanio reports are self-contained HTML files typically shared as email attachments or CI artifacts. That means they are opened by people who did not generate them, from SARIF files they may not have audited. A malicious or malformed SARIF could embed JavaScript or data URIs in finding titles, descriptions, or file paths. Go's `html/template` context-escapes all SARIF-derived values, but defence in depth requires that even a future escaping bypass cannot execute injected code.

## The nonce-based policy

Each render generates a 16-byte random nonce with `crypto/rand`, base64url-encoded, and places it on every inline `<script>` and `<style>` tag. A `<meta http-equiv="Content-Security-Policy">` tag is injected as the first element in `<head>`:

```
default-src 'none';
script-src 'nonce-{random}';
style-src 'nonce-{random}';
img-src data:;
base-uri 'none';
form-action 'none'
```

This policy blocks:

- Inline event handlers (`onclick`, `onerror`, etc.) — no `'unsafe-inline'`
- `<script>` tags without the nonce — any injected script tag is refused
- `javascript:` URIs — covered by `default-src 'none'`
- External resource loads — no CDN calls, no exfiltration via `<img src>` or `<link>`
- `<base>` tag injection — `base-uri 'none'` prevents redirecting relative URLs
- Form submission — `form-action 'none'`

The nonce changes on every render. A script injected into the SARIF cannot know it ahead of time, so it cannot pass the policy even if template escaping were bypassed.

## Why `<meta>` CSP and not a response header

The report is a static file, not served by a web server. Browser headers are not available. The `<meta>` tag is the only delivery mechanism for offline files and email attachments. Its coverage is slightly narrower than header-based CSP (it cannot restrict `<frame>` ancestors, for example) but covers all the attack surfaces present in a static report.

## External links

All external links in the report carry `rel="noopener noreferrer"` to prevent reverse tabnabbing, where an opened tab could navigate the opener to a phishing page via `window.opener`.

## When to disable CSP

Some email clients and legacy document viewers strip or reject `<meta>` CSP tags, which can cause the report's inline scripts to be blocked or the page to render incorrectly. Pass `--no-csp` to omit the policy for those environments. The template escaping still applies; only the nonce-gating layer is removed.
