package events

import (
	"fmt"
	"time"

	"github.com/justincampbell/gh-watch/internal/tag"
)

// DiffTag compares old and new tag states and returns detected events.
// If old is nil, this is the initial fetch.
func DiffTag(old, new *tag.State) []Event {
	if old == nil {
		out := []Event{tagInitialStateEvent(new, time.Now())}

		// In contains-mode, if a tag already carries the target at watch start,
		// emit tag-created for the newest such tag so --exit / --exit-on can fire
		// (initial-state alone never triggers exit).
		if new.ContainsTarget != "" {
			for _, t := range new.Tags {
				if t.Contains {
					out = append(out, tagCreatedEvent(new, t, true))
					break
				}
			}
		}

		return out
	}

	seen := make(map[string]struct{}, len(old.Tags))
	for _, t := range old.Tags {
		seen[t.Name] = struct{}{}
	}

	var out []Event
	for _, t := range new.Tags {
		if _, ok := seen[t.Name]; ok {
			continue
		}
		if new.ContainsTarget != "" && !t.Contains {
			continue
		}
		out = append(out, tagCreatedEvent(new, t, false))
	}

	return out
}

func tagCreatedEvent(state *tag.State, t tag.Tag, preexisting bool) Event {
	summary := fmt.Sprintf("New tag %s at %s: %s", t.Name, shortSHA(t.SHA), t.MessageHeadline)

	details := map[string]interface{}{
		"tag":              t.Name,
		"sha":              t.SHA,
		"message_headline": t.MessageHeadline,
	}
	if state.ContainsTarget != "" {
		details["contains"] = state.ContainsTarget
	}
	if preexisting {
		details["preexisting"] = true
	}

	return Event{
		Timestamp: time.Now(),
		Event:     TagCreated,
		Summary:   summary,
		Details:   details,
	}
}

func tagInitialStateEvent(state *tag.State, now time.Time) Event {
	summary := fmt.Sprintf("%d tags", len(state.Tags))
	if state.Match != "" {
		summary = fmt.Sprintf("%d tags matching %s", len(state.Tags), state.Match)
	}

	details := map[string]interface{}{
		"tag_count": len(state.Tags),
	}
	if state.Match != "" {
		details["match"] = state.Match
	}
	if state.ContainsTarget != "" {
		details["contains"] = state.ContainsTarget
	}
	if len(state.Tags) > 0 {
		latest := state.Tags[0]
		details["latest_tag"] = latest.Name
		details["latest_sha"] = latest.SHA
		summary = fmt.Sprintf("%s (latest: %s)", summary, latest.Name)
	}

	return Event{
		Timestamp: now,
		Event:     InitialState,
		Summary:   summary,
		Details:   details,
	}
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
