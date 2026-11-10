#!/usr/bin/env bash
set -euo pipefail

# Check that the generated parts of the docs are up to date: render them
# into a temporary directory (tools/docsgen) and compare with the committed
# files. A difference means the TUI, a command's output or a command's help
# changed and the docs were not regenerated (the command reference covers
# the site pages and the man pages); the script then says which files
# differ, how to fix it, and exits 1. The goldens, CLI samples, reference
# pages and man pages compare by bytes, the PNG screenshots by decoded
# pixels and dimensions (tools/docsgen compare-png). The mise tasks
# docs:screenshots:check and docs:reference:check and the CI call it.
#
# Usage: scripts/docs-check.sh screenshots|reference

ROOT=$(cd "$(dirname "$0")/.." && pwd)
WHAT=${1:-}

TMP=$(mktemp -d "${TMPDIR:-/tmp}/odoo-work-cli-docs-check.XXXXXX")
trap 'rm -rf "$TMP"' EXIT

stale=0
compare() { # committed fresh label
  if ! diff -r "$1" "$2" >"$TMP/diff"; then
    echo "$3 are stale:" >&2
    sed -e "s#$TMP#<fresh render>#g" "$TMP/diff" | head -n 60 >&2
    stale=1
  fi
}

cd "$ROOT"
case "$WHAT" in
  screenshots)
    go run ./tools/docsgen screenshots \
      -png "$TMP/png" -golden "$TMP/golden" -samples "$TMP/cli" -state "$TMP/demo-state.json" >/dev/null
    compare internal/docsgen/testdata/frames "$TMP/golden" "TUI goldens in internal/docsgen/testdata/frames/"
    # The PNGs compare by decoded pixels: their zlib bytes change with the
    # Go version, the image does not.
    if ! go run ./tools/docsgen compare-png docs/assets/screenshots "$TMP/png" 2>"$TMP/png.diff"; then
      echo "TUI screenshots in docs/assets/screenshots/ are stale:" >&2
      head -n 60 "$TMP/png.diff" >&2
      stale=1
    fi
    compare docs/assets/cli "$TMP/cli" "CLI samples in docs/assets/cli/"
    if ! cmp -s docs/assets/demo-state.json "$TMP/demo-state.json"; then
      echo "docs/assets/demo-state.json is stale" >&2
      stale=1
    fi
    fix="mise run docs:screenshots"
    ;;
  reference)
    go run ./tools/docsgen reference -out "$TMP/reference" -man "$TMP/man1" >/dev/null
    compare docs/reference/commands "$TMP/reference" "command reference pages in docs/reference/commands/"
    compare man/man1 "$TMP/man1" "man pages in man/man1/"
    fix="mise run docs:reference"
    ;;
  *)
    echo "usage: $0 screenshots|reference" >&2
    exit 2
    ;;
esac

if [ "$stale" -ne 0 ]; then
  echo "run \`$fix\` and commit the result" >&2
  exit 1
fi
echo "$WHAT are up to date"
