#!/bin/bash
# Validates that new migrations in a PR have timestamps greater than
# the latest migration timestamp on the master branch.
#
# This prevents migration ordering conflicts when multiple PRs add migrations.
# Modified existing migrations only produce warnings, not errors.

set -e

# Get the latest migration timestamp from master branch
# Filter for files matching the pattern: 14-digit timestamp followed by _ and ending in .sql or .go
MASTER_MIGRATIONS=$(git ls-tree --name-only origin/master -- db/migrations/ | grep -E '^db/migrations/[0-9]{14}_.*\.(sql|go)$' || true)

if [ -z "$MASTER_MIGRATIONS" ]; then
    echo "No migrations found on master branch"
    exit 0
fi

MASTER_LATEST=$(echo "$MASTER_MIGRATIONS" | sed 's|db/migrations/||' | cut -c1-14 | sort -n | tail -1)

# Get NEW migrations (added in this PR)
NEW_MIGRATIONS=$(git diff --name-only --diff-filter=A origin/master -- db/migrations/ | grep -E '^db/migrations/[0-9]{14}_.*\.(sql|go)$' || true)

# Get MODIFIED migrations (existing files that were changed)
MODIFIED_MIGRATIONS=$(git diff --name-only --diff-filter=M origin/master -- db/migrations/ | grep -E '^db/migrations/[0-9]{14}_.*\.(sql|go)$' || true)

if [ -z "$NEW_MIGRATIONS" ] && [ -z "$MODIFIED_MIGRATIONS" ]; then
    echo "No new or modified migrations found in this PR"
    exit 0
fi

echo "Latest migration on master: $MASTER_LATEST"

HAS_ERRORS=false

# Check NEW migrations - these MUST have valid timestamps (errors)
if [ -n "$NEW_MIGRATIONS" ]; then
    echo ""
    echo "New migrations in this PR:"
    for migration in $NEW_MIGRATIONS; do
        TIMESTAMP=$(basename "$migration" | cut -c1-14)
        echo "  - $migration (timestamp: $TIMESTAMP)"
        
        if [ "$TIMESTAMP" -le "$MASTER_LATEST" ]; then
            echo "::error file=$migration::Migration timestamp $TIMESTAMP must be greater than latest master timestamp $MASTER_LATEST"
            HAS_ERRORS=true
        fi
    done
fi

# Check MODIFIED migrations - only warn, don't fail
if [ -n "$MODIFIED_MIGRATIONS" ]; then
    echo ""
    echo "Modified existing migrations in this PR:"
    for migration in $MODIFIED_MIGRATIONS; do
        TIMESTAMP=$(basename "$migration" | cut -c1-14)
        echo "  - $migration (timestamp: $TIMESTAMP)"
        echo "::warning file=$migration::Modifying existing migration files may cause issues for users who have already applied them"
    done
fi

if [ "$HAS_ERRORS" = "true" ]; then
    echo ""
    echo "ERROR: One or more NEW migrations have timestamps that are not after the latest migration on master."
    echo "Please regenerate the migration with a newer timestamp using:"
    echo "  make migration-sql name=your_migration_name"
    echo "  make migration-go name=your_migration_name"
    exit 1
fi

echo ""
echo "All migration timestamps are valid!"
