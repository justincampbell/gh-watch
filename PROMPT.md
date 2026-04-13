# gh-watch-pr

Build a `gh` CLI extension in Go that monitors a pull request and reports state changes.

## What it does

`gh watch-pr [<number>]` watches a single PR in the current repo, polling for changes and printing updates as they occur. If no number is given, detect the PR for the current branch. It is designed to run as a background process (e.g. from Claude Code or a CI pipeline) and emit structured, machine-readable output alongside human-friendly messages.

### State changes to detect

- **CI status changed** — a check run or status context transitions (e.g. pending -> success, pending -> failure). Report which job and the new state.
- **All CI succeeded** — all required checks are now passing.
- **Any CI failed** — a required check has failed. Include the job name and a link.
- **Review submitted** — a review was left (approved, changes requested, or commented). Include reviewer and verdict.
- **Comment added** — a new issue comment or review comment appeared. Include author and a preview.
- **Merge conflict status changed** — the PR's `mergeable` state changed (e.g. became conflicted or was resolved).
- **PR merged or closed** — terminal states; print and exit.

### Output format

Each event should be printed as a single line of JSON to stdout, with at minimum:
- `timestamp` (ISO 8601)
- `event` (string enum of the above categories)
- `summary` (human-readable one-liner)
- `details` (object with event-specific fields)

Also print a human-friendly summary to stderr so it's readable in a terminal.

### Polling

Start with a simple 60-second polling interval. Structure the code so the polling strategy is pluggable — a future version will adjust the interval based on context (e.g. poll faster when CI is running, slower when waiting for review overnight).

### CLI interface

```
gh watch-pr [<number>] [flags]

Flags:
  --interval <duration>   Polling interval (default: 60s)
  --json                  JSON-only output (suppress stderr human-friendly lines)
  --exit-on <event>       Exit after a specific event type (e.g. --exit-on ci-passed)
```

Use `cobra` for CLI parsing and the `go-gh` SDK (`github.com/cli/go-gh/v2`) for GitHub API access and auth (inherits `gh` credentials automatically).

## Project setup

- **Go module**: `github.com/justincampbell/gh-watch-pr`
- **README.md**: Describe the tool, installation (`gh extension install justincampbell/gh-watch-pr`), usage examples, and output format.
- **LICENSE**: MIT, copyright Justin Campbell.
- **CI** (`.github/workflows/ci.yml`): Run `go build`, `go test`, and `go vet` on push and PR. Test on latest Go version, ubuntu-latest.
- **Release** (`.github/workflows/release.yml`): Use `cli/gh-extension-precompile` to cross-compile and attach binaries on tag push. This is the standard release mechanism for precompiled gh extensions — research how it works and set it up correctly.
- **.goreleaser.yml**: Only if needed by `gh-extension-precompile`; research whether it's required or if the action handles it.

## Architecture guidance

- Keep the poller, event detection, and output formatting as separate concerns so they can evolve independently.
- Store "last known state" in memory (no persistence needed) to diff against each poll.
- Use the GraphQL API where it reduces round-trips (e.g. fetching PR status + reviews + comments in one query), REST where simpler.
- Write tests for the state-diffing logic. The poller and API calls can be tested via interfaces.

## Research

Look at existing well-implemented gh extensions in Go for patterns:
- How they use `go-gh` for API access
- How they structure `main.go` vs internal packages
- How `cli/gh-extension-precompile` works for releases

Make implementation decisions based on what you find — this prompt describes the desired behavior, not a rigid architecture.
