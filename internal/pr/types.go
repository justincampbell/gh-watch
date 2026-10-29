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
	IsDraft   bool
	Mergeable string // "MERGEABLE", "CONFLICTING", "UNKNOWN"

	InMergeQueue       bool   // true while the PR has a merge queue entry
	MergeQueueState    string // entry state: QUEUED, AWAITING_CHECKS, MERGEABLE, UNMERGEABLE, LOCKED; "" when not queued
	MergeQueuePosition int    // position in the queue; 0 when not queued

	AutoMerge *AutoMerge // nil when auto-merge is not enabled

	HeadSHA   string // commit the CheckRuns belong to
	CheckRuns []checks.CheckRun
	Reviews   []Review
	Comments  []Comment
}

type AutoMerge struct {
	MergeMethod string
	EnabledBy   string
}

type Review struct {
	ID     string
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

// SettleMergeable carries prev's mergeable value forward while GitHub is
// recomputing it, so the snapshot always holds the last settled value.
func (s *State) SettleMergeable(prev *State) {
	if s.Mergeable == "UNKNOWN" && prev != nil {
		s.Mergeable = prev.Mergeable
	}
}
