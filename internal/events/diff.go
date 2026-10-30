package events

import (
	"fmt"
	"strings"
	"time"

	"github.com/justincampbell/gh-watch/internal/checks"
	"github.com/justincampbell/gh-watch/internal/pr"
)

// Diff compares old and new PR states and returns detected events, each
// tagged with the PR number. If old is nil, this is the initial fetch.
// new's mergeable value is settled against old in place.
func Diff(old, new *pr.State) []Event {
	new.SettleMergeable(old)
	events := diffPR(old, new)
	for _, e := range events {
		e.Details["number"] = new.Number
	}
	return events
}

func diffPR(old, new *pr.State) []Event {
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
		checks.ConfirmFailures(new.CheckRuns)
		events = append(events, initialStateEvent(new, now))
		return events
	}

	// CI changes
	events = append(events, diffCI(old.HeadSHA, old.CheckRuns, new.HeadSHA, new.CheckRuns, now)...)

	events = append(events, diffDraft(old, new, now)...)
	events = append(events, diffAutoMerge(old, new, now)...)

	seenReviews := make(map[string]bool, len(old.Reviews))
	for _, r := range old.Reviews {
		seenReviews[r.ID] = true
	}
	for _, review := range new.Reviews {
		if !seenReviews[review.ID] {
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

	oldBodies := make(map[string]string, len(old.Comments))
	for _, c := range old.Comments {
		oldBodies[c.ID] = c.Body
	}
	for _, comment := range new.Comments {
		oldBody, seen := oldBodies[comment.ID]
		details := map[string]interface{}{
			"id":     comment.ID,
			"author": comment.Author,
			"body":   truncate(comment.Body, 200),
			"url":    comment.URL,
		}
		switch {
		case !seen:
			events = append(events, Event{
				Timestamp: now,
				Event:     CommentAdded,
				Summary:   fmt.Sprintf("Comment from %s: %s", comment.Author, truncate(comment.Body, 80)),
				Details:   details,
			})
		case oldBody != comment.Body:
			events = append(events, Event{
				Timestamp: now,
				Event:     CommentEdited,
				Summary:   fmt.Sprintf("Comment from %s edited: %s", comment.Author, truncate(comment.Body, 80)),
				Details:   details,
			})
		}
	}

	// Values are settled, so UNKNOWN here means GitHub hasn't computed one since the watch began.
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

	// Merge queue changes. Reached only when the PR is still open — the merged/closed
	// early returns above ensure a PR that leaves the queue by merging reports pr-merged,
	// not merge-queue-removed.
	events = append(events, diffMergeQueue(old, new, now)...)

	return events
}

func diffDraft(old, new *pr.State, now time.Time) []Event {
	switch {
	case old.IsDraft && !new.IsDraft:
		return []Event{{
			Timestamp: now,
			Event:     ReadyForReview,
			Summary:   fmt.Sprintf("PR #%d marked ready for review", new.Number),
			Details:   map[string]interface{}{},
		}}
	case !old.IsDraft && new.IsDraft:
		return []Event{{
			Timestamp: now,
			Event:     ConvertedToDraft,
			Summary:   fmt.Sprintf("PR #%d converted to draft", new.Number),
			Details:   map[string]interface{}{},
		}}
	}
	return nil
}

func diffAutoMerge(old, new *pr.State, now time.Time) []Event {
	switch {
	case old.AutoMerge == nil && new.AutoMerge != nil:
		return []Event{{
			Timestamp: now,
			Event:     AutoMergeEnabled,
			Summary:   fmt.Sprintf("Auto-merge (%s) enabled by %s", strings.ToLower(new.AutoMerge.MergeMethod), new.AutoMerge.EnabledBy),
			Details: map[string]interface{}{
				"merge_method": new.AutoMerge.MergeMethod,
				"enabled_by":   new.AutoMerge.EnabledBy,
			},
		}}
	case old.AutoMerge != nil && new.AutoMerge == nil:
		return []Event{{
			Timestamp: now,
			Event:     AutoMergeDisabled,
			Summary:   "Auto-merge disabled",
			Details:   map[string]interface{}{},
		}}
	}
	return nil
}

// diffMergeQueue detects merge queue lifecycle transitions between two PR states.
func diffMergeQueue(old, new *pr.State, now time.Time) []Event {
	switch {
	case !old.InMergeQueue && new.InMergeQueue:
		return []Event{{
			Timestamp: now,
			Event:     MergeQueueEntered,
			Summary:   fmt.Sprintf("Entered merge queue (position %d)", new.MergeQueuePosition),
			Details: map[string]interface{}{
				"state":    new.MergeQueueState,
				"position": new.MergeQueuePosition,
			},
		}}
	case old.InMergeQueue && !new.InMergeQueue:
		return []Event{{
			Timestamp: now,
			Event:     MergeQueueRemoved,
			Summary:   fmt.Sprintf("Removed from merge queue (was %s)", old.MergeQueueState),
			Details: map[string]interface{}{
				"old_state": old.MergeQueueState,
			},
		}}
	case old.InMergeQueue && new.InMergeQueue && old.MergeQueueState != new.MergeQueueState:
		return []Event{{
			Timestamp: now,
			Event:     MergeQueueStatusChanged,
			Summary:   fmt.Sprintf("Merge queue status: %s → %s", old.MergeQueueState, new.MergeQueueState),
			Details: map[string]interface{}{
				"old":      old.MergeQueueState,
				"new":      new.MergeQueueState,
				"position": new.MergeQueuePosition,
			},
		}}
	}
	return nil
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

	if state.IsDraft {
		parts = append(parts, "draft")
	}

	if state.Mergeable == "CONFLICTING" {
		parts = append(parts, "has merge conflicts")
	}

	if state.AutoMerge != nil {
		parts = append(parts, "auto-merge enabled")
	}

	// Merge queue
	if state.InMergeQueue {
		parts = append(parts, fmt.Sprintf("in merge queue (position %d)", state.MergeQueuePosition))
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
			"number":            state.Number,
			"title":             state.Title,
			"status":            state.Status,
			"draft":             state.IsDraft,
			"auto_merge":        state.AutoMerge != nil,
			"mergeable":         state.Mergeable,
			"in_merge_queue":    state.InMergeQueue,
			"merge_queue_state": state.MergeQueueState,
			"checks":            s.Total,
			"passed":            s.Passed,
			"failed":            s.Failed,
			"pending":           s.Pending,
			"reviews":           len(state.Reviews),
			"comments":          len(state.Comments),
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
