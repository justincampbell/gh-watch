---
name: watch-pr
description: Watch a GitHub pull request for CI status, reviews, comments, merge conflicts, and terminal states using the gh-watch extension. Use when the user wants to monitor a PR, wait for CI, or track PR progress.
argument-hint: "[PR number]"
---

# Watch a Pull Request

Monitor a pull request for state changes using the `gh watch` CLI extension. It polls GitHub and emits one JSON event per line to stdout, which makes it a natural fit for the **Monitor tool** — each event arrives in the conversation as a notification.

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

Run with the **Monitor tool** (not `Bash` with `run_in_background`). Each stdout line — starting with `initial-state` — is delivered as a notification, so you react to events as they land instead of polling an output file.

**Wait for CI to finish, then exit** — use a bounded `timeout_ms` and let `--exit-on` end the watch:

- `command`: `gh watch pr $ARGUMENTS --exit-on ci-passed,ci-failed,pr-merged,pr-closed`
- `description`: `PR $ARGUMENTS CI status`
- `persistent`: `false`
- `timeout_ms`: `1800000` (30 min — raise for slow CI)

**Wait for a code review:**

- `command`: `gh watch pr $ARGUMENTS --exit-on review-submitted`

**Watch the PR for the rest of the session** (CI + reviews + comments + merge state):

- `command`: `gh watch pr $ARGUMENTS`
- `description`: `PR $ARGUMENTS state changes`
- `persistent`: `true` (no timeout — stop with `TaskStop` or end of session)

If no PR number is provided, the PR for the current branch is detected automatically. To watch a PR in another repo, prefix the command with `GH_REPO=owner/repo` (the `pr` subcommand takes a number, not a URL).

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

**The first notification is always `initial-state`.** Report it to the user immediately — don't just say "watching in background." Tell them what you see: how many checks passed/pending/failed, review status, mergeable state.

**Use `initial-state` to decide next steps:**
- If CI already passed and reviews are approved → you may not need to wait; check if the PR can be merged or marked ready
- If CI already failed → investigate immediately instead of waiting (consider stopping the monitor with `TaskStop`)
- If there are merge conflicts → flag them before waiting for CI
- If reviews have changes requested → address those before watching CI

**Subsequent notifications are state changes.** Each one is the full JSON event — react in context (e.g., "CI failed" combined with initial-state showing 28/29 passed tells you only 1 check broke).
