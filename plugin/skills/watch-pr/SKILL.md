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
gh watch pr $ARGUMENTS --exit-on ci-passed --exit-on ci-failed
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
| `--exit-on <event>` | Exit after a specific event type | |

## Event types

| Event | Description |
|-------|-------------|
| `ci-passed` | All CI checks passed |
| `ci-failed` | CI failed (at least one check failed) |
| `review-submitted` | A review was submitted |
| `comment-added` | A new comment was posted |
| `merge-conflict-changed` | Mergeable state changed |
| `pr-merged` | PR was merged (terminal — always exits) |
| `pr-closed` | PR was closed (terminal — always exits) |

## Output format

One JSON object per line on stdout:

```json
{"timestamp":"2026-04-13T10:30:00Z","event":"ci-passed","summary":"All CI checks passed","details":{}}
```
