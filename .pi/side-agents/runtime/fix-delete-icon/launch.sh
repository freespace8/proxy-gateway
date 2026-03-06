#!/usr/bin/env bash
set -euo pipefail

AGENT_ID='fix-delete-icon'
PARENT_SESSION='/Users/taylor/.pi/agent/sessions/--Users-taylor-working-github.com-freespace8-proxy-gateway--/2026-03-06T00-59-34-748Z_6ac27259-2ac1-4129-bc65-58ba0346b0e9.jsonl'
PARENT_REPO='/Users/taylor/working/github.com/freespace8/proxy-gateway'
STATE_ROOT='/Users/taylor/working/github.com/freespace8/proxy-gateway'
WORKTREE='/Users/taylor/working/github.com/freespace8/proxy-gateway-agent-worktree-0001'
WINDOW_ID='@1'
PROMPT_FILE='/Users/taylor/working/github.com/freespace8/proxy-gateway/.pi/side-agents/runtime/fix-delete-icon/kickoff.md'
EXIT_FILE='/Users/taylor/working/github.com/freespace8/proxy-gateway/.pi/side-agents/runtime/fix-delete-icon/exit.json'
MODEL_SPEC='proxy-openai/gpt-5.4'
RUNTIME_DIR='/Users/taylor/working/github.com/freespace8/proxy-gateway/.pi/side-agents/runtime/fix-delete-icon'
START_SCRIPT="$WORKTREE/.pi/side-agent-start.sh"
CHILD_SKILLS_DIR="$WORKTREE/.pi/side-agent-skills"

export PI_SIDE_AGENT_ID="$AGENT_ID"
export PI_SIDE_PARENT_SESSION="$PARENT_SESSION"
export PI_SIDE_PARENT_REPO="$PARENT_REPO"
export PI_SIDE_AGENTS_ROOT="$STATE_ROOT"
export PI_SIDE_RUNTIME_DIR="$RUNTIME_DIR"

write_exit() {
  local code="$1"
  printf '{"exitCode":%d,"finishedAt":"%s"}
' "$code" "$(date -Is)" > "$EXIT_FILE"
}

cd "$WORKTREE"

if [[ -x "$START_SCRIPT" ]]; then
  set +e
  "$START_SCRIPT" "$PARENT_REPO" "$WORKTREE" "$AGENT_ID"
  start_exit=$?
  set -e
  if [[ "$start_exit" -ne 0 ]]; then
    echo "[side-agent] start script failed with code $start_exit"
    write_exit "$start_exit"
    read -n 1 -s -r -p "[side-agent] Press any key to close this tmux window..." || true
    echo
    tmux kill-window -t "$WINDOW_ID" || true
    exit "$start_exit"
  fi
fi

PI_CMD=(pi)
if [[ -n "$MODEL_SPEC" ]]; then
  PI_CMD+=(--model "$MODEL_SPEC")
fi
if [[ -d "$CHILD_SKILLS_DIR" ]]; then
  # agent-setup writes the child-only finish skill here; load it explicitly.
  PI_CMD+=(--skill "$CHILD_SKILLS_DIR")
fi

set +e
"${PI_CMD[@]}" "$(cat "$PROMPT_FILE")"
exit_code=$?
set -e

write_exit "$exit_code"

if [[ "$exit_code" -eq 0 ]]; then
  echo "[side-agent] Agent finished."
else
  echo "[side-agent] Agent exited with code $exit_code."
fi

read -n 1 -s -r -p "[side-agent] Press any key to close this tmux window..." || true
echo

tmux kill-window -t "$WINDOW_ID" || true
