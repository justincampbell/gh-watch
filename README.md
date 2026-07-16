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

#### `gh watch commit <sha-or-url> [flags]`

Watch a commit for CI status changes. Exits automatically when all checks complete.

Accepts a bare SHA (uses the current repo) or a full GitHub commit URL:

```
gh watch commit abc1234
gh watch commit https://github.com/owner/repo/commit/abc1234
```

#### `gh watch branch <name> [flags]`

Watch a branch for new commits pushed to it.

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
| `ci-failed` | A required check has failed |
| `review-submitted` | A review was submitted |
| `comment-added` | A new comment was posted |
| `merge-conflict-changed` | The PR's mergeable state changed |
| `merge-queue-entered` | The PR was added to the merge queue |
| `merge-queue-status-changed` | The PR's merge queue entry changed state (e.g. `AWAITING_CHECKS` → `UNMERGEABLE`) |
| `merge-queue-removed` | The PR left the merge queue without merging (e.g. kicked out on failure) |
| `pr-merged` | The PR was merged (terminal) |
| `pr-closed` | The PR was closed (terminal) |
| `new-commit` | A new commit was pushed to a watched branch |
| `tag-created` | A new tag appeared (in `--contains` mode, only tags including the target commit; `"preexisting": true` if it already existed at watch start) |

## Errors

Transient GitHub failures (HTTP 5xx, 429, network timeouts) are retried automatically with exponential backoff, up to a 5-minute budget per poll. Retry attempts are logged to **stderr** so the JSON event stream on stdout stays clean. Non-transient errors (4xx, auth failures) exit immediately.

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
