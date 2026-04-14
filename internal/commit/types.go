package commit

import "github.com/justincampbell/gh-watch/internal/checks"

// State represents a snapshot of a commit's CI status.
type State struct {
	SHA       string
	CheckRuns []checks.CheckRun
}
