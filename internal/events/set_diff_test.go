package events

import (
	"testing"

	"github.com/justincampbell/gh-watch/internal/pr"
)

func set(states ...*pr.State) *pr.SetState {
	s := &pr.SetState{PRs: map[int]*pr.State{}}
	for _, st := range states {
		s.PRs[st.Number] = st
	}
	return s
}

func eventTypes(es []Event) []EventType {
	var out []EventType
	for _, e := range es {
		out = append(out, e.Event)
	}
	return out
}

func TestDiffSet_InitialStatePerPR(t *testing.T) {
	events := DiffSet(nil, set(
		&pr.State{Number: 2, Status: "open"},
		&pr.State{Number: 1, Status: "merged"},
	))
	if len(events) != 2 || events[0].Event != InitialState || events[1].Event != InitialState {
		t.Fatalf("expected two initial-state events, got %v", eventTypes(events))
	}
	if events[0].Details["number"] != 1 {
		t.Errorf("expected PRs in number order, got %v first", events[0].Details["number"])
	}
}

func TestDiffSet_Added(t *testing.T) {
	old := set(&pr.State{Number: 1, Status: "open"})
	new := set(&pr.State{Number: 1, Status: "open"}, &pr.State{Number: 2, Status: "open", IsDraft: true})
	events := DiffSet(old, new)
	if len(events) != 1 || events[0].Event != PRAdded {
		t.Fatalf("expected pr-added, got %v", eventTypes(events))
	}
	if events[0].Details["number"] != 2 || events[0].Details["draft"] != true {
		t.Errorf("unexpected details: %v", events[0].Details)
	}
}

func TestDiffSet_RemovedWhileOpen(t *testing.T) {
	events := DiffSet(set(&pr.State{Number: 1, Status: "open"}), set())
	if len(events) != 1 || events[0].Event != PRRemoved {
		t.Fatalf("expected pr-removed, got %v", eventTypes(events))
	}
}

func TestDiffSet_MergedThenGoneReportsOnce(t *testing.T) {
	s1 := set(&pr.State{Number: 1, Status: "open"})
	s2 := set(&pr.State{Number: 1, Status: "merged"})
	events := DiffSet(s1, s2)
	if len(events) != 1 || events[0].Event != PRMerged {
		t.Fatalf("expected pr-merged, got %v", eventTypes(events))
	}
	if events := DiffSet(s2, set()); len(events) != 0 {
		t.Fatalf("expected silent departure after merge, got %v", eventTypes(events))
	}
}

func TestDiffSet_PerPRChanges(t *testing.T) {
	old := set(&pr.State{Number: 1, Status: "open", IsDraft: true}, &pr.State{Number: 2, Status: "open"})
	new := set(&pr.State{Number: 1, Status: "open"}, &pr.State{Number: 2, Status: "open", AutoMerge: &pr.AutoMerge{}})
	events := DiffSet(old, new)
	if len(events) != 2 || events[0].Event != ReadyForReview || events[1].Event != AutoMergeEnabled {
		t.Fatalf("expected ready-for-review then auto-merge-enabled, got %v", eventTypes(events))
	}
}
