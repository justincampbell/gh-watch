package events

import (
	"fmt"
	"time"

	"github.com/justincampbell/gh-watch/internal/checks"
	"github.com/justincampbell/gh-watch/internal/commit"
)

// DiffCommit compares old and new commit states and returns detected events.
// If old is nil, this is the initial fetch.
func DiffCommit(old, new *commit.State) []Event {
	if old == nil {
		return []Event{commitInitialStateEvent(new, time.Now())}
	}

	return diffCI(old.CheckRuns, new.CheckRuns, time.Now())
}

func commitInitialStateEvent(state *commit.State, now time.Time) Event {
	s := checks.Summarize(state.CheckRuns)

	shortSHA := state.SHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	var summary string
	if s.Total == 0 {
		summary = fmt.Sprintf("Commit %s: no CI checks", shortSHA)
	} else if s.Pending > 0 {
		summary = fmt.Sprintf("Commit %s: CI %d/%d passed, %d pending", shortSHA, s.Passed, s.Total, s.Pending)
	} else if s.Failed > 0 {
		summary = fmt.Sprintf("Commit %s: CI %d/%d passed, %d failed", shortSHA, s.Passed, s.Total, s.Failed)
	} else {
		summary = fmt.Sprintf("Commit %s: all %d CI checks passed", shortSHA, s.Total)
	}

	return Event{
		Timestamp: now,
		Event:     InitialState,
		Summary:   summary,
		Details: map[string]interface{}{
			"sha":     state.SHA,
			"checks":  s.Total,
			"passed":  s.Passed,
			"failed":  s.Failed,
			"pending": s.Pending,
		},
	}
}
