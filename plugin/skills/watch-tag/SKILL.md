---
name: watch-tag
description: Watch a GitHub repository for new tags using the gh-watch extension. Use when the user wants to be notified when a tag is created, when a release is cut, or when a tag that includes a specific commit appears (e.g. "tell me when my merge ships in a release").
argument-hint: "[--match <glob>] [--contains <sha>]"
---

# Watch Tags

Monitor a repository's tags using the `gh watch` CLI extension. It polls GitHub and emits one JSON event per line to stdout when a new tag appears, which makes it a natural fit for the **Monitor tool** — each event arrives in the conversation as a notification.

Two modes:

- **All new tags** — `--match '<glob>'` filters tag names (e.g. `v*`). Every new matching tag is reported.
- **Tags containing a commit** — `--contains <sha>` reports only tags whose commit includes `<sha>` in its history. This answers *"which release includes my merged commit?"* If such a tag already exists when watching starts, it is reported immediately with `"preexisting": true`.

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

Run with the **Monitor tool** (not `Bash` with `run_in_background`). The first notification is the `initial-state` snapshot; subsequent notifications are `tag-created` events.

**Wait for the release tag that includes a merged commit, then exit:**

- `command`: `gh watch tag --match 'v*' --contains <sha> --exit-on tag-created`
- `description`: `Release tag containing <short-sha>`
- `persistent`: `false`
- `timeout_ms`: `7200000` (2 hours — raise/lower based on how long releases take)

**Watch for every new release tag for the rest of the session:**

- `command`: `gh watch tag --match 'v*'`
- `description`: `New v* tags`
- `persistent`: `true` (no timeout — stop with `TaskStop` or end of session)

## Composing with `gh watch commit`

A `tag-created` event tells you the tag *exists* — not that its CI or deploy finished. To follow what happens after the tag, take the `sha` from the `tag-created` event and hand off to the commit watcher:

```
gh watch commit <tag-sha> --exit-on ci-passed,ci-failed
```

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--match <glob>` | Filter tag names (`*`, `?`, `[…]`) | all tags |
| `--contains <sha>` | Only report tags whose commit contains this commit | |
| `--interval <duration>` | Polling interval | `60s` |
| `--exit` | Exit after any state change | `false` |
| `--exit-on <events>` | Exit after specific event types (comma-separated) | |

## Event types

| Event | Description |
|-------|-------------|
| `initial-state` | Snapshot of matching tags at the moment watching started (always emitted first) |
| `tag-created` | A new tag appeared. In `--contains` mode, only tags including the target commit are reported; `"preexisting": true` marks a tag that already existed at watch start |

## Output format

One JSON object per line on stdout. The **first line is always `initial-state`** — a snapshot of the matching tags at the moment watching began:

```json
{"timestamp":"...","event":"initial-state","summary":"3 tags matching v* (latest: v1.2.0)","details":{"tag_count":3,"match":"v*","latest_tag":"v1.2.0","latest_sha":"abc1234..."}}
```

Subsequent lines are `tag-created` events:

```json
{"timestamp":"...","event":"tag-created","summary":"New tag v1.3.0 at def5678: Release 1.3.0","details":{"tag":"v1.3.0","sha":"def5678...","message_headline":"Release 1.3.0"}}
```

In `--contains` mode, a tag that already includes the target commit at watch start is reported immediately:

```json
{"timestamp":"...","event":"tag-created","summary":"New tag v1.2.0 at abc1234: Release 1.2.0","details":{"tag":"v1.2.0","sha":"abc1234...","message_headline":"Release 1.2.0","contains":"<target-sha>","preexisting":true}}
```

## Interpreting the output

**The first notification is always `initial-state`.** Report it to the user immediately — how many matching tags exist and the latest one. Don't just say "watching in background."

**Subsequent `tag-created` notifications** carry the tag name, its commit SHA, and the commit message — surface each one. In `--contains` mode, a `tag-created` means *the commit shipped in that tag*; note whether it was `preexisting` (already released before watching) or newly created during the watch.
