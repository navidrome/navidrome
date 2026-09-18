#!/usr/bin/env bash
#
# Tests detect-changes.sh against a throwaway repo: builds a synthetic PR for
# each change shape and asserts the four emitted flags.
#
#   ./.github/workflows/detect-changes_test.sh          # tests the sibling script
#   ./.github/workflows/detect-changes_test.sh <path>   # tests another copy
#
# To confirm a case still bites, edit a pattern out of detect-changes.sh and
# re-run: exactly the case that covers it should fail.
set -uo pipefail
SCRIPT="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/detect-changes.sh}"
T=$(mktemp -d); O=$(mktemp -d)
cd "$T"
git init -q -b master .
git config user.email t@t; git config user.name t
mkdir -p ui/src/i18n resources/i18n .github/workflows db/migrations
echo x > README.md; echo x > main.go; echo x > ui/src/a.js
echo x > resources/i18n/pt.json; echo x > ui/src/i18n/en.json; echo x > go.mod
git add -A; git commit -qm base
git clone -q --bare . "$O/origin.git"
git remote add origin "$O/origin.git"
git fetch -q origin

fails=0
run() { # $1=label $2=expected "go js i18n build" ; rest=files
  label="$1"; want="$2"; shift 2
  git checkout -q -B test master
  for f in "$@"; do mkdir -p "$(dirname "$f")"; echo change >> "$f"; done
  git add -A >/dev/null; git commit -qm "$label"
  got=$(GITHUB_EVENT_NAME=pull_request BASE_REF=master bash "$SCRIPT" 2>&1 \
        | grep -E '^(go|js|i18n|build)=' | cut -d= -f2 | tr '\n' ' ' | sed 's/ $//')
  if [ "$got" = "$want" ]; then printf 'ok    %-32s %s\n' "$label" "$got"
  else printf 'FAIL  %-32s got[%s] want[%s]\n' "$label" "$got" "$want"; fails=$((fails+1)); fi
}

#                                      go    js    i18n  build
run "docs only"        "false false false false" README.md
run "gitignore only"   "false false false false" .gitignore
run "go only"          "true false false true"   core/thing.go
run "ui only"          "false true false true"   ui/src/b.js
run "i18n resources"   "true false true true"    resources/i18n/fr.json
run "ui en.json"       "false true true true"    ui/src/i18n/en.json
run "ui other i18n"    "false true false true"   ui/src/i18n/provider.js
run "db migration sql" "true false false true"   db/migrations/20260101000000_x.sql
run "tests fixture"      "true false false true"   tests/fixtures/playlist.m3u
run "tests toml"         "true false false true"   tests/navidrome-test.toml
run "conf testdata"      "false false false true"  conf/testdata/cfg.toml
run "manifest schema"  "true false false true"   plugins/manifest-schema.json
run "other plugin json" "false false false true"  plugins/testdata/fake/manifest-schema.json
run "nested go.mod"    "true false false true"   plugins/testdata/x/go.mod
run "Dockerfile only"  "false false false true"  Dockerfile
run "pipeline.yml"     "true true true true"    .github/workflows/pipeline.yml
run "detect-changes.sh" "true true true true"   .github/workflows/detect-changes.sh
run "other workflow"   "false false false true"  .github/workflows/stale.yml
run "validate-trans.sh" "false false true true"  .github/workflows/validate-translations.sh

echo "--- large diff must not lose flags to SIGPIPE ---"
git checkout -q -B test master
mkdir -p big/pkg
python3 -c "
import os
os.makedirs('big/pkg', exist_ok=True)
open('big/pkg/aaa_first.go','w').write('x')
for i in range(2500): open('big/pkg/filler_%04d.txt' % i,'w').write('x')
"
git add -A >/dev/null; git commit -qm big
nfiles=$(git diff --no-renames --name-only master...HEAD | wc -l | tr -d ' ')
for i in 1 2 3 4 5; do
  got=$(GITHUB_EVENT_NAME=pull_request BASE_REF=master bash "$SCRIPT" 2>&1 \
        | grep -E '^(go|js|i18n|build)=' | cut -d= -f2 | tr '\n' ' ' | sed 's/ $//')
  if [ "$got" = "true false false true" ]; then printf 'ok    %-32s %s (%s files)\n' "large diff run $i" "$got" "$nfiles"
  else printf 'FAIL  %-32s got[%s] want[true false false true] (%s files)\n' "large diff run $i" "$got" "$nfiles"; fails=$((fails+1)); fi
done

echo "--- renames must count the source path ---"
rn() { # $1=label $2=expected $3=from $4=to
  git checkout -q -B test master
  mkdir -p "$(dirname "$4")"; git mv "$3" "$4"
  git add -A >/dev/null; git commit -qm "$1"
  got=$(GITHUB_EVENT_NAME=pull_request BASE_REF=master bash "$SCRIPT" 2>&1 \
        | grep -E '^(go|js|i18n|build)=' | cut -d= -f2 | tr '\n' ' ' | sed 's/ $//')
  if [ "$got" = "$2" ]; then printf 'ok    %-32s %s\n' "$1" "$got"
  else printf 'FAIL  %-32s got[%s] want[%s]\n' "$1" "$got" "$2"; fails=$((fails+1)); fi
}
#                          go    js    i18n  build
rn "ui .js -> docs .md"  "false true false true"  ui/src/a.js docs/a.js.md
rn "go -> docs .md"      "true false false true"  main.go docs/main.go.md

echo "--- non-PR events ---"
git checkout -q master
for ev in push workflow_dispatch; do
  got=$(GITHUB_EVENT_NAME=$ev bash "$SCRIPT" 2>&1 | grep -E '^(go|js|i18n|build)=' | cut -d= -f2 | tr '\n' ' ' | sed 's/ $//')
  if [ "$got" = "true true true true" ]; then printf 'ok    %-32s %s\n' "$ev" "$got"
  else printf 'FAIL  %-32s got[%s]\n' "$ev" "$got"; fails=$((fails+1)); fi
done

echo "--- unresolvable base ref must fail closed ---"
git checkout -q -B test master; echo x >> main.go; git add -A >/dev/null; git commit -qm x
out=$(GITHUB_EVENT_NAME=pull_request BASE_REF=does-not-exist bash "$SCRIPT" 2>&1); rc=$?
if [ "$rc" != "0" ] && ! grep -qE '^(go|js|i18n|build)=' <<<"$out"; then
  printf 'ok    %-32s exit=%s, no flags emitted\n' "bad base ref" "$rc"
else
  printf 'FAIL  %-32s exit=%s out[%s]\n' "bad base ref" "$rc" "$out"; fails=$((fails+1))
fi

echo
echo "failures: $fails"
cd /; rm -rf "$T" "$O"
exit "$fails"
