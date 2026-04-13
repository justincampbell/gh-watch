package events

import (
	"testing"

	"github.com/justincampbell/gh-watch/internal/pr"
)

func TestDiff_NilOld_OpenPR(t *testing.T) {
	state := &pr.State{Number: 1, Title: "Test PR", Status: "open"}
	events := Diff(nil, state)
	if len(events) != 1 {
		t.Fatalf("expected 1 initial-state event, got %d", len(events))
	}
	if events[0].Event != InitialState {
		t.Errorf("expected InitialState, got %s", events[0].Event)
	}
}

func TestDiff_NilOld_InitialStateWithChecks(t *testing.T) {
	state := &pr.State{
		Number: 42,
		Title:  "Add feature",
		Status: "open",
		CheckRuns: []pr.CheckRun{
			{Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Name: "lint", Status: "IN_PROGRESS", Conclusion: ""},
			{Name: "build", Status: "QUEUED", Conclusion: ""},
		},
		Reviews: []pr.Review{
			{Author: "alice", State: "APPROVED"},
		},
	}
	events := Diff(nil, state)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Event != InitialState {
		t.Errorf("expected InitialState, got %s", e.Event)
	}
	if e.Details["passed"] != 1 {
		t.Errorf("expected 1 passed, got %v", e.Details["passed"])
	}
	if e.Details["pending"] != 2 {
		t.Errorf("expected 2 pending, got %v", e.Details["pending"])
	}
	if e.Details["reviews"] != 1 {
		t.Errorf("expected 1 review, got %v", e.Details["reviews"])
	}
}

func TestDiff_NilOld_MergedPR(t *testing.T) {
	state := &pr.State{Number: 1, Status: "merged"}
	events := Diff(nil, state)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != PRMerged {
		t.Errorf("expected PRMerged, got %s", events[0].Event)
	}
}

func TestDiff_NilOld_ClosedPR(t *testing.T) {
	state := &pr.State{Number: 1, Status: "closed"}
	events := Diff(nil, state)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != PRClosed {
		t.Errorf("expected PRClosed, got %s", events[0].Event)
	}
}

func TestDiff_PRMerged(t *testing.T) {
	old := &pr.State{Number: 1, Status: "open"}
	new := &pr.State{Number: 1, Status: "merged"}
	events := Diff(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != PRMerged {
		t.Errorf("expected PRMerged, got %s", events[0].Event)
	}
}

func TestDiff_PRClosed(t *testing.T) {
	old := &pr.State{Number: 1, Status: "open"}
	new := &pr.State{Number: 1, Status: "closed"}
	events := Diff(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != PRClosed {
		t.Errorf("expected PRClosed, got %s", events[0].Event)
	}
}

func TestDiff_CIStatusChanged(t *testing.T) {
	old := &pr.State{
		Number: 1,
		Status: "open",
		CheckRuns: []pr.CheckRun{
			{Name: "test", Status: "IN_PROGRESS", Conclusion: ""},
		},
	}
	new := &pr.State{
		Number: 1,
		Status: "open",
		CheckRuns: []pr.CheckRun{
			{Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
		},
	}
	events := Diff(old, new)

	hasStatusChange := false
	hasAllPassed := false
	for _, e := range events {
		if e.Event == CIStatusChanged {
			hasStatusChange = true
		}
		if e.Event == CIAllPassed {
			hasAllPassed = true
		}
	}
	if !hasStatusChange {
		t.Error("expected CIStatusChanged event")
	}
	if !hasAllPassed {
		t.Error("expected CIAllPassed event")
	}
}

func TestDiff_CIFailed(t *testing.T) {
	old := &pr.State{
		Number: 1,
		Status: "open",
		CheckRuns: []pr.CheckRun{
			{Name: "test", Status: "IN_PROGRESS", Conclusion: ""},
			{Name: "lint", Status: "IN_PROGRESS", Conclusion: ""},
		},
	}
	new := &pr.State{
		Number: 1,
		Status: "open",
		CheckRuns: []pr.CheckRun{
			{Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Name: "lint", Status: "COMPLETED", Conclusion: "FAILURE", URL: "https://example.com/lint"},
		},
	}
	events := Diff(old, new)

	hasFailed := false
	for _, e := range events {
		if e.Event == CIFailed {
			hasFailed = true
			if e.Details["name"] != "lint" {
				t.Errorf("expected failed check name 'lint', got %v", e.Details["name"])
			}
		}
	}
	if !hasFailed {
		t.Error("expected CIFailed event")
	}
}

func TestDiff_ReviewSubmitted(t *testing.T) {
	old := &pr.State{
		Number:  1,
		Status:  "open",
		Reviews: []pr.Review{},
	}
	new := &pr.State{
		Number: 1,
		Status: "open",
		Reviews: []pr.Review{
			{Author: "alice", State: "APPROVED", Body: "LGTM!"},
		},
	}
	events := Diff(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != ReviewSubmitted {
		t.Errorf("expected ReviewSubmitted, got %s", events[0].Event)
	}
	if events[0].Details["author"] != "alice" {
		t.Errorf("expected author alice, got %v", events[0].Details["author"])
	}
}

func TestDiff_CommentAdded(t *testing.T) {
	old := &pr.State{
		Number:   1,
		Status:   "open",
		Comments: []pr.Comment{},
	}
	new := &pr.State{
		Number: 1,
		Status: "open",
		Comments: []pr.Comment{
			{ID: "1", Author: "bob", Body: "Nice work!"},
		},
	}
	events := Diff(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != CommentAdded {
		t.Errorf("expected CommentAdded, got %s", events[0].Event)
	}
}

func TestDiff_MergeConflictChanged(t *testing.T) {
	old := &pr.State{Number: 1, Status: "open", Mergeable: "MERGEABLE"}
	new := &pr.State{Number: 1, Status: "open", Mergeable: "CONFLICTING"}
	events := Diff(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != MergeConflictChanged {
		t.Errorf("expected MergeConflictChanged, got %s", events[0].Event)
	}
}

func TestDiff_MergeConflictIgnoredWhenUnknown(t *testing.T) {
	old := &pr.State{Number: 1, Status: "open", Mergeable: "UNKNOWN"}
	new := &pr.State{Number: 1, Status: "open", Mergeable: "MERGEABLE"}
	events := Diff(old, new)
	if len(events) != 0 {
		t.Errorf("expected no events when transitioning from UNKNOWN, got %d", len(events))
	}
}

func TestDiff_NoChangeNoEvents(t *testing.T) {
	state := &pr.State{
		Number:    1,
		Status:    "open",
		Mergeable: "MERGEABLE",
		CheckRuns: []pr.CheckRun{
			{Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
		},
		Reviews: []pr.Review{
			{Author: "alice", State: "APPROVED"},
		},
		Comments: []pr.Comment{
			{ID: "1", Author: "bob", Body: "ok"},
		},
	}
	events := Diff(state, state)
	if len(events) != 0 {
		t.Errorf("expected no events for identical state, got %d", len(events))
	}
}

func TestDiff_MultipleNewReviews(t *testing.T) {
	old := &pr.State{
		Number: 1,
		Status: "open",
		Reviews: []pr.Review{
			{Author: "alice", State: "COMMENTED"},
		},
	}
	new := &pr.State{
		Number: 1,
		Status: "open",
		Reviews: []pr.Review{
			{Author: "alice", State: "COMMENTED"},
			{Author: "bob", State: "APPROVED"},
			{Author: "carol", State: "CHANGES_REQUESTED"},
		},
	}
	events := Diff(old, new)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	for _, e := range events {
		if e.Event != ReviewSubmitted {
			t.Errorf("expected ReviewSubmitted, got %s", e.Event)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("expected 'short', got '%s'", got)
	}
	if got := truncate("this is a longer string", 10); got != "this is..." {
		t.Errorf("expected 'this is...', got '%s'", got)
	}
	if got := truncate("line1\nline2", 20); got != "line1 line2" {
		t.Errorf("expected 'line1 line2', got '%s'", got)
	}
}
