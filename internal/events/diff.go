package events

import (
	"fmt"
	"strings"
	"time"

	"github.com/justincampbell/gh-watch/internal/checks"
	"github.com/justincampbell/gh-watch/internal/pr"
)

// Diff compares old and new PR states and returns detected events.
// If old is nil, this is the initial fetch and only terminal states are reported.
func Diff(old, new *pr.State) []Event {
	now := time.Now()
	var events []Event

	// Terminal states
	if new.Status == "merged" && (old == nil || old.Status != "merged") {
		events = append(events, Event{
			Timestamp: now,
			Event:     PRMerged,
			Summary:   fmt.Sprintf("PR #%d has been merged", new.Number),
			Details:   map[string]interface{}{"number": new.Number},
		})
		return events
	}
	if new.Status == "closed" && (old == nil || old.Status != "closed") {
		events = append(events, Event{
			Timestamp: now,
			Event:     PRClosed,
			Summary:   fmt.Sprintf("PR #%d has been closed", new.Number),
			Details:   map[string]interface{}{"number": new.Number},
		})
		return events
	}

	if old == nil {
		events = append(events, initialStateEvent(new, now))
		return events
	}

	// CI changes
	events = append(events, diffCI(old.CheckRuns, new.CheckRuns, now)...)

	// Review changes
	if len(new.Reviews) > len(old.Reviews) {
		for _, review := range new.Reviews[len(old.Reviews):] {
			events = append(events, Event{
				Timestamp: now,
				Event:     ReviewSubmitted,
				Summary:   fmt.Sprintf("Review from %s: %s", review.Author, strings.ToLower(review.State)),
				Details: map[string]interface{}{
					"author": review.Author,
					"state":  review.State,
					"body":   truncate(review.Body, 200),
				},
			})
		}
	}

	// Comment changes
	if len(new.Comments) > len(old.Comments) {
		for _, comment := range new.Comments[len(old.Comments):] {
			events = append(events, Event{
				Timestamp: now,
				Event:     CommentAdded,
				Summary:   fmt.Sprintf("Comment from %s: %s", comment.Author, truncate(comment.Body, 80)),
				Details: map[string]interface{}{
					"author": comment.Author,
					"body":   truncate(comment.Body, 200),
				},
			})
		}
	}

	// Mergeable state change
	if old.Mergeable != new.Mergeable && new.Mergeable != "UNKNOWN" && old.Mergeable != "UNKNOWN" {
		summary := fmt.Sprintf("Merge conflict status: %s → %s", old.Mergeable, new.Mergeable)
		events = append(events, Event{
			Timestamp: now,
			Event:     MergeConflictChanged,
			Summary:   summary,
			Details: map[string]interface{}{
				"old": old.Mergeable,
				"new": new.Mergeable,
			},
		})
	}

	return events
}

func initialStateEvent(state *pr.State, now time.Time) Event {
	var parts []string

	s := checks.Summarize(state.CheckRuns)
	if s.Total > 0 {
		if s.Pending > 0 {
			parts = append(parts, fmt.Sprintf("CI: %d/%d passed, %d pending", s.Passed, s.Total, s.Pending))
		} else if s.Failed > 0 {
			parts = append(parts, fmt.Sprintf("CI: %d/%d passed, %d failed", s.Passed, s.Total, s.Failed))
		} else {
			parts = append(parts, fmt.Sprintf("CI: all %d checks passed", s.Total))
		}
	}

	// Reviews summary
	if len(state.Reviews) > 0 {
		approved, changesRequested := 0, 0
		for _, r := range state.Reviews {
			switch r.State {
			case "APPROVED":
				approved++
			case "CHANGES_REQUESTED":
				changesRequested++
			}
		}
		if changesRequested > 0 {
			parts = append(parts, fmt.Sprintf("%d reviews (%d approved, %d changes requested)", len(state.Reviews), approved, changesRequested))
		} else if approved > 0 {
			parts = append(parts, fmt.Sprintf("%d reviews (%d approved)", len(state.Reviews), approved))
		} else {
			parts = append(parts, fmt.Sprintf("%d reviews", len(state.Reviews)))
		}
	}

	// Mergeable
	if state.Mergeable == "CONFLICTING" {
		parts = append(parts, "has merge conflicts")
	}

	summary := fmt.Sprintf("PR #%d: %s", state.Number, state.Title)
	if len(parts) > 0 {
		summary += " — " + strings.Join(parts, ", ")
	}

	return Event{
		Timestamp: now,
		Event:     InitialState,
		Summary:   summary,
		Details: map[string]interface{}{
			"number":    state.Number,
			"title":     state.Title,
			"status":    state.Status,
			"mergeable": state.Mergeable,
			"checks":    s.Total,
			"passed":    s.Passed,
			"failed":    s.Failed,
			"pending":   s.Pending,
			"reviews":   len(state.Reviews),
			"comments":  len(state.Comments),
		},
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
