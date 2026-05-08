---
name: watch-branch
description: Watch a GitHub branch for new commits using the gh-watch extension. Use when the user wants to be notified when new commits are pushed to a branch, monitor main for merges, or track branch activity.
argument-hint: "<branch-name>"
---

# Watch a Branch

Monitor a branch for new commits using the `gh watch` CLI extension. It polls GitHub and emits one JSON event per line to stdout when the branch tip changes, which makes it a natural fit for the **Monitor tool** — each event arrives in the conversation as a notification.

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

Run with the **Monitor tool** (not `Bash` with `run_in_background`). The first notification is the `initial-state` snapshot; subsequent notifications are `new-commit` events as they land.

**Wait for the next commit on main, then exit:**

- `command`: `gh watch branch $ARGUMENTS --exit-on new-commit`
- `description`: `New commits on $ARGUMENTS`
- `persistent`: `false`
- `timeout_ms`: `3600000` (1 hour — raise/lower based on how long you're willing to wait)

**Watch the branch for the rest of the session** (every new commit reported as it lands):

- `command`: `gh watch branch $ARGUMENTS`
- `description`: `Commits on $ARGUMENTS`
- `persistent`: `true` (no timeout — stop with `TaskStop` or end of session)

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

**The first notification is always `initial-state`.** Report it to the user immediately — tell them the current branch tip (SHA, commit message, author). Don't just say "watching in background."

**Subsequent `new-commit` notifications** are full JSON events with the new SHA, message, author, and previous SHA — surface each one as it lands.
