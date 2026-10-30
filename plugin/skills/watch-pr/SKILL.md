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

**Wait until a queued PR resolves** (merges or gets kicked out of the merge queue):

- `command`: `gh watch pr $ARGUMENTS --exit-on pr-merged,merge-queue-removed`

**Watch the PR for the rest of the session** (CI + reviews + comments + merge state):

- `command`: `gh watch pr $ARGUMENTS`
- `description`: `PR $ARGUMENTS state changes`
- `persistent`: `true` (no timeout — stop with `TaskStop` or end of session)

**Watch a set of PRs defined by a search** (e.g. every open PR you have, or every PR with a given label). The query is re-run each poll, so new PRs join and merged ones leave without restarting:

- `command`: `gh watch pr --search "author:@me is:open" --events ready-for-review,auto-merge-enabled,merge-conflict-changed,ci-failed,pr-merged,pr-added --ignore-check "Lint*" --resume`
- `description`: `My open PRs`
- `persistent`: `true`

If no PR number is provided, the PR for the current branch is detected automatically. To watch a PR in another repo, prefix the command with `GH_REPO=owner/repo` (the `pr` subcommand takes a number, not a URL).

**Re-arming an expired monitor:** pass `--resume` every time you start a watch you may need to restart. A restarted `--resume` watch reports everything that changed while it wasn't running instead of a fresh `initial-state`, so nothing is lost when a monitor times out and you start it again with the same command.

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--interval <duration>` | Polling interval | `60s` |
| `--exit` | Exit after any state change | `false` |
| `--exit-on <events>` | Exit after specific event types (comma-separated) | |
| `--events <events>` | Only emit these event types; `initial-state`, terminal, and `--exit-on` events always come through | all |
| `--search <query>` | Watch every PR in the repo matching the query (no PR number) | |
| `--ignore-check <glob>` | Ignore noisy checks (repeatable; `*` matches across `/`) | |
| `--resume` | Report what changed since the last run of this same watch | `false` |

## Event types

| Event | Description |
|-------|-------------|
| `initial-state` | Snapshot of PR state at the moment watching started (always emitted first) |
| `ci-passed` | All CI checks passed |
| `ci-failed` | CI failed, confirmed on two consecutive polls of the same head commit (so it arrives one interval late, but brief false failures are dropped). `details.sha` is the head; `details.failed` lists every confirmed failure with its conclusion, `new: true` on the ones just confirmed |
| `review-submitted` | A review was submitted |
| `comment-added` | A new comment was posted |
| `comment-edited` | An existing comment's body changed (bots that re-review by editing their summary comment fire this, not `comment-added`). `details.url` links to the comment for the full body |
| `merge-conflict-changed` | Mergeable settled to a different value (MERGEABLE ↔ CONFLICTING) |
| `ready-for-review` | Draft marked ready for review |
| `converted-to-draft` | PR converted back to draft |
| `auto-merge-enabled` | Auto-merge armed |
| `auto-merge-disabled` | Auto-merge disarmed |
| `pr-added` | A PR started matching `--search` |
| `pr-removed` | An open PR stopped matching `--search` |
| `merge-queue-entered` | PR was added to the merge queue |
| `merge-queue-status-changed` | Merge queue entry changed state (e.g. `AWAITING_CHECKS` → `UNMERGEABLE`) |
| `merge-queue-removed` | PR left the merge queue without merging (e.g. kicked out on failure) |
| `pr-merged` | PR was merged (terminal — exits, except under `--search`) |
| `pr-closed` | PR was closed (terminal — exits, except under `--search`) |

Every PR event carries `details.number`, which is how you tell PRs apart under `--search`.

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
