#!/usr/bin/env bash
#
# Setup a git worktree for Navidrome development.
# This script is called automatically by `make worktree` and by Claude Code's
# worktree isolation, but can also be run standalone:
#
#   ./scripts/setup-worktree.sh <worktree-path> [--go-only]
#
# Options:
#   --go-only   Skip frontend (npm) setup. Useful for agents working only on Go code.
#
set -euo pipefail

WORKTREE_PATH="${1:?Usage: $0 <worktree-path> [--go-only]}"
GO_ONLY="${2:-}"

# Resolve the main worktree root (where the original repo lives)
MAIN_WORKTREE="$(git -C "$WORKTREE_PATH" worktree list --porcelain | head -1 | sed 's/^worktree //')"

if [ ! -d "$WORKTREE_PATH" ]; then
    echo "ERROR: Worktree path does not exist: $WORKTREE_PATH"
    exit 1
fi

cd "$WORKTREE_PATH"

echo "==> Setting up worktree at $WORKTREE_PATH"

# 1. Download Go dependencies
echo "==> Downloading Go dependencies..."
go mod download

# 2. Install frontend dependencies (unless --go-only)
if [ "$GO_ONLY" != "--go-only" ]; then
    echo "==> Installing frontend dependencies..."
    (cd ui && npm ci --prefer-offline --no-audit 2>/dev/null || npm ci)
else
    echo "==> Skipping frontend setup (--go-only)"
fi

# 3. Create required directories
mkdir -p data

# 4. Copy navidrome.toml from main worktree if it exists and not already present
if [ ! -f navidrome.toml ] && [ -f "$MAIN_WORKTREE/navidrome.toml" ]; then
    echo "==> Copying navidrome.toml from main worktree..."
    cp "$MAIN_WORKTREE/navidrome.toml" navidrome.toml
fi

# 5. Copy existing database from main worktree (already migrated and scanned)
#    This is much faster than running migrations + a full scan from scratch.
if [ ! -f data/navidrome.db ] && [ -f "$MAIN_WORKTREE/data/navidrome.db" ]; then
    echo "==> Copying database from main worktree (pre-migrated, pre-scanned)..."
    cp "$MAIN_WORKTREE/data/navidrome.db" data/navidrome.db
fi

echo "==> Worktree setup complete: $WORKTREE_PATH"
