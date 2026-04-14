---
name: watch-commit
description: Watch a GitHub commit for CI status changes using the gh-watch extension. Use when the user wants to monitor a commit's CI checks, wait for a build to finish, or track CI progress on a specific SHA.
argument-hint: "<sha-or-url>"
---

# Watch a Commit

Monitor a commit's CI checks for state changes using the `gh watch` CLI extension. Run it as a background process — it polls GitHub and emits one JSON event per line to stdout. Exits automatically when all checks complete.

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

Run as a **background task** (Bash with `run_in_background: true`). The command exits automatically when CI finishes (all checks passed or any failed).

**Watch a commit by SHA (current repo):**

```
gh watch commit $ARGUMENTS
```

**Watch a commit by GitHub URL:**

```
gh watch commit https://github.com/owner/repo/commit/abc1234
```

You can also use `--exit-on` to exit on a specific event:

**Exit only if CI passes:**

```
gh watch commit $ARGUMENTS --exit-on ci-passed
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
| `ci-passed` | All CI checks passed (terminal — always exits) |
| `ci-failed` | CI failed (at least one check failed) (terminal — always exits) |

## Output format

One JSON object per line on stdout:

```json
{"timestamp":"2026-04-13T10:30:00Z","event":"ci-passed","summary":"All CI checks passed","details":{}}
```
