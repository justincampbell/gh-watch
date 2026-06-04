package pr

import (
	"time"

	"github.com/justincampbell/gh-watch/internal/checks"
)

// State represents a snapshot of a PR's current state.
type State struct {
	Number    int
	Title     string
	Status    string // "open", "closed", "merged"
	Mergeable string // "MERGEABLE", "CONFLICTING", "UNKNOWN"

	InMergeQueue       bool   // true while the PR has a merge queue entry
	MergeQueueState    string // entry state: QUEUED, AWAITING_CHECKS, MERGEABLE, UNMERGEABLE, LOCKED; "" when not queued
	MergeQueuePosition int    // position in the queue; 0 when not queued

	CheckRuns []checks.CheckRun
	Reviews   []Review
	Comments  []Comment
}

type Review struct {
	Author string
	State  string // "APPROVED", "CHANGES_REQUESTED", "COMMENTED", "DISMISSED"
	Body   string
}

type Comment struct {
	ID        string
	Author    string
	Body      string
	CreatedAt time.Time
}
