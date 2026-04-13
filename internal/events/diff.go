package events

import (
	"fmt"
	"strings"
	"time"

	"github.com/justincampbell/gh-watch-pr/internal/pr"
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
		return events
	}

	// CI status changes
	oldChecks := buildCheckMap(old.CheckRuns)
	allCompleted := len(new.CheckRuns) > 0
	anyFailed := false
	var failedName, failedURL string

	for _, check := range new.CheckRuns {
		oldCheck, existed := oldChecks[check.Name]
		if check.Status != "COMPLETED" {
			allCompleted = false
		}

		if existed && (oldCheck.Status != check.Status || oldCheck.Conclusion != check.Conclusion) {
			summary := fmt.Sprintf("CI: %s → %s", check.Name, formatCheckState(check))
			events = append(events, Event{
				Timestamp: now,
				Event:     CIStatusChanged,
				Summary:   summary,
				Details: map[string]interface{}{
					"name":       check.Name,
					"status":     check.Status,
					"conclusion": check.Conclusion,
					"url":        check.URL,
				},
			})
		}

		if check.Conclusion == "FAILURE" || check.Conclusion == "TIMED_OUT" || check.Conclusion == "CANCELLED" {
			anyFailed = true
			if failedName == "" {
				failedName = check.Name
				failedURL = check.URL
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
				events = append(events, Event{
					Timestamp: now,
					Event:     CIFailed,
					Summary:   fmt.Sprintf("CI failed: %s", failedName),
					Details: map[string]interface{}{
						"name": failedName,
						"url":  failedURL,
					},
				})
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

func buildCheckMap(checks []pr.CheckRun) map[string]pr.CheckRun {
	m := make(map[string]pr.CheckRun, len(checks))
	for _, c := range checks {
		m[c.Name] = c
	}
	return m
}

func formatCheckState(c pr.CheckRun) string {
	if c.Status == "COMPLETED" {
		return strings.ToLower(c.Conclusion)
	}
	return strings.ToLower(c.Status)
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
