# gh-watch

A `gh` CLI extension that watches GitHub resources for state changes. See README.md for usage, flags, event types, and output format.

## Architecture

- `main.go` — CLI entry point (cobra). Root command + `pr` and `commit` subcommands.
- `internal/checks/` — Shared `CheckRun` type, CI summary helpers, and StatusContext mappers used by both PR and commit.
- `internal/pr/` — PR state snapshot types and GraphQL fetcher (via go-gh SDK).
- `internal/commit/` — Commit CI state snapshot types and GraphQL fetcher.
- `internal/events/` — Event types and diffing logic. `ci.go` has shared CI diff logic; `diff.go` adds PR-specific events (reviews, comments, merge conflicts); `commit_diff.go` handles commit initial state.
- `internal/poller/` — Generic poller using Go generics (`Config[S any]`). Pluggable strategy interface (currently `FixedStrategy`).
- `internal/output/` — JSON to stdout.
- `plugin/` — Claude Code plugin: provides skills for monitoring PRs and commits.

## Key design decisions

- Named `gh-watch` (not `gh-watch-pr`) to support future subcommands like `gh watch commit`, `gh watch branch`.
- Poller uses generics so it's reusable for future watch targets beyond PRs.
- Only aggregate CI events are emitted (`ci-passed`, `ci-failed`) — individual check transitions are noise.
- Dependencies: only `go-gh/v2` (GitHub API/auth) and `cobra` (CLI). Auth is inherited from `gh` automatically.

## Development

```
go build -o ./gh-watch . && gh watch pr
```

## Testing

```
go test ./...
```

Tests live in `internal/events/diff_test.go` and `internal/events/commit_diff_test.go` focused on the state diffing logic.

## Releasing

Tag-triggered via `cli/gh-extension-precompile` which cross-compiles for all platforms:

```
git tag vX.Y.Z && git push origin vX.Y.Z
```
