---
name: watch-branch
description: Watch a GitHub branch for new commits using the gh-watch extension. Use when the user wants to be notified when new commits are pushed to a branch, monitor main for merges, or track branch activity.
argument-hint: "<branch-name>"
---

# Watch a Branch

Monitor a branch for new commits using the `gh watch` CLI extension. Run it as a background process — it polls GitHub and emits one JSON event per line to stdout when the branch tip changes.

## Prerequisites

Check if the extension is installed:

```
gh watch --help
```

If not installed:

```
gh extension install justincampbell/gh-watch
```

## Usage

Run as a **background task** (Bash with `run_in_background: true`). Use `--exit` or `--exit-on` to control when the process terminates.

**Wait for the next commit on main:**

```
gh watch branch $ARGUMENTS --exit-on new-commit
```

**Exit on any change:**

```
gh watch branch $ARGUMENTS --exit
```

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--interval <duration>` | Polling interval | `60s` |
| `--exit` | Exit after any state change | `false` |
| `--exit-on <events>` | Exit after specific event types (comma-separated) | |

## Event types

| Event | Description |
|-------|-------------|
| `initial-state` | Snapshot of the branch tip at the moment watching started (always emitted first) |
| `new-commit` | A new commit was pushed to the branch |

## Output format

One JSON object per line on stdout. The **first line is always `initial-state`** — a snapshot of the branch at the moment watching began:

```json
{"timestamp":"...","event":"initial-state","summary":"Branch main at abc1234: Latest commit message","details":{"branch":"main","sha":"abc1234...","message_headline":"Latest commit message","author":"octocat"}}
```

Subsequent lines are change events:

```json
{"timestamp":"...","event":"new-commit","summary":"New commit on main: def5678 Add feature","details":{"branch":"main","sha":"def5678...","previous_sha":"abc1234...","message_headline":"Add feature","author":"octocat"}}
```

## Interpreting the output

**Always read and report the `initial-state` first.** Tell the user the current branch tip — the SHA, commit message, and author. Don't just say "watching in background."
