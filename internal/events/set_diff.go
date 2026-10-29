package events

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/justincampbell/gh-watch/internal/checks"
	"github.com/justincampbell/gh-watch/internal/pr"
)

// DiffSet compares snapshots of a PR search. Membership changes emit
// pr-added / pr-removed; PRs present in both are diffed individually.
// A PR that leaves after merging or closing was already reported by the
// per-PR diff, so its departure is silent.
func DiffSet(old, new *pr.SetState) []Event {
	now := time.Now()
	var out []Event

	for _, number := range slices.Sorted(maps.Keys(new.PRs)) {
		state := new.PRs[number]
		if old == nil {
			checks.ConfirmFailures(state.CheckRuns)
			out = append(out, initialStateEvent(state, now))
			continue
		}
		prev, ok := old.PRs[number]
		if !ok {
			checks.ConfirmFailures(state.CheckRuns)
			out = append(out, addedEvent(state, now))
			continue
		}
		out = append(out, Diff(prev, state)...)
	}

	if old == nil {
		return out
	}
	for _, number := range slices.Sorted(maps.Keys(old.PRs)) {
		prev := old.PRs[number]
		if _, ok := new.PRs[number]; ok || prev.Status != "open" {
			continue
		}
		out = append(out, Event{
			Timestamp: now,
			Event:     PRRemoved,
			Summary:   fmt.Sprintf("PR #%d no longer matches the search: %s", number, prev.Title),
			Details:   map[string]interface{}{"number": number, "title": prev.Title},
		})
	}
	return out
}

func addedEvent(state *pr.State, now time.Time) Event {
	e := initialStateEvent(state, now)
	e.Event = PRAdded
	e.Summary = "Added " + e.Summary
	return e
}
