#!/usr/bin/env bash
set -euo pipefail

PARENT_ROOT="${1:-}"
WORKTREE="${2:-$(pwd)}"
AGENT_ID="${3:-unknown}"
MAIN_BRANCH="main"

BRANCH="$(git -C "$WORKTREE" branch --show-current 2>/dev/null || true)"
if [[ -z "$BRANCH" ]]; then
  echo "[side-agent-start] Could not determine current branch in $WORKTREE."
  exit 1
fi

echo "[side-agent-start] agent=$AGENT_ID branch=$BRANCH main=$MAIN_BRANCH"

if [[ "$BRANCH" == "$MAIN_BRANCH" ]]; then
  echo "[side-agent-start] ERROR: child worktree is on $MAIN_BRANCH; expected a dedicated agent branch."
  exit 1
fi

# The worktree is already set to the parent's HEAD by the TypeScript extension.
# Just verify it's on the right branch.
echo "[side-agent-start] Worktree based on parent HEAD ($(git -C "$WORKTREE" rev-parse --short HEAD))."

# Project bootstrap
cd "$WORKTREE"

if [[ -f frontend/bun.lock ]]; then
  if command -v bun >/dev/null 2>&1; then
    (cd frontend && bun install)
  else
    echo "[side-agent-start] Warning: bun not found; skipping frontend dependency install."
  fi
fi

mkdir -p backend-go/frontend/dist
