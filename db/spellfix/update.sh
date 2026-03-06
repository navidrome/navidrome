#!/usr/bin/env bash
#
# Updates the vendored spellfix1 source files to match the SQLite version
# bundled with the current go-sqlite3 dependency.
#
set -euo pipefail

cd "$(dirname "$0")"

SQLITE_VERSION=$(grep '#define SQLITE_VERSION ' \
  "$(go env GOMODCACHE)/$(go list -m -f '{{.Path}}@{{.Version}}' github.com/mattn/go-sqlite3)/sqlite3-binding.h" \
  | awk '{gsub(/"/, "", $3); print $3}')

if [ -z "$SQLITE_VERSION" ]; then
  echo "ERROR: Could not determine SQLite version from go-sqlite3" >&2
  exit 1
fi

TAG="version-${SQLITE_VERSION}"
BASE_URL="https://raw.githubusercontent.com/sqlite/sqlite/${TAG}"

echo "SQLite version from go-sqlite3: ${SQLITE_VERSION}"
echo "Downloading from tag: ${TAG}"

curl -sfL "${BASE_URL}/ext/misc/spellfix.c" -o spellfix.c
echo "  Updated spellfix.c"

curl -sfL "${BASE_URL}/src/sqlite3ext.h" -o sqlite3ext.h
echo "  Updated sqlite3ext.h"

echo "Done."
