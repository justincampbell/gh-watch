# gh-watch

A `gh` CLI extension that watches GitHub resources for state changes. See README.md for usage, flags, event types, and output format.

## Architecture

- `main.go` — CLI entry point (cobra). Root command + `pr`, `commit`, `branch`, and `tag` subcommands.
- `internal/checks/` — Shared `CheckRun` type, CI summary helpers, and StatusContext mappers used by both PR and commit.
- `internal/pr/` — PR state snapshot types and GraphQL fetcher (via go-gh SDK). `search.go` backs `pr --search`: one GraphQL search per tick sharing the `prFields` fragment, refetching PRs that drop out while open so merges are reported.
- `internal/commit/` — Commit CI state snapshot types and GraphQL fetcher.
- `internal/branch/` — Branch head-commit snapshot types and GraphQL fetcher. Lists commits since the previous head via the REST compare API.
- `internal/tag/` — Tag list snapshot types and fetcher. Lists tags via GraphQL; for `--contains`, resolves ancestry via the REST compare API (`api.DefaultRESTClient`), memoized per tag SHA.
- `internal/events/` — Event types and diffing logic. `ci.go` has shared CI diff logic; `diff.go` adds PR-specific events (reviews, comments, merge conflicts, draft, auto-merge); `set_diff.go` diffs `--search` sets; `commit_diff.go` handles commit initial state; `branch_diff.go` emits `new-commit`; `tag_diff.go` emits `tag-created`.
- `internal/poller/` — Generic poller using Go generics (`Config[S any]`). Pluggable strategy interface (currently `FixedStrategy`). Wraps `Fetch` in `cenkalti/backoff/v5` to retry transient errors (default 30min budget per poll, so a watch outlives a GitHub incident). Applies the `--events` filter and an optional `Store` for `--resume`.
- `internal/state/` — JSON snapshot files for `--resume`, keyed by a hash of repo + subcommand + target.
- `internal/retry/` — Classifies API errors as transient (`IsTransient`): GitHub 5xx/429, secondary rate limits (403 + `Retry-After`, surfaced via `RetryAfter`), GraphQL `INTERNAL` errors returned with HTTP 200, network errors, timeouts, and malformed bodies.
- `internal/output/` — JSON to stdout.
- `plugin/` — Claude Code plugin: provides skills for monitoring PRs, commits, branches, and tags.

## Key design decisions

- Named `gh-watch` (not `gh-watch-pr`) to support future subcommands like `gh watch commit`, `gh watch branch`.
- Poller uses generics so it's reusable for future watch targets beyond PRs.
- Only aggregate CI events are emitted (`ci-passed`, `ci-failed`) — individual check transitions are noise. At most one per poll; `ci-failed` lists every failing check. A failure must appear on two consecutive polls of the same head SHA before it's reported (tracked via `CheckRun.FailureConfirmed`), because CI providers briefly post failures for superseded runs.
- Snapshots carry the last *settled* mergeable value (UNKNOWN carries the previous one forward), so the recompute after every base-branch merge is silent.
- Fetchers that depend on the previous snapshot (`--search` refetch, branch commit listing) get it through closure state in `main.go`, seeded from the `--resume` store on startup.
- Dependencies: `go-gh/v2` (GitHub API/auth), `cobra` (CLI), `cenkalti/backoff/v5` (retry). Auth is inherited from `gh` automatically.

## Development

```
go build -o ./gh-watch . && gh watch pr
```

## Testing

```
go test ./...
```

Tests live in `internal/events/` (diffing logic), `internal/poller/` (retry, filtering, resume), `internal/retry/` (transient-error classifier), `internal/checks/`, `internal/pr/`, and `internal/state/`.

## Releasing

Tag-triggered via `cli/gh-extension-precompile` which cross-compiles for all platforms:

```
git tag vX.Y.Z && git push origin vX.Y.Z
```
