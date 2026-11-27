#!/usr/bin/env bash
#
# Validates DB migrations added by a pull request:
#   1. Ordering  - an added migration must be NEWER than the latest migration
#                  already on the base branch. Goose applies migrations in
#                  timestamp order, so an older-timestamped migration would be
#                  silently skipped on databases already upgraded past it.
#   2. Uniqueness - no two migration files may share a timestamp.
#   3. Naming     - files must match YYYYMMDDHHMMSS_lower_snake_name.(sql|go).
#
# On failure it prints a human-readable message and, when running in GitHub
# Actions, emits an error annotation bound to the offending file so the message
# also renders inline in the PR "Files changed" tab.
#
# Compares HEAD against $BASE_REF (default origin/master). Requires full history
# (fetch-depth: 0 in CI).
# -e is intentionally omitted: the script accumulates violations into $status
# and must not exit on the first non-zero command (grep no-match, a false [[ ]]
# in an if, `is_migration || continue`).
set -uo pipefail
export LC_ALL=C

MIGRATIONS_DIR="db/migrations"
BASE_REF="${BASE_REF:-origin/master}"
NAME_RE='^[0-9]{14}_[a-z0-9_]+\.(sql|go)$'

status=0

# Log a message to stderr and mark the run as failed.
fail() {
  printf '%s\n' "$1" >&2
  status=1
}

# Emit a GitHub Actions error annotation bound to a file, so the message renders
# inline on the offending migration in the PR "Files changed" tab. No-op outside
# CI. `%`, newline and CR are encoded as required by the workflow-command syntax
# (the `%` replacement must run first so the encodings we add aren't re-escaped).
annotate() { # $1=file  $2=message
  [ "${GITHUB_ACTIONS:-}" = "true" ] || return 0
  local msg="$2"
  msg="${msg//'%'/%25}"
  msg="${msg//$'\n'/%0A}"
  msg="${msg//$'\r'/%0D}"
  printf '::error file=%s,line=1::%s\n' "$1" "$msg"
}

# Report a migration problem: log it, annotate the offending file, mark failed.
report() { # $1=file  $2=message
  fail "$2"
  printf '\n' >&2
  annotate "$1" "$2"
}

human_ts() {
  local t="$1"
  printf '%s-%s-%s %s:%s:%s' "${t:0:4}" "${t:4:2}" "${t:6:2}" "${t:8:2}" "${t:10:2}" "${t:12:2}"
}

is_migration() { # $1=basename -> 0 if a .sql/.go file with a 14-digit prefix
  local b="$1"
  case "$b" in
    *.sql | *.go) ;;
    *) return 1 ;;
  esac
  [[ "${b%%_*}" =~ ^[0-9]{14}$ ]]
}

if ! git rev-parse --verify --quiet "$BASE_REF" >/dev/null; then
  printf '❌ Cannot resolve base ref "%s". In CI, check out with fetch-depth: 0.\n' "$BASE_REF" >&2
  exit 1
fi

# --- Newest timestamp already on the base branch ---
base_max=""
base_max_file=""
while IFS= read -r f; do
  [ -z "$f" ] && continue
  b="$(basename "$f")"
  is_migration "$b" || continue
  ts="${b%%_*}"
  if [[ "$ts" > "$base_max" ]]; then
    base_max="$ts"
    base_max_file="$f"
  fi
done < <(git ls-tree -r --name-only "$BASE_REF" -- "$MIGRATIONS_DIR" 2>/dev/null)

# --- Ordering + naming on files added by this PR ---
while IFS= read -r f; do
  [ -z "$f" ] && continue
  b="$(basename "$f")"
  case "$b" in
    *.sql) ;;                                  # any .sql in this dir must be a migration
    *.go) [[ "$b" == [0-9]* ]] || continue ;;  # non-timestamped .go = helper (e.g. migration.go), skip
    *) continue ;;
  esac
  if [ "${f%/*}" != "$MIGRATIONS_DIR" ]; then
    report "$f" "❌ Migration file in a subdirectory: $f
   Migrations must live directly in $MIGRATIONS_DIR/ — only $MIGRATIONS_DIR/*.sql (and
   top-level .go migrations) are embedded, so a nested file would be SILENTLY SKIPPED.
   Move it to $MIGRATIONS_DIR/$b."
    continue
  fi
  if ! [[ "$b" =~ $NAME_RE ]]; then
    report "$f" "❌ Malformed migration filename: $f
   Expected YYYYMMDDHHMMSS_lower_snake_name.(sql|go); the name segment must be lowercase.
   Regenerate with: make migration-sql name=<description>  (or make migration-go name=<description>)"
    continue
  fi
  ts="${b%%_*}"
  if [[ -n "$base_max" ]] && ! [[ "$ts" > "$base_max" ]]; then
    report "$f" "❌ Migration ordering error: $f ($(human_ts "$ts"))
   is older than (or equal to) the newest migration already on ${BASE_REF#origin/}:
   $base_max_file ($(human_ts "$base_max"))

   Goose applies migrations in timestamp order, so databases already upgraded
   past that point would SILENTLY SKIP your migration.

   Fix: regenerate it with a current timestamp:
     make migration-sql name=<description>   (or make migration-go name=<description>)
   then move your SQL/Go body into the new file and delete the old one."
  fi
done < <(git diff --diff-filter=A --name-only "$BASE_REF"...HEAD -- "$MIGRATIONS_DIR" 2>/dev/null)

# --- Duplicate timestamps across the merged set (HEAD) ---
all_migs="$(git ls-tree -r --name-only HEAD -- "$MIGRATIONS_DIR" 2>/dev/null)"
dups="$(printf '%s\n' "$all_migs" | while IFS= read -r f; do
  b="$(basename "$f")"
  is_migration "$b" || continue
  printf '%s\n' "${b%%_*}"
done | sort | uniq -d)"
if [ -n "$dups" ]; then
  while IFS= read -r ts; do
    [ -z "$ts" ] && continue
    colliding="$(printf '%s\n' "$all_migs" | grep "/${ts}_" || true)"
    printf '❌ Duplicate migration timestamp %s used by multiple files:\n' "$ts" >&2
    while IFS= read -r cf; do
      [ -z "$cf" ] && continue
      printf '   %s\n' "$cf" >&2
      annotate "$cf" "Duplicate migration timestamp $ts — shared by another migration. Timestamps must be unique; regenerate one with make migration-*."
    done <<< "$colliding"
    printf '   Every migration needs a unique timestamp. Regenerate one with make migration-*.\n' >&2
    status=1
  done <<< "$dups"
fi

if [ "$status" -eq 0 ]; then
  echo "✅ DB migrations OK (ordering, uniqueness, naming)."
fi
exit "$status"
