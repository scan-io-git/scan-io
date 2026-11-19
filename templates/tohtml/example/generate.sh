#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

TMP="$(mktemp -d)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

# Create a throwaway git repo whose remote identity matches the demo organisation.
# A deterministic empty commit keeps the permalink commit ref stable across re-runs.
cd "$TMP"
git init -b main -q
git config user.email "scanio-demo@example.com"
git config user.name  "Scanio Demo"
git remote add origin "https://github.com/scan-io-git/scanio-demo.git"
GIT_AUTHOR_DATE="2024-01-15T12:00:00Z" \
GIT_COMMITTER_DATE="2024-01-15T12:00:00Z" \
  git commit --allow-empty -m "chore: example report baseline" -q

cd "$REPO_ROOT"
go run . to-html \
  --input    "$SCRIPT_DIR/example.sarif" \
  --output   "$SCRIPT_DIR/example.html" \
  --source   "$TMP" \
  --vcs      github \
  --templates-path "$SCRIPT_DIR/.." \
  --title    "Scanio Demo Report"

go run . to-html \
  --input        "$SCRIPT_DIR/example.sarif" \
  --output       "$SCRIPT_DIR/example-pr.html" \
  --source       "$TMP" \
  --vcs          github \
  --pull-request 42 \
  --templates-path "$SCRIPT_DIR/.." \
  --title        "Scanio Demo Report (PR #42)"

go run . to-html \
  --input    "$SCRIPT_DIR/example.sarif" \
  --output   "$SCRIPT_DIR/example-required.html" \
  --source   "$TMP" \
  --vcs      github \
  --required "critical,high" \
  --templates-path "$SCRIPT_DIR/.." \
  --title    "Scanio Demo Report (Required/Recommended)"

go run . to-html \
  --input    "$SCRIPT_DIR/example.sarif" \
  --input    "$SCRIPT_DIR/example-ai-scan.sarif" \
  --output   "$SCRIPT_DIR/example-consolidated.html" \
  --source   "$TMP" \
  --vcs      github \
  --required "critical,high,medium,low" \
  --templates-path "$SCRIPT_DIR/.." \
  --title    "Scanio Demo Report (Consolidated)"

# PR decisions and finding IDs. The source folder has no .git on purpose: repository,
# commit, branch and every link come from the SARIF's versionControlProvenance, and
# the code snippets come from the SARIF itself.
NOGIT="$(mktemp -d)"
trap 'cleanup; rm -rf "$NOGIT"' EXIT
go run . to-html \
  --input        "$SCRIPT_DIR/example-pr-decisions.sarif" \
  --input        "$SCRIPT_DIR/example-pr-decisions-ai.sarif" \
  --output       "$SCRIPT_DIR/example-pr-decisions.html" \
  --source       "$NOGIT" \
  --vcs          github \
  --pull-request 7 \
  --required     "critical,high,medium" \
  --never-demote "critical" \
  --templates-path "$SCRIPT_DIR/.." \
  --title        "Scanio Demo Report (PR decisions)"

echo "Reports written to templates/tohtml/example/example.html, example-pr.html, example-required.html, example-consolidated.html and example-pr-decisions.html"
