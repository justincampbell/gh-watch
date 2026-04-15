package events

import (
	"fmt"
	"time"

	"github.com/justincampbell/gh-watch/internal/branch"
)

// DiffBranch compares old and new branch states and returns detected events.
// If old is nil, this is the initial fetch.
func DiffBranch(old, new *branch.State) []Event {
	if old == nil {
		return []Event{branchInitialStateEvent(new, time.Now())}
	}

	var out []Event

	if old.SHA != new.SHA {
		shortSHA := new.SHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}

		summary := fmt.Sprintf("New commit on %s: %s %s", new.Name, shortSHA, new.MessageHeadline)

		details := map[string]interface{}{
			"branch":           new.Name,
			"sha":              new.SHA,
			"previous_sha":     old.SHA,
			"message_headline": new.MessageHeadline,
		}
		if new.Author != "" {
			details["author"] = new.Author
		}

		out = append(out, Event{
			Timestamp: time.Now(),
			Event:     NewCommit,
			Summary:   summary,
			Details:   details,
		})
	}

	return out
}

func branchInitialStateEvent(state *branch.State, now time.Time) Event {
	shortSHA := state.SHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	summary := fmt.Sprintf("Branch %s at %s: %s", state.Name, shortSHA, state.MessageHeadline)

	details := map[string]interface{}{
		"branch":           state.Name,
		"sha":              state.SHA,
		"message_headline": state.MessageHeadline,
	}
	if state.Author != "" {
		details["author"] = state.Author
	}

	return Event{
		Timestamp: now,
		Event:     InitialState,
		Summary:   summary,
		Details:   details,
	}
}
