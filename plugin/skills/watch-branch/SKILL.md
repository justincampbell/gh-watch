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
| `new-commit` | A new commit was pushed to the branch |

## Output format

One JSON object per line on stdout:

```json
{"timestamp":"2026-04-15T10:30:00Z","event":"new-commit","summary":"New commit on main: abc1234 Add feature","details":{"branch":"main","sha":"abc1234...","previous_sha":"def5678...","message_headline":"Add feature","author":"octocat"}}
```
