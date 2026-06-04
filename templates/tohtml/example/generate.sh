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

echo "Report written to templates/tohtml/example/example.html"
