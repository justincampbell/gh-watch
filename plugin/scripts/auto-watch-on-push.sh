#!/bin/bash
# PostToolUse hook: after a Bash tool call that includes "git push",
# check if there's a PR for the current branch and start gh watch pr
# in the background.

set -euo pipefail

# Read hook input from stdin
INPUT=$(cat)

# Extract the command that was run
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# Only act on git push commands
if ! echo "$COMMAND" | grep -qE 'git\s+push'; then
  exit 0
fi

# Check if gh watch is installed
if ! gh watch --help >/dev/null 2>&1; then
  exit 0
fi

# Check if there's already a gh watch process running for this repo
if pgrep -f "gh-watch.*watch.*pr" >/dev/null 2>&1; then
  exit 0
fi

# Try to detect a PR for the current branch
PR_NUMBER=$(gh pr view --json number --jq '.number' 2>/dev/null || true)

if [ -z "$PR_NUMBER" ]; then
  exit 0
fi

# Start watching in the background
nohup gh watch pr "$PR_NUMBER" --json >/tmp/gh-watch-pr-${PR_NUMBER}.log 2>&1 &

# Report back to Claude
echo "Started watching PR #${PR_NUMBER} in the background (pid $!)"
