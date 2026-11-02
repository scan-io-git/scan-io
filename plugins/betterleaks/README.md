# Betterleaks Plugin

Wraps the [betterleaks](https://github.com/betterleaks/betterleaks) v2 CLI. The plugin runs the scanner and converts its JSON report to SARIF. Which betterleaks flags to use is up to the caller.

## Usage

```
scanio analyse --scanner betterleaks [--command fs|git] [--config <betterleaks.toml>] \
  --format sarif --output result.sarif <target> -- <betterleaks flags>
```

Command line built by the plugin:

```
betterleaks <command> [additional args...] [-c <config>] --exit-code 2 -o <result>.json <target>
```

- `--command`: `fs` (default) or `git`. Anything else is a validation error.
- Additional args pass through unchanged, for example `--no-allow-signatures --ignore-file /dev/null --no-banner --no-color -j 1`, or `--log-opts="--no-merges A..B" --include=commit-messages` for `git`.
- The environment is inherited as is. The plugin writes nothing into the target.

## Result handling

The scan counts as successful only when betterleaks exits with 0 (clean) or 2 (findings), the JSON report parses, and `scan.state` is `complete`. Otherwise the plugin returns an error that includes the betterleaks output, writes no SARIF, and keeps the raw `.json` if one exists. Partial findings are never converted.

## Formats

- `sarif`: the `.json` report is converted to `.sarif`; both are kept and the `.sarif` path is returned.
- `json`: the raw report is returned.
- Anything else: warning, falls back to `json`.

## SARIF mapping

- Driver `betterleaks`, version from `scan.betterleaks_version`.
- `level` comes from a rule tag `severity:<critical|high|medium|low>` (error, error, warning, note). Without a tag it is `error`. Confidence never affects the level.
- `properties.confidence` carries the betterleaks value. Other properties: `resource`, `git.sha`, `git.date`, `validation_status`, `encodings`, `decode_depth`, `component_rule_ids`.
- Secret-bearing fields are copied as properties: `secret` (`match.value`), `captures` (`match.captures`, object of named groups), `git.message` (`attributes.git.message`, history scans), `validation_reason` and `validation_metadata` (`analysis.status_reason` and `status_metadata`). Each is omitted when empty.
- `components` lists every `component_sets[].components[]` entry as `{set, rule_id, optional, value, full, location}`. `set` is the 0-based component set index and `location` keeps betterleaks coordinates (1-based, inclusive `end_column`, unlike `region`).
- `partialFingerprints["betterleaks/v1"]` is `match.fingerprint`. `snippet.text` is `match.full`.
- Findings without a path (commit messages) have no location.
- `match.context` becomes `locations[0].physicalLocation.contextRegion.snippet.text`. betterleaks does not report the context's line range, so the region has no line numbers. Commit-message findings have no location, so their context is not emitted.
- Git author and committer fields are never copied.
