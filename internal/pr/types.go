package pr

import "time"

// State represents a snapshot of a PR's current state.
type State struct {
	Number    int
	Title     string
	Status    string // "open", "closed", "merged"
	Mergeable string // "MERGEABLE", "CONFLICTING", "UNKNOWN"

	CheckRuns []CheckRun
	Reviews   []Review
	Comments  []Comment

	FetchedAt time.Time
}

type CheckRun struct {
	Name       string
	Status     string // "QUEUED", "IN_PROGRESS", "COMPLETED"
	Conclusion string // "SUCCESS", "FAILURE", "NEUTRAL", "CANCELLED", "TIMED_OUT", "ACTION_REQUIRED", ""
	URL        string
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
