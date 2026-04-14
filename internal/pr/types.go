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
