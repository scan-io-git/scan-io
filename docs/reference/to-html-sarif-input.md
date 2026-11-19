# SARIF Input Read by to-html

This page is the input contract for any tool that writes SARIF for [`scanio to-html`](cmd-to-html.md). It lists every field the command reads, what the field does in the report, and what happens when the value is missing or invalid. Standard SARIF fields keep their SARIF meaning. The properties in [Producer contract](#producer-contract) (`findingId`, `triage`, `fp`, `confidence`, `pre_fp_confidence`) are not part of SARIF: they are a contract between to-html and the tools that feed it. For a walkthrough, see [How to Add Finding IDs and PR Decisions to the HTML Report](../how-to/add-pr-decisions-to-html-report.md).

## Table of Contents

- [Scope](#scope)
- [Run level](#run-level)
- [Rule level](#rule-level)
- [Result level, standard SARIF](#result-level-standard-sarif)
- [Producer contract](#producer-contract)
  - [properties.findingId](#propertiesfindingid)
  - [properties.triage](#propertiestriage)
  - [properties.fp](#propertiesfp)
  - [properties.confidence and properties.pre_fp_confidence](#propertiesconfidence-and-propertiespre_fp_confidence)
- [Classification precedence](#classification-precedence)
- [Annotated example](#annotated-example)

## Scope

- Only `runs[0]` of each input file is rendered: its results, rules and tool. Further runs are ignored. An input with no `runs`, or a first run without `tool` or `results`, renders a valid report with no findings. `null` entries in `results` and `rules` are skipped. The exception is `versionControlProvenance`, which is searched in every run of every input.
- Several `--input` files are merged into one report; each file contributes its first run.
- The output file is written only after the whole report rendered. A failed render creates no file and leaves an existing one unchanged.
- Property keys that start with an uppercase letter (`Severity`, `Title`, `Required`, `Excluded` and similar) are set by to-html while it renders. Do not write them.

## Run level

| Path | Type | Optional | Allowed values | Effect | Missing or invalid |
|------|------|----------|----------------|--------|--------------------|
| `runs[].tool.driver.name` | string | yes | any | Scanner name in the header, the scanner tabs, the `Scanner` metadata row and the By scanner grouping. The names `trufflehog3`, `trufflehog`, `gitleaks` and `betterleaks` (case-insensitive) also set the category of every finding to `Hardcoded secret`. | Empty name: no scanner label, no tab. |
| `runs[].tool.driver.semanticVersion` | string | yes | any | Shown after the name in the header. `version` is not read. | Header shows the name only. |
| `runs[].tool.driver.rules[]` | array | yes | see [Rule level](#rule-level) | Titles, descriptions, severity, references and categories. | See the rule table. |
| `runs[].versionControlProvenance[].repositoryUri` | string | yes | HTTPS web URL of the repository, for example `https://github.com/scan-io-git/scanio-demo` | Repository pill and every link (file permalinks, commit, branch, PR). Parsed with the same provider detection as a git remote: `--vcs` and the configured host map apply. | The entry is skipped when blank. A URL that does not parse is ignored with a warning and git metadata stays as it was. |
| `runs[].versionControlProvenance[].revisionId` | string | yes | commit hash | Commit pill and file permalinks (`.../blob/<revisionId>/<path>#L<line>` on GitHub). | No commit pill and no file permalinks. PR diff links do not need it. |
| `runs[].versionControlProvenance[].branch` | string | yes | branch name | Branch pill and its link. | No branch pill. |

Provenance is used only when git supplies no repository URL: the `--source` folder has no `.git`, the repository has no `origin` remote, or `--source` is not set. Then the first entry with a non-blank `repositoryUri` supplies repository, commit and branch together; nothing is mixed with local git values. When git has a repository URL, provenance is ignored.

## Rule level

A rule is matched to a result by `rules[].id` equal to `results[].ruleId`.

| Path (under `runs[0].tool.driver.rules[]`) | Type | Optional | Allowed values | Effect | Missing or invalid |
|------|------|----------|----------------|--------|--------------------|
| `id` | string | yes | the `ruleId` of its results | Links the rule to its results; shown as the `Rule` metadata row. | A result whose rule is not listed renders as a result without a rule: the title comes from `message.text` and there are no rule-derived references, fix or severity. |
| `shortDescription.text` | string | yes | any | First choice for the finding title. | Next candidate. |
| `name` | string | yes | any | Second choice for the title. | Next candidate. |
| `fullDescription.text` | string | yes | any | Description when the result has no `message.text`. | No description. |
| `helpUri` | string | yes | URL starting with `http` | The single reference link when the result has no `properties.references`. | Next source. |
| `help.markdown` | string | yes | markdown | A `References` section (heading `References`, `**References**` or `<b>References:</b>`) supplies reference links; a `Fix` section supplies the suggested fix. | No links or fix from this source. |
| `defaultConfiguration.level` | string | yes | see [severity](#severity) | Last fallback for severity. | Severity `unknown`. |
| `properties.tags` | array of strings | yes | `CWE-<n>: ...` or `external/cwe/cwe-<n>`; `HIGH CONFIDENCE`, `MEDIUM CONFIDENCE`, `LOW CONFIDENCE` | The first recognized CWE number maps to the `Category` row. The confidence tags give a confidence of 85%, 65% and 40% when the result has no `properties.confidence`. | Category falls back to keywords in the rule ID, then `Other`. |
| `properties.precision` | string | yes | `very-high`, `high`, `medium`, `low` | Confidence of 95%, 85%, 65%, 40% when nothing more specific exists. | Ignored. |
| `properties["security-severity"]` | number or numeric string | yes | 0.0 to 10.0 | Severity bucket, first choice. | Next source. |
| `properties["problem.severity"]` | string | yes | see [severity](#severity) | Severity bucket, second choice. | Next source. |

The title is the first non-empty candidate of: `shortDescription.text`, `name`, the first sentence of `message.text`, the first line of `message.text`. A candidate that contains the rule ID is replaced by the humanized rule ID (last dot segment, dashes and underscores turned into spaces). A result with no `ruleId`, or whose rule is not in `tool.driver.rules`, has no `shortDescription` and no `name`, so its title is the first sentence or line of `message.text`. When every candidate is empty the title is the humanized rule ID, or `Finding`. Titles are cut at 120 characters.

### Severity

The severity bucket (`critical`, `high`, `medium`, `low`, `info`, or `unknown`) is taken from the first of these that resolves:

1. `rule.properties["security-severity"]`: 9.0 and above is `critical`, 7.0 to 8.9 `high`, 4.0 to 6.9 `medium`, above 0 `low`, 0 `info`.
2. `rule.properties["problem.severity"]`.
3. `result.level`.
4. `rule.defaultConfiguration.level`.

String values are matched case-insensitively: `critical`; `high` or `error`; `medium` or `warning`; `low` or `note`; `info` or `none`. An unrecognized string moves on to the next source.

## Result level, standard SARIF

| Path (under `runs[0].results[]`) | Type | Optional | Allowed values | Effect | Missing or invalid |
|------|------|----------|----------------|--------|--------------------|
| `ruleId` | string | yes | an `id` from `tool.driver.rules` | Links to the rule. | The finding renders without a rule: no `Rule` row, and the title falls back to `message.text` as described under [Rule level](#rule-level). |
| `level` | string | yes | see [severity](#severity) | Severity when the rule gives none. | Next source. |
| `message.text` | string | yes | any | Finding description; also a title candidate. | Falls back to the rule's `fullDescription.text`; without a rule the description is empty. |
| `locations[0].physicalLocation.artifactLocation.uri` | string | yes | relative path, or absolute path under `--source` | File shown in the card header and the Location link. Absolute paths have the `--source` prefix removed. | The location is left out: no file in the card header, no `Location` line, no permalink and no code block. |
| `locations[0].physicalLocation.region` | object | yes | `startLine`, `endLine`, `startColumn`, `endColumn` (1-based), `snippet.text` | Line number, column highlight and the code block. A missing `endLine` equals `startLine`; missing columns highlight the whole line. | The location is left out, as for a missing `uri`. A region without `startLine` keeps the file but drops the line: the header shows the path only, the Location line and the permalink or PR diff link point to the file without a line anchor, and a code block from `snippet.text` has no line numbers. Source lines cannot be read from `--source` without `startLine`. |
| `locations` | array | yes | at least one | The first location is the card's primary location. | Missing or empty: the finding renders without a location, as for a missing `uri`. |
| `region.snippet.text` | string | yes | source text | Code shown as is, including its line count. | The lines are read from `--source`, confined to that folder. If the file cannot be read the card has no code block. |
| `codeFlows[].threadFlows[].locations[].location` | object | yes | same shape as `locations[]` entries | Data flow steps. With more than one step in the first flow the section is labeled `Data flow`, otherwise `Affected code`. Duplicate thread flows are removed. | Without code flows each location becomes its own flow. A step without `uri` or `region` is dropped. A result left with no steps has no code block. |
| `suppressions[0].kind` | string | yes | `inSource`, `external` | Shown on the suppression banner. | Empty. |
| `suppressions[0].status` | string | yes | `accepted`, `underReview`, `rejected` | Banner `Suppressed`, `Suppression under review`, `Suppression rejected`. Any other value shows `Suppressed`. | Treated as accepted. |
| `suppressions[0].justification` | string | yes | any | Shown on the banner. | Omitted. |
| `properties.references` | array of strings | yes | URLs starting with `http` | Up to 10 reference links; takes priority over `helpUri` and `help.markdown`. | Next source. |
| `properties.recommendation` | string | yes | text, may contain fenced code blocks | Suggested fix block. | Falls back to the `Fix` section of `help.markdown`. |

A result with any entry in `suppressions` goes to the Suppressed section and is never classified. `--no-supressions` removes such results before rendering.

## Producer contract

### properties.findingId

| Path | Type | Optional | Allowed values | Effect | Missing or invalid |
|------|------|----------|----------------|--------|--------------------|
| `results[].properties.findingId` | string | yes | any string that is not empty after trimming, for example `F-3f9a2c1d` or `F-3f9a2c1d.81d0` for a duplicate | Replaces the positional `#n` in the card header, the findings panel rows and the chips. Searchable by any part of it. | Empty, blank or non-string values are ignored and the finding keeps `#n`. |

The ID is shown as given and HTML-escaped. It is not validated. In the card header and the findings panel rows it is cut with an ellipsis beyond 16 characters (`F-3f9a2c1d.81d0`, 15 characters, fits); a chip is cut at the width of its strip. The full value is in the hover title. A chip strip that contains IDs shows at most 20 chips before `+N more`; strips of `#n` chips show 48.

### properties.triage

Written by a tool that records decisions people made about findings in a pull request. Present only on results someone decided about.

| Field | Type | Nullable | Values | Meaning |
|-------|------|----------|--------|---------|
| `decision` | string | no | `false_positive`, `accepted_risk`, `true_positive` | What was decided. Exact, case-sensitive match. |
| `status` | string | no | `applied`, `pending` | `applied`: the decision is in force. `pending`: an exclusion was requested and awaits approval. |
| `author` | string | yes | any | Who decided, or who requested. Shown as plain text. |
| `reason` | string | yes | any | Free text, shown in quotes. It does not repeat `ticket`. |
| `approver` | string | yes | any | Who approved a request. When set on an applied exclusion, the panel shows Requested by and Approved by. |
| `ticket` | string | yes | any, printed as given | Tracking reference. Shown on Excluded panels only. |
| `source` | string | yes | absolute http(s) URL | The deciding comment. For a pending request it is the request comment. |
| `requestSource` | string | yes | absolute http(s) URL | The request comment. Set only once an approver exists. |
| `decidedAt` | string | yes | RFC 3339 timestamp | Shown as `17 Nov 2026`, converted to UTC. |

Rules:

- An unknown `decision` or `status`, a `triage` that is not an object, or a `true_positive` with status `pending` is treated as no triage.
- Triage on a result that has `suppressions` is ignored.
- Only absolute `http` and `https` URLs become links. Any other value is dropped and the link is omitted.
- For a pending request the report links the request from `requestSource` and falls back to `source` when `requestSource` is missing or not a usable URL.
- A `decidedAt` that does not parse as RFC 3339 produces no Date row.
- `author` and `approver` are not validated; an empty one renders an empty value.

| decision, status | Class | Card |
|------------------|-------|------|
| `false_positive` or `accepted_risk`, `applied` | Excluded | Notice `Excluded in this PR`, collapsed panel with rows Excluded as, Decided by (or Requested by and Approved by), Ticket, Reason, Date |
| `true_positive`, `applied` | Required with `--required`, otherwise unclassified | Notice `confirmed real by a reviewer`, collapsed panel with rows Marked as, Decided by, Reason, Date |
| `false_positive` or `accepted_risk`, `pending` | unchanged | The usual notice, with `An exclusion is waiting for approval` appended on a Required card; collapsed panel with rows Requested as, Requested by, Status, Reason, Date |

### properties.fp

False-positive review result, written by a tool that reviews findings.

| Path | Type | Optional | Allowed values | Effect | Missing or invalid |
|------|------|----------|----------------|--------|--------------------|
| `results[].properties.fp.verdict` | string | yes | `TRUE_POSITIVE`, `FALSE_POSITIVE`, `NEEDS_VERIFICATION` | Shows the False-positive review panel (`Confirmed as a real issue`, `Likely false positive`, `Needs verification`) and, with `--required`, decides the class. | Any other value, or no `fp` object, means not reviewed: no panel, and classification falls back to confidence. |
| `results[].properties.fp.reasoning` | string | yes | any | Reasoning row of the panel. | Row omitted when blank. |

Other keys in `fp` (for example `evidence`) are not read and never shown.

### properties.confidence and properties.pre_fp_confidence

| Path | Type | Optional | Allowed values | Effect | Missing or invalid |
|------|------|----------|----------------|--------|--------------------|
| `results[].properties.confidence` | number, or string | yes | number from 0.0 to 1.0 (values outside are clamped), or `very-high`, `high`, `medium`, `low` | `Confidence` row, for example `High (85%)`. With `--required` thresholds it decides the class when there is no `fp` verdict. | Falls back to the rule's confidence tag, then `precision`; with none the row is omitted. |
| `results[].properties.pre_fp_confidence` | number, or string | yes | same as `confidence` | The value before the false-positive review. The row reads `High (75%) → Very high (92%)`. | Omitted. It only shows next to a resolved `confidence`; no prior value is invented. |

Labels by value: 0.90 and above `Very high`, 0.75 and above `High`, 0.50 and above `Medium`, below that `Low`.

## Classification precedence

With `--required`, each non-suppressed result gets one class. The first rule that applies wins:

1. An applied `false_positive` or `accepted_risk` triage: **Excluded**, whatever the severity, verdict or flags. This also applies without `--required`.
2. An applied `true_positive` triage: **Required**.
3. Severity not listed in `--required`: **Recommended**.
4. Severity listed in `--never-demote`: **Required**.
5. A recognized `fp.verdict`: `FALSE_POSITIVE` is Recommended, the other two are Required.
6. Otherwise the confidence threshold for the severity, if one is configured; with no threshold the finding is Required.

A pending request never changes the class. The full rules and the notice texts are in [Required and Recommended](cmd-to-html.md#required-and-recommended). A producer that computes the same classes must match the shared case table `internal/sarif/testdata/triage-classification-cases.json`, which lists a severity, a verdict and a triage for every case together with the expected class and decision kind.

## Annotated example

One run with provenance and three results:

- `F-3f9a2c1d`: a high finding the review confirmed, with a confidence before and after the review. Class: Required.
- `F-5b1c07aa`: a false positive that alice requested and bob approved. Class: Excluded; the panel links the request and the approval.
- `F-9d44e1c2`: a pending accepted-risk request on a high finding. Class: Required, with the waiting notice.

```json
{
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "Example Scanner",
          "semanticVersion": "1.4.0",
          "rules": [
            {
              "id": "python.sql-injection",
              "shortDescription": {
                "text": "SQL injection via string formatting"
              },
              "helpUri": "https://owasp.org/www-community/attacks/SQL_Injection",
              "properties": {
                "tags": [
                  "CWE-89: Improper Neutralization of Special Elements used in an SQL Command"
                ],
                "security-severity": "8.1"
              }
            },
            {
              "id": "python.weak-hash",
              "shortDescription": {
                "text": "Weak hash function"
              },
              "properties": {
                "precision": "medium"
              }
            },
            {
              "id": "python.open-redirect",
              "shortDescription": {
                "text": "Open redirect"
              },
              "properties": {
                "security-severity": "7.5"
              }
            }
          ]
        }
      },
      "versionControlProvenance": [
        {
          "repositoryUri": "https://github.com/scan-io-git/scanio-demo",
          "revisionId": "0123456789abcdef0123456789abcdef01234567",
          "branch": "feature/example"
        }
      ],
      "results": [
        {
          "ruleId": "python.sql-injection",
          "level": "error",
          "message": {
            "text": "User input reaches a SQL query built with string formatting."
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "app/db.py"
                },
                "region": {
                  "startLine": 12,
                  "endLine": 12,
                  "snippet": {
                    "text": "cursor.execute(\"SELECT * FROM users WHERE id=%s\" % user_id)"
                  }
                }
              }
            }
          ],
          "properties": {
            "findingId": "F-3f9a2c1d",
            "confidence": 0.92,
            "pre_fp_confidence": 0.75,
            "fp": {
              "verdict": "TRUE_POSITIVE",
              "reasoning": "The value comes from a request parameter and is not validated."
            }
          }
        },
        {
          "ruleId": "python.weak-hash",
          "level": "warning",
          "message": {
            "text": "MD5 is used to build a cache key."
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "app/cache.py"
                },
                "region": {
                  "startLine": 30,
                  "endLine": 30,
                  "snippet": {
                    "text": "key = hashlib.md5(name.encode()).hexdigest()"
                  }
                }
              }
            }
          ],
          "properties": {
            "findingId": "F-5b1c07aa",
            "confidence": 0.1,
            "pre_fp_confidence": 0.65,
            "fp": {
              "verdict": "FALSE_POSITIVE",
              "reasoning": "The checksum is used as a file name, not for security."
            },
            "triage": {
              "decision": "false_positive",
              "status": "applied",
              "author": "alice",
              "reason": "The checksum is used as a file name, not for security.",
              "approver": "bob",
              "ticket": null,
              "source": "https://github.com/scan-io-git/scanio-demo/pull/7#issuecomment-102",
              "requestSource": "https://github.com/scan-io-git/scanio-demo/pull/7#issuecomment-101",
              "decidedAt": "2026-11-17T04:10:16Z"
            }
          }
        },
        {
          "ruleId": "python.open-redirect",
          "level": "error",
          "message": {
            "text": "The redirect target comes from a query parameter."
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "app/redirect.py"
                },
                "region": {
                  "startLine": 19,
                  "endLine": 19,
                  "snippet": {
                    "text": "return redirect(request.args['next'])"
                  }
                }
              }
            }
          ],
          "properties": {
            "findingId": "F-9d44e1c2",
            "triage": {
              "decision": "accepted_risk",
              "status": "pending",
              "author": "alice",
              "reason": "The target is checked against an allow list.",
              "approver": null,
              "ticket": null,
              "source": "https://github.com/scan-io-git/scanio-demo/pull/7#issuecomment-103",
              "requestSource": null,
              "decidedAt": "2026-11-18T09:00:00Z"
            }
          }
        }
      ]
    }
  ]
}
```

Render it from an empty folder, so repository, commit and links come from `versionControlProvenance`:

```bash
scanio to-html -i results.sarif -o report.html -s "$(mktemp -d)" \
  -t ./templates/tohtml --vcs github --pull-request 7 --required critical,high
```

The report shows Required 2, Excluded 1, and an `Excluded in this PR` section that starts collapsed. A larger working input is [`example-pr-decisions.sarif`](../../templates/tohtml/example/example-pr-decisions.sarif).
