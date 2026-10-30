# gh-watch

A [gh](https://cli.github.com/) CLI extension that watches GitHub resources for state changes.

## Installation

```
gh extension install justincampbell/gh-watch
```

## Using with Claude Code

This repo ships a [Claude Code plugin](https://docs.claude.com/en/docs/claude-code/plugins) with skills that let Claude watch PRs, commits, branches, and tags on your behalf.

### Install

Add the marketplace, then install the plugin:

```
/plugin marketplace add justincampbell/gh-watch
/plugin install gh-watch@gh-watch
```

The `gh-watch` CLI extension must also be installed (see [Installation](#installation) above) — the skills shell out to `gh watch`.

### Skills

| Skill | When to use |
|-------|-------------|
| `/watch-pr` | Wait for a PR's CI, reviews, comments, or merge state to change |
| `/watch-commit` | Wait for CI checks to finish on a specific SHA |
| `/watch-branch` | Wait for new commits to land on a branch (e.g. `main` merges) |
| `/watch-tag` | Wait for a new tag, or for a tag that includes a specific commit |

Claude will invoke the right skill automatically when you ask things like *"watch PR 42 and let me know when CI passes"* or *"wait for the build on this commit"*. You can also invoke a skill explicitly by typing its slash command.

## Usage

```
gh watch <command> [flags]
```

### Commands

#### `gh watch pr [<number>] [flags]`

Watch a pull request for state changes. If no PR number is given, the PR for the current branch is detected automatically.

#### `gh watch pr --search <query> [flags]`

Watch every PR in the current repository matching a search query (anything `gh pr list --search` accepts, including `@me`). The query is re-run each interval with one GraphQL search, so new PRs join (`pr-added`) and PRs that stop matching leave (`pr-removed`). A PR that drops out of an `is:open` query by merging or closing reports `pr-merged` / `pr-closed` instead, and the watch keeps going. Every event carries `details.number`. Up to 100 PRs; `repo:`, `org:`, and `user:` qualifiers are rejected.

```
gh watch pr --search "author:@me is:open label:release"
```

| Flag | Description | Default |
|------|-------------|---------|
| `--search` | Watch every PR matching this query | |
| `--ignore-check` | Ignore checks whose name matches this glob (repeatable; `*` matches across `/`). Ignored checks never fire `ci-failed` and don't block `ci-passed`. Also on `commit`. | |

#### `gh watch commit <sha-or-url> [flags]`

Watch a commit for CI status changes. Exits automatically when all checks complete.

Accepts a bare SHA (uses the current repo) or a full GitHub commit URL:

```
gh watch commit abc1234
gh watch commit https://github.com/owner/repo/commit/abc1234
```

#### `gh watch branch <name> [flags]`

Watch a branch for new commits pushed to it. Every commit that lands between polls gets its own `new-commit` event, oldest first; if the branch was rewritten, only the new head is reported, with `"force_pushed": true`. When the headline names a PR (a squash's `(#123)` suffix or `Merge pull request #123`), the event includes `pr_number`.

| Flag | Description | Default |
|------|-------------|---------|
| `--skip-author` | Don't report commits by this login (repeatable; `@me` for yourself) | |

#### `gh watch tag [flags]`

Watch a repository's tags for newly created tags. Use `--match` to filter tag names by glob, and `--contains` to report only tags whose commit includes a given commit. In `--contains` mode, if a matching tag already exists when watching begins, it is reported immediately (with `"preexisting": true`) so `--exit` / `--exit-on tag-created` can fire.

```
gh watch tag --match 'v*'
gh watch tag --match 'v*' --contains abc1234
```

| Flag | Description | Default |
|------|-------------|---------|
| `--match` | Glob to filter tag names (`*`, `?`, `[…]`) | all tags |
| `--contains` | Only report tags whose commit contains this commit SHA | |

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--interval` | Polling interval | `60s` |
| `--exit` | Exit after any state change | `false` |
| `--exit-on` | Exit after specific event types (comma-separated) | |
| `--events` | Only emit these event types (comma-separated). `initial-state`, terminal events, and `--exit-on` types always come through. | all |
| `--resume` | Pick up where the last run of this same watch left off (see below) | `false` |

### Resuming

With `--resume`, each poll's snapshot is saved to `$XDG_STATE_HOME/gh-watch` (default `~/.local/state/gh-watch`), keyed by repository, subcommand, and target (PR number, search query, SHA, branch, or tag filters). The next `--resume` run of the same watch compares its first poll against that snapshot instead of emitting `initial-state`, so anything that changed in between is reported. Use this to re-arm a monitor that gets killed on a schedule without losing events. A terminal event (PR merged or closed, commit CI finished) clears the saved state.

It reports the net change between snapshots, not every step in between: a check that failed and then passed while nothing was watching shows up as passing. Branch watches replay every commit. Changing `--ignore-check` between runs can produce spurious CI events on the first poll.

### Examples

Watch PR #42 with default polling:

```
gh watch pr 42
```

Watch the current branch's PR, polling every 30 seconds:

```
gh watch pr --interval 30s
```

Wait for CI to pass, then exit:

```
gh watch pr 42 --exit-on ci-passed
```

Exit on any change:

```
gh watch pr 42 --exit
```

Watch a commit's CI from another repo:

```
gh watch commit https://github.com/owner/repo/commit/abc1234
```

Wait for a release tag that includes a merged commit, then exit:

```
gh watch tag --match 'v*' --contains abc1234 --exit-on tag-created
```

## Output Format

Each event is printed as a single line of JSON to **stdout**:

```json
{
  "timestamp": "2026-04-13T10:30:00Z",
  "event": "ci-passed",
  "summary": "All CI checks passed",
  "details": {}
}
```

### Event Types

The first event emitted is always `initial-state` — a snapshot of the resource at the moment watching began. Subsequent events are state changes.

| Event | Description |
|-------|-------------|
| `initial-state` | Snapshot of current state (always first, never triggers `--exit` or `--exit-on`) |
| `ci-passed` | All required checks are passing |
| `ci-failed` | A check failed. It's reported once the same failure shows up on two polls in a row for the same head commit, which filters out the brief false failures some CI providers post for superseded runs (so it arrives one interval after the failure). One event per poll: `sha` is the head commit, `name`/`conclusion`/`url` describe the first newly confirmed failure, and `failed` lists every confirmed failure with `new` marking the ones just confirmed |
| `review-submitted` | A review was submitted |
| `comment-added` | A new comment was posted (`id`, `author`, `body` trimmed to 200 chars, `url`) |
| `comment-edited` | An existing comment's body changed, e.g. a review bot updating its summary in place instead of posting a new one. Same details as `comment-added` |
| `merge-conflict-changed` | The PR's mergeable state settled to a different value (MERGEABLE ↔ CONFLICTING). The UNKNOWN blips GitHub shows while it recomputes after every merge to the base branch are ignored |
| `ready-for-review` | A draft PR was marked ready for review |
| `converted-to-draft` | The PR was converted back to a draft |
| `auto-merge-enabled` | Auto-merge was enabled (`merge_method`, `enabled_by`) |
| `auto-merge-disabled` | Auto-merge was disabled |
| `merge-queue-entered` | The PR was added to the merge queue |
| `merge-queue-status-changed` | The PR's merge queue entry changed state (e.g. `AWAITING_CHECKS` → `UNMERGEABLE`) |
| `merge-queue-removed` | The PR left the merge queue without merging (e.g. kicked out on failure) |
| `pr-merged` | The PR was merged (terminal, except under `--search`) |
| `pr-closed` | The PR was closed (terminal, except under `--search`) |
| `pr-added` | A PR started matching a `--search` query (details like `initial-state`) |
| `pr-removed` | An open PR stopped matching a `--search` query |
| `new-commit` | A new commit was pushed to a watched branch |
| `tag-created` | A new tag appeared (in `--contains` mode, only tags including the target commit; `"preexisting": true` if it already existed at watch start) |

## Errors

Transient GitHub failures are retried automatically with jittered exponential backoff, up to a 30-minute budget per poll — long enough to ride out a GitHub incident instead of aborting the watch. Treated as transient:

- HTTP 5xx and 429
- Secondary rate limits (403 with a `Retry-After` header, whose delay is honored)
- GraphQL `INTERNAL` errors, which GitHub returns with HTTP 200
- Network errors, DNS failures, and request timeouts
- Truncated or non-JSON response bodies

Retries and recovery are logged to **stderr** so the JSON event stream on stdout stays clean. Everything else — 401, 404, genuine 403 permission errors, invalid arguments — exits non-zero immediately.

## Development

Build and install from local source:

```
gh extension install .
go build -o ./gh-watch .
```

Run after making changes:

```
go build -o ./gh-watch . && gh watch pr
```

## Releasing

Push a version tag to trigger the release workflow, which cross-compiles binaries for all platforms via `cli/gh-extension-precompile`:

```
git tag v0.2.0
git push origin v0.2.0
```

Tags with hyphens (e.g. `v0.2.0-rc.1`) are published as prereleases.

## License

MIT
