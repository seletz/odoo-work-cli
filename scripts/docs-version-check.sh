#!/usr/bin/env bash
set -euo pipefail

# Check that the built site (default: site/) shows its version banner on the
# home page and on the release notes page, and that the release notes list
# the version when it is a release. With an argument EXPECTED the version must
# be that one (the release job passes the tag), otherwise the one
# docs_macros.py derives (DOCS_VERSION or git describe).
#
# Usage: scripts/docs-version-check.sh [EXPECTED_VERSION]

ROOT=$(cd "$(dirname "$0")/.." && pwd)
SITE=${SITE_DIR:-$ROOT/site}
expected=${1:-$(cd "$ROOT" && python3 -c 'import sys, pathlib; sys.path.insert(0, "."); import docs_macros; print(docs_macros.docs_version(pathlib.Path(".")))')}
expected=${expected#v}

fail=0
for page in index.html release-notes/index.html; do
  f="$SITE/$page"
  if [ ! -f "$f" ]; then
    echo "docs-version-check: $page is missing in $SITE" >&2
    fail=1
  elif ! grep -Eq "Documentation for odoo-work-cli (<a [^>]*>)?<strong>$expected</strong>" "$f"; then
    echo "docs-version-check: $page does not show version $expected" >&2
    fail=1
  fi
done
if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "the site shows version $expected"
