---
name: github-watch-pull-request
description: Watch a GitHub pull request for CI status, reviews, comments, merge conflicts, and terminal states using the gh-watch extension. Use when the user wants to monitor a PR, wait for CI, or track PR progress.
argument-hint: "[PR number]"
---

# Watch a Pull Request

Monitor a pull request for state changes using the `gh watch` CLI extension. This runs as a background process, polling GitHub and emitting structured JSON events to stdout and human-friendly summaries to stderr.

## Prerequisites

Check if the extension is installed:

```
gh watch --help
```

If not installed, install it:

```
gh extension install justincampbell/gh-watch
```

## Usage

Watch a specific PR by number:

```
gh watch pr $ARGUMENTS
```

If no PR number is provided, detect the PR for the current branch:

```
gh watch pr
```

### Common patterns

**Wait for CI to pass, then exit:**

```
gh watch pr $ARGUMENTS --exit-on ci-passed
```

**Wait for CI to complete (pass or fail), then exit:**

```
gh watch pr $ARGUMENTS --exit-on ci-passed --exit-on ci-failed
```

**Poll faster (every 30 seconds):**

```
gh watch pr $ARGUMENTS --interval 30s
```

**JSON-only output (suppress human-friendly stderr):**

```
gh watch pr $ARGUMENTS --json
```

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--interval <duration>` | Polling interval | `60s` |
| `--json` | JSON-only output (suppress stderr) | `false` |
| `--exit-on <event>` | Exit after a specific event type | |

## Event types

These are the `--exit-on` values and the events emitted as JSON:

| Event | Description |
|-------|-------------|
| `ci-status-changed` | A check run transitioned state |
| `ci-passed` | All CI checks are passing |
| `ci-failed` | A required check has failed |
| `review-submitted` | A review was submitted |
| `comment-added` | A new comment was posted |
| `merge-conflict-changed` | The PR's mergeable state changed |
| `pr-merged` | The PR was merged (terminal — auto exits) |
| `pr-closed` | The PR was closed (terminal — auto exits) |

## Output format

Each event is a single line of JSON on stdout:

```json
{"timestamp":"2026-04-13T10:30:00Z","event":"ci-passed","summary":"All CI checks passed","details":{}}
```

The process automatically exits on terminal events (`pr-merged`, `pr-closed`).

## Recommended workflow

1. Start watching in the background or in a separate terminal
2. Use `--exit-on ci-passed` when you only need to wait for CI
3. Use `--json` when parsing output programmatically
4. The default 60s poll interval is respectful of API rate limits — use shorter intervals only when actively waiting
