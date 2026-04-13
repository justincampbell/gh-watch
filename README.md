# gh-watch

A [gh](https://cli.github.com/) CLI extension that watches GitHub resources for state changes.

## Installation

```
gh extension install justincampbell/gh-watch
```

## Usage

```
gh watch <command> [flags]
```

### Commands

#### `gh watch pr [<number>] [flags]`

Watch a pull request for state changes. If no PR number is given, the PR for the current branch is detected automatically.

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--interval` | Polling interval | `60s` |
| `--json` | JSON-only output (suppress human-friendly stderr) | `false` |
| `--exit-on` | Exit after a specific event type | |

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
gh watch pr 42 --exit-on any
```

JSON-only output for machine consumption:

```
gh watch pr 42 --json
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

A human-friendly summary is printed to **stderr** (unless `--json` is used):

```
[10:30:00] All CI checks passed
```

### Event Types

| Event | Description |
|-------|-------------|
| `any` | Exit on any event (for `--exit-on`) |
| `ci-status-changed` | A check run transitioned state |
| `ci-passed` | All required checks are passing |
| `ci-failed` | A required check has failed |
| `review-submitted` | A review was submitted |
| `comment-added` | A new comment was posted |
| `merge-conflict-changed` | The PR's mergeable state changed |
| `pr-merged` | The PR was merged (terminal) |
| `pr-closed` | The PR was closed (terminal) |

## Releasing

Push a version tag to trigger the release workflow, which cross-compiles binaries for all platforms via `cli/gh-extension-precompile`:

```
git tag v0.2.0
git push origin v0.2.0
```

Tags with hyphens (e.g. `v0.2.0-rc.1`) are published as prereleases.

## License

MIT
