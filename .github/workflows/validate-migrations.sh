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
# Compares HEAD against $BASE_REF (default origin/master). Requires full history
# (fetch-depth: 0 in CI).
set -uo pipefail
export LC_ALL=C

MIGRATIONS_DIR="db/migrations"
BASE_REF="${BASE_REF:-origin/master}"
NAME_RE='^[0-9]{14}_[a-z0-9_]+\.(sql|go)$'

status=0
fail() {
  printf '%s\n' "$@" >&2
  status=1
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
    *.sql | *.go) ;;
    *) continue ;;
  esac
  if ! [[ "$b" =~ $NAME_RE ]]; then
    fail \
      "❌ Malformed migration filename: $f" \
      "   Expected YYYYMMDDHHMMSS_lower_snake_name.(sql|go); the name segment must be lowercase." \
      "   Regenerate with: make migration-sql name=<description>  (or make migration-go name=<description>)" \
      ""
    continue
  fi
  ts="${b%%_*}"
  if [[ -n "$base_max" ]] && ! [[ "$ts" > "$base_max" ]]; then
    fail \
      "❌ Migration ordering error:" \
      "   $f ($(human_ts "$ts"))" \
      "   is older than (or equal to) the newest migration already on ${BASE_REF#origin/}:" \
      "   $base_max_file ($(human_ts "$base_max"))" \
      "" \
      "   Goose applies migrations in timestamp order, so databases already upgraded" \
      "   past that point would SILENTLY SKIP your migration." \
      "" \
      "   Fix: regenerate it with a current timestamp:" \
      "     make migration-sql name=<description>   (or make migration-go name=<description>)" \
      "   then move your SQL/Go body into the new file and delete the old one." \
      ""
  fi
done < <(git diff --diff-filter=A --name-only "$BASE_REF"...HEAD -- "$MIGRATIONS_DIR" 2>/dev/null)

# --- Duplicate timestamps across the merged set (HEAD) ---
dups="$(git ls-tree -r --name-only HEAD -- "$MIGRATIONS_DIR" 2>/dev/null | while IFS= read -r f; do
  b="$(basename "$f")"
  is_migration "$b" || continue
  printf '%s\n' "${b%%_*}"
done | sort | uniq -d)"
if [ -n "$dups" ]; then
  while IFS= read -r ts; do
    [ -z "$ts" ] && continue
    colliding="$(git ls-tree -r --name-only HEAD -- "$MIGRATIONS_DIR" 2>/dev/null | grep "/${ts}_" || true)"
    fail \
      "❌ Duplicate migration timestamp $ts used by multiple files:" \
      "$(printf '   %s\n' $colliding)" \
      "   Every migration needs a unique timestamp. Regenerate one with make migration-*." \
      ""
  done <<< "$dups"
fi

if [ "$status" -eq 0 ]; then
  echo "✅ DB migrations OK (ordering, uniqueness, naming)."
fi
exit "$status"
