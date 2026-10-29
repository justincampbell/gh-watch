package events

import (
	"fmt"
	"time"

	"github.com/justincampbell/gh-watch/internal/checks"
)

// diffCI compares old and new check runs and returns CI-related events.
// Shared by both PR and commit diffing. At most one CI event is emitted per diff.
func diffCI(oldChecks, newChecks []checks.CheckRun, now time.Time) []Event {
	oldMap := checks.BuildMap(oldChecks)

	var failed []map[string]interface{}
	var firstNew, firstFailed *checks.CheckRun
	for i, check := range newChecks {
		if !check.IsFailed() {
			continue
		}
		oldCheck, existed := oldMap[check.Name]
		isNew := !existed || oldCheck.Conclusion != check.Conclusion
		if firstFailed == nil {
			firstFailed = &newChecks[i]
		}
		if isNew && firstNew == nil {
			firstNew = &newChecks[i]
		}
		failed = append(failed, map[string]interface{}{
			"name":       check.Name,
			"conclusion": check.Conclusion,
			"url":        check.URL,
			"new":        isNew,
		})
	}

	// Report failures as soon as they appear, without waiting for the rest to complete.
	if firstNew != nil {
		return []Event{ciFailedEvent(*firstNew, failed, now)}
	}

	if len(newChecks) == 0 || !allCompleted(newChecks) || allCompleted(oldChecks) {
		return nil
	}
	if firstFailed != nil {
		return []Event{ciFailedEvent(*firstFailed, failed, now)}
	}
	return []Event{{
		Timestamp: now,
		Event:     CIAllPassed,
		Summary:   "All CI checks passed",
		Details:   map[string]interface{}{},
	}}
}

func ciFailedEvent(check checks.CheckRun, failed []map[string]interface{}, now time.Time) Event {
	summary := fmt.Sprintf("CI failed: %s (%s)", check.Name, check.Conclusion)
	if len(failed) > 1 {
		summary += fmt.Sprintf(" and %d more", len(failed)-1)
	}
	return Event{
		Timestamp: now,
		Event:     CIFailed,
		Summary:   summary,
		Details: map[string]interface{}{
			"name":       check.Name,
			"conclusion": check.Conclusion,
			"url":        check.URL,
			"failed":     failed,
		},
	}
}

func allCompleted(runs []checks.CheckRun) bool {
	for _, c := range runs {
		if c.Status != "COMPLETED" {
			return false
		}
	}
	return true
}
