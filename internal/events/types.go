package events

import "time"

// EventType identifies the kind of state change detected.
type EventType string

const (
	CIStatusChanged      EventType = "ci-status-changed"
	CIAllPassed          EventType = "ci-passed"
	CIFailed             EventType = "ci-failed"
	ReviewSubmitted      EventType = "review-submitted"
	CommentAdded         EventType = "comment-added"
	MergeConflictChanged EventType = "merge-conflict-changed"
	PRMerged             EventType = "pr-merged"
	PRClosed             EventType = "pr-closed"
)

// Event represents a detected state change.
type Event struct {
	Timestamp time.Time              `json:"timestamp"`
	Event     EventType              `json:"event"`
	Summary   string                 `json:"summary"`
	Details   map[string]interface{} `json:"details"`
}
