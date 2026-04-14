package events

import (
	"fmt"
	"time"

	"github.com/justincampbell/gh-watch/internal/checks"
)

// diffCI compares old and new check runs and returns CI-related events.
// Shared by both PR and commit diffing.
func diffCI(oldChecks, newChecks []checks.CheckRun, now time.Time) []Event {
	var events []Event

	// Emit ci-failed immediately when any check newly fails
	oldMap := checks.BuildMap(oldChecks)
	for _, check := range newChecks {
		if check.IsFailed() {
			oldCheck, existed := oldMap[check.Name]
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

	// Emit ci-passed or ci-failed when all checks complete
	allCompleted := len(newChecks) > 0
	anyFailed := false
	var firstFailedName, firstFailedURL string
	for _, check := range newChecks {
		if check.Status != "COMPLETED" {
			allCompleted = false
		}
		if check.IsFailed() {
			anyFailed = true
			if firstFailedName == "" {
				firstFailedName = check.Name
				firstFailedURL = check.URL
			}
		}
	}
	if allCompleted && len(newChecks) > 0 {
		oldAllCompleted := true
		for _, c := range oldChecks {
			if c.Status != "COMPLETED" {
				oldAllCompleted = false
				break
			}
		}
		if !oldAllCompleted {
			if anyFailed {
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

	return events
}
