# gh-watch

A [gh](https://cli.github.com/) CLI extension that watches GitHub resources for state changes.

## Installation

```
gh extension install justincampbell/gh-watch
```

## Using with Claude Code

This repo ships a [Claude Code plugin](https://docs.claude.com/en/docs/claude-code/plugins) with skills that let Claude watch PRs, commits, and branches on your behalf.

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
| `pr-merged` | The PR was merged (terminal) |
| `pr-closed` | The PR was closed (terminal) |

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
