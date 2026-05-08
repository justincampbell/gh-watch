---
name: watch-commit
description: Watch a GitHub commit for CI status changes using the gh-watch extension. Use when the user wants to monitor a commit's CI checks, wait for a build to finish, or track CI progress on a specific SHA.
argument-hint: "<sha-or-url>"
---

# Watch a Commit

Monitor a commit's CI checks for state changes using the `gh watch` CLI extension. It polls GitHub and emits one JSON event per line to stdout, which makes it a natural fit for the **Monitor tool** — each event arrives in the conversation as a notification. Exits automatically when all checks complete.

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

Run with the **Monitor tool** (not `Bash` with `run_in_background`). The first notification is the `initial-state` snapshot; the next is the terminal `ci-passed` or `ci-failed`, after which gh-watch exits.

**Watch a commit by SHA (current repo):**

- `command`: `gh watch commit $ARGUMENTS`
- `description`: `Commit $ARGUMENTS CI status`
- `persistent`: `false`
- `timeout_ms`: `1800000` (30 min — raise for slow CI)

**Watch a commit by GitHub URL:**

- `command`: `gh watch commit https://github.com/owner/repo/commit/abc1234`

**Exit only if CI passes** (use `--exit-on` to ignore failures):

- `command`: `gh watch commit $ARGUMENTS --exit-on ci-passed`

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--interval <duration>` | Polling interval | `60s` |
| `--exit` | Exit after any state change | `false` |
| `--exit-on <events>` | Exit after specific event types (comma-separated) | |

## Event types

| Event | Description |
|-------|-------------|
| `initial-state` | Snapshot of commit CI state at the moment watching started (always emitted first) |
| `ci-passed` | All CI checks passed (terminal — always exits) |
| `ci-failed` | CI failed (at least one check failed) (terminal — always exits) |

## Output format

One JSON object per line on stdout. The **first line is always `initial-state`** — a snapshot of the commit's CI at the moment watching began:

```json
{"timestamp":"...","event":"initial-state","summary":"Commit abc1234: Message — CI: 3/5 passed, 2 pending","details":{"sha":"abc1234...","checks":5,"passed":3,"failed":0,"pending":2}}
```

Subsequent lines are change events:

```json
{"timestamp":"...","event":"ci-passed","summary":"All CI checks passed","details":{}}
```

## Interpreting the output

**The first notification is always `initial-state`.** Report it to the user immediately — don't just say "watching in background." Tell them how many checks passed/pending/failed.

**Use `initial-state` to decide next steps:**
- If CI already passed → no need to wait; stop the monitor with `TaskStop` and act on the result
- If CI already failed → investigate the failure; stop the monitor with `TaskStop`
