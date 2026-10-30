package events

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// EventType identifies the kind of state change detected.
type EventType string

const (
	AnyEvent                EventType = "any"
	InitialState            EventType = "initial-state"
	CIAllPassed             EventType = "ci-passed"
	CIFailed                EventType = "ci-failed"
	ReviewSubmitted         EventType = "review-submitted"
	CommentAdded            EventType = "comment-added"
	CommentEdited           EventType = "comment-edited"
	MergeConflictChanged    EventType = "merge-conflict-changed"
	ReadyForReview          EventType = "ready-for-review"
	ConvertedToDraft        EventType = "converted-to-draft"
	AutoMergeEnabled        EventType = "auto-merge-enabled"
	AutoMergeDisabled       EventType = "auto-merge-disabled"
	MergeQueueEntered       EventType = "merge-queue-entered"
	MergeQueueStatusChanged EventType = "merge-queue-status-changed"
	MergeQueueRemoved       EventType = "merge-queue-removed"
	PRMerged                EventType = "pr-merged"
	PRClosed                EventType = "pr-closed"
	PRAdded                 EventType = "pr-added"
	PRRemoved               EventType = "pr-removed"
	NewCommit               EventType = "new-commit"
	BranchDeleted           EventType = "branch-deleted"
	TagCreated              EventType = "tag-created"
)

// Selectable lists the event types accepted by --events.
var Selectable = []EventType{
	CIAllPassed, CIFailed, ReviewSubmitted, CommentAdded, CommentEdited, MergeConflictChanged,
	ReadyForReview, ConvertedToDraft, AutoMergeEnabled, AutoMergeDisabled,
	MergeQueueEntered, MergeQueueStatusChanged, MergeQueueRemoved,
	PRMerged, PRClosed, PRAdded, PRRemoved, NewCommit, TagCreated,
}

// ParseSelection validates event type names given to --events.
func ParseSelection(names []string) ([]EventType, error) {
	var out []EventType
	for _, n := range names {
		t := EventType(n)
		if !slices.Contains(Selectable, t) {
			valid := make([]string, len(Selectable))
			for i, s := range Selectable {
				valid[i] = string(s)
			}
			return nil, fmt.Errorf("unknown event type %q (valid: %s)", n, strings.Join(valid, ", "))
		}
		out = append(out, t)
	}
	return out, nil
}

// Event represents a detected state change.
type Event struct {
	Timestamp time.Time              `json:"timestamp"`
	Event     EventType              `json:"event"`
	Summary   string                 `json:"summary"`
	Details   map[string]interface{} `json:"details"`
}
