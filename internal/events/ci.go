package events

import (
	"fmt"
	"time"

	"github.com/justincampbell/gh-watch/internal/checks"
)

// diffCI compares old and new check runs on a head commit and returns
// CI-related events. Shared by both PR and commit diffing. At most one CI
// event is emitted per diff.
//
// A failure is only reported once it's confirmed: the same check failed with
// the same conclusion on the same head SHA in two consecutive polls. CI
// providers briefly report superseded pipelines as failed when a branch is
// pushed twice in quick succession, and those failures vanish a poll later.
// Confirmation is recorded on newChecks so the next diff sees it.
func diffCI(oldSHA string, oldChecks []checks.CheckRun, newSHA string, newChecks []checks.CheckRun, now time.Time) []Event {
	oldMap := checks.BuildMap(oldChecks)

	var failed []map[string]interface{}
	var firstNew *checks.CheckRun
	anyFailed := false
	for i, check := range newChecks {
		if !check.IsFailed() {
			continue
		}
		anyFailed = true
		oldCheck, existed := oldMap[check.Name]
		if oldSHA != newSHA || !existed || !oldCheck.IsFailed() || oldCheck.Conclusion != check.Conclusion {
			continue
		}
		newChecks[i].FailureConfirmed = true
		isNew := !oldCheck.FailureConfirmed
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

	if firstNew != nil {
		return []Event{ciFailedEvent(newSHA, *firstNew, failed, now)}
	}

	if anyFailed || len(newChecks) == 0 || !allCompleted(newChecks) || allCompleted(oldChecks) {
		return nil
	}
	return []Event{{
		Timestamp: now,
		Event:     CIAllPassed,
		Summary:   "All CI checks passed",
		Details:   map[string]interface{}{"sha": newSHA},
	}}
}

func ciFailedEvent(sha string, check checks.CheckRun, failed []map[string]interface{}, now time.Time) Event {
	summary := fmt.Sprintf("CI failed: %s (%s)", check.Name, check.Conclusion)
	if len(failed) > 1 {
		summary += fmt.Sprintf(" and %d more", len(failed)-1)
	}
	return Event{
		Timestamp: now,
		Event:     CIFailed,
		Summary:   summary,
		Details: map[string]interface{}{
			"sha":        sha,
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
