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

**Exit on any change (most common):**

```
gh watch pr $ARGUMENTS --exit
```

**Poll faster (every 30 seconds):**

```
gh watch pr $ARGUMENTS --exit --interval 30s
```

**JSON-only output (suppress human-friendly stderr):**

```
gh watch pr $ARGUMENTS --exit --json
```

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--interval <duration>` | Polling interval | `60s` |
| `--json` | JSON-only output (suppress stderr) | `false` |
| `--exit` | Exit after any state change | `false` |

## Output format

Each event is a single line of JSON on stdout:

```json
{"timestamp":"2026-04-13T10:30:00Z","event":"ci-passed","summary":"All CI checks passed","details":{}}
```

The process automatically exits on terminal events (`pr-merged`, `pr-closed`).

## Recommended workflow

1. Default to `--exit` — it exits after the first state change, which is usually what you want
2. Use `--json` when parsing output programmatically
3. The default 60s poll interval is respectful of API rate limits — use shorter intervals only when actively waiting
