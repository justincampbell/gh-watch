package events

import (
	"fmt"
	"strings"
	"time"

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

	// CI: emit ci-failed immediately when any check fails
	oldChecks := buildCheckMap(old.CheckRuns)
	for _, check := range new.CheckRuns {
		if check.Conclusion == "FAILURE" || check.Conclusion == "TIMED_OUT" || check.Conclusion == "CANCELLED" {
			oldCheck, existed := oldChecks[check.Name]
			if !existed || oldCheck.Conclusion != check.Conclusion {
				events = append(events, Event{
					Timestamp: now,
					Event:     CIFailed,
					Summary:   fmt.Sprintf("CI failed: %s", check.Name),
					Details: map[string]interface{}{
						"name": check.Name,
						"url":  check.URL,
					},
				})
				break
			}
		}
	}

	// CI: emit ci-passed or ci-failed when all checks complete
	allCompleted := len(new.CheckRuns) > 0
	anyFailed := false
	var firstFailedName, firstFailedURL string
	for _, check := range new.CheckRuns {
		if check.Status != "COMPLETED" {
			allCompleted = false
		}
		if check.Conclusion == "FAILURE" || check.Conclusion == "TIMED_OUT" || check.Conclusion == "CANCELLED" {
			anyFailed = true
			if firstFailedName == "" {
				firstFailedName = check.Name
				firstFailedURL = check.URL
			}
		}
	}
	if allCompleted && len(new.CheckRuns) > 0 {
		oldAllCompleted := true
		for _, c := range old.CheckRuns {
			if c.Status != "COMPLETED" {
				oldAllCompleted = false
				break
			}
		}
		if !oldAllCompleted {
			if anyFailed {
				// Only emit if we didn't already emit ci-failed for a newly-failed check above
				alreadyEmitted := false
				for _, e := range events {
					if e.Event == CIFailed {
						alreadyEmitted = true
						break
					}
				}
				if !alreadyEmitted {
					events = append(events, Event{
						Timestamp: now,
						Event:     CIFailed,
						Summary:   fmt.Sprintf("CI failed: %s", firstFailedName),
						Details: map[string]interface{}{
							"name": firstFailedName,
							"url":  firstFailedURL,
						},
					})
				}
			} else {
				events = append(events, Event{
					Timestamp: now,
					Event:     CIAllPassed,
					Summary:   "All CI checks passed",
					Details:   map[string]interface{}{},
				})
			}
		}
	}

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

	// CI summary
	completed, passed, failed, pending := 0, 0, 0, 0
	for _, c := range state.CheckRuns {
		if c.Status == "COMPLETED" {
			completed++
			if c.Conclusion == "FAILURE" || c.Conclusion == "TIMED_OUT" || c.Conclusion == "CANCELLED" {
				failed++
			} else {
				passed++
			}
		} else {
			pending++
		}
	}
	if len(state.CheckRuns) > 0 {
		if pending > 0 {
			parts = append(parts, fmt.Sprintf("CI: %d/%d passed, %d pending", passed, len(state.CheckRuns), pending))
		} else if failed > 0 {
			parts = append(parts, fmt.Sprintf("CI: %d/%d passed, %d failed", passed, len(state.CheckRuns), failed))
		} else {
			parts = append(parts, fmt.Sprintf("CI: all %d checks passed", len(state.CheckRuns)))
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
			"checks":    len(state.CheckRuns),
			"passed":    passed,
			"failed":    failed,
			"pending":   pending,
			"reviews":   len(state.Reviews),
			"comments":  len(state.Comments),
		},
	}
}

func buildCheckMap(checks []pr.CheckRun) map[string]pr.CheckRun {
	m := make(map[string]pr.CheckRun, len(checks))
	for _, c := range checks {
		m[c.Name] = c
	}
	return m
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
