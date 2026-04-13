# gh-watch

A `gh` CLI extension that watches GitHub resources for state changes. See README.md for usage, flags, event types, and output format.

## Architecture

- `main.go` — CLI entry point (cobra). Root command + `pr` subcommand + hidden `hook` subcommand.
- `internal/pr/` — PR state snapshot types and GraphQL fetcher (via go-gh SDK).
- `internal/events/` — Event types and `Diff(old, new)` state diffing logic. This is the core business logic.
- `internal/poller/` — Generic poller using Go generics (`Config[S any]`). Pluggable strategy interface (currently `FixedStrategy`).
- `internal/output/` — JSON to stdout, human-friendly to stderr.
- `internal/hook/` — Claude Code PostToolUse hook handler. Fail-silent by design (never breaks user workflow).
- `plugin/` — Claude Code plugin: provides a skill and a PostToolUse hook that auto-starts `gh watch pr` after `git push`.

## Key design decisions

- Named `gh-watch` (not `gh-watch-pr`) to support future subcommands like `gh watch commit`, `gh watch branch`.
- Poller uses generics so it's reusable for future watch targets beyond PRs.
- The `hook` subcommand is hidden from help — only called by the Claude Code plugin hook.
- Dependencies: only `go-gh/v2` (GitHub API/auth) and `cobra` (CLI). Auth is inherited from `gh` automatically.

## Development

```
go build -o "$(gh extension list | grep watch | awk '{print $NF}')/gh-watch" . && gh watch pr
```

## Testing

```
go test ./...
```

Tests live in `internal/events/diff_test.go` focused on the state diffing logic.

## Releasing

Tag-triggered via `cli/gh-extension-precompile` which cross-compiles for all platforms:

```
git tag vX.Y.Z && git push origin vX.Y.Z
```
