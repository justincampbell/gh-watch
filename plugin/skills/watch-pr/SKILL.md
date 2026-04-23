---
name: watch-pr
description: Watch a GitHub pull request for CI status, reviews, comments, merge conflicts, and terminal states using the gh-watch extension. Use when the user wants to monitor a PR, wait for CI, or track PR progress.
argument-hint: "[PR number]"
---

# Watch a Pull Request

Monitor a pull request for state changes using the `gh watch` CLI extension. Run it as a background process — it polls GitHub and emits one JSON event per line to stdout.

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

Run as a **background task** (Bash with `run_in_background: true`). Always use `--exit` or `--exit-on` so the process terminates.

**Wait for CI to finish (pass or fail):**

```
gh watch pr $ARGUMENTS --exit-on ci-passed,ci-failed
```

**Wait for a code review:**

```
gh watch pr $ARGUMENTS --exit-on review-submitted
```

**Exit on any meaningful change:**

```
gh watch pr $ARGUMENTS --exit
```

If no PR number is provided, the PR for the current branch is detected automatically.

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--interval <duration>` | Polling interval | `60s` |
| `--exit` | Exit after any state change | `false` |
| `--exit-on <events>` | Exit after specific event types (comma-separated) | |

## Event types

| Event | Description |
|-------|-------------|
| `initial-state` | Snapshot of PR state at the moment watching started (always emitted first) |
| `ci-passed` | All CI checks passed |
| `ci-failed` | CI failed (at least one check failed) |
| `review-submitted` | A review was submitted |
| `comment-added` | A new comment was posted |
| `merge-conflict-changed` | Mergeable state changed |
| `pr-merged` | PR was merged (terminal — always exits) |
| `pr-closed` | PR was closed (terminal — always exits) |

## Output format

One JSON object per line on stdout. The **first line is always `initial-state`** — a snapshot of the PR at the moment watching began:

```json
{"timestamp":"...","event":"initial-state","summary":"PR #123: Title — CI: 5/8 passed, 3 pending, 1 reviews","details":{"number":123,"title":"...","status":"open","mergeable":"MERGEABLE","checks":8,"passed":5,"failed":0,"pending":3,"reviews":1,"comments":2}}
```

Subsequent lines are change events:

```json
{"timestamp":"...","event":"ci-passed","summary":"All CI checks passed","details":{}}
```

## Interpreting the output

**Always read and report the `initial-state` first.** It tells you the current PR state — don't just say "watching in background." Tell the user what you see: how many checks passed/pending/failed, review status, mergeable state.

**Use `initial-state` to decide next steps:**
- If CI already passed and reviews are approved → you may not need to wait; check if the PR can be merged or marked ready
- If CI already failed → investigate immediately instead of waiting
- If there are merge conflicts → flag them before waiting for CI
- If reviews have changes requested → address those before watching CI

**When the watcher exits**, the output file contains both the initial-state and the exit event. Read the exit event to determine what happened, but cross-reference with initial-state for full context (e.g., "CI failed" + initial-state showing 28/29 passed tells you only 1 check broke).
