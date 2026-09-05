#!/usr/bin/env bash
#
# Emits per-area change flags to $GITHUB_OUTPUT so the pipeline can skip work a
# pull request cannot affect:
#
#   go    - Go sources, module files, linter config, embedded resources
#   js    - anything under ui/
#   i18n  - translation files and their validation script
#   build - anything that ends up in a binary, image or package (i.e. every
#           change except the doc-only paths in $DOC_ONLY_RE)
#
# Only pull requests are narrowed. Master pushes and tags always get every flag,
# so a release can never be built from a partially validated tree.
#
# Flags gate STEPS, not jobs: a job-level skip propagates through the needs
# chain (actions/runner#491) and would take the release jobs down with it.
#
# Compares HEAD against $BASE_REF (default master). Requires full history
# (fetch-depth: 0 in CI).
set -uo pipefail
export LC_ALL=C

GO_RE='(\.go$|(^|/)go\.(mod|sum)$|^Makefile$|^\.golangci\.yml$|^resources/)'
JS_RE='^ui/'
I18N_RE='(^resources/i18n/|^\.github/workflows/validate-translations\.sh$)'
DOC_ONLY_RE='(\.md$|^LICENSE$|^\.git-blame-ignore-revs$|^\.gitignore$|^\.devcontainer/)'

emit() { printf '%s=%s\n' "$1" "$2" | tee -a "${GITHUB_OUTPUT:-/dev/null}"; }

if [ "${GITHUB_EVENT_NAME:-}" != "pull_request" ]; then
  echo "Not a pull request — running everything."
  for area in go js i18n build; do emit "$area" true; done
  exit 0
fi

BASE_REF="${BASE_REF:-master}"
git fetch --no-tags --quiet origin "+refs/heads/${BASE_REF}:refs/remotes/origin/${BASE_REF}"

files="$(git diff --name-only "origin/${BASE_REF}...HEAD")"
echo "Changed files:"
printf '%s\n' "$files" | sed 's/^/  /'
echo

flag() { # $1=name  $2=regex
  if printf '%s\n' "$files" | grep -qE "$2"; then emit "$1" true; else emit "$1" false; fi
}

flag go "$GO_RE"
flag js "$JS_RE"
flag i18n "$I18N_RE"

if printf '%s\n' "$files" | grep -vE "$DOC_ONLY_RE" | grep -q '[^[:space:]]'; then
  emit build true
else
  emit build false
fi
