package events

import (
	"testing"

	"github.com/justincampbell/gh-watch/internal/checks"
	"github.com/justincampbell/gh-watch/internal/commit"
)

func TestDiffCommit_NilOld_NoChecks(t *testing.T) {
	state := &commit.State{SHA: "abc1234567890"}
	events := DiffCommit(nil, state)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != InitialState {
		t.Errorf("expected InitialState, got %s", events[0].Event)
	}
	if events[0].Details["sha"] != "abc1234567890" {
		t.Errorf("expected sha in details, got %v", events[0].Details["sha"])
	}
}

func TestDiffCommit_NilOld_WithChecks(t *testing.T) {
	state := &commit.State{
		SHA: "abc1234567890",
		CheckRuns: []checks.CheckRun{
			{Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Name: "lint", Status: "IN_PROGRESS", Conclusion: ""},
		},
	}
	events := DiffCommit(nil, state)
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
	if e.Details["pending"] != 1 {
		t.Errorf("expected 1 pending, got %v", e.Details["pending"])
	}
}

func TestDiffCommit_CIAllPassed(t *testing.T) {
	old := &commit.State{
		SHA: "abc1234",
		CheckRuns: []checks.CheckRun{
			{Name: "test", Status: "IN_PROGRESS", Conclusion: ""},
		},
	}
	new := &commit.State{
		SHA: "abc1234",
		CheckRuns: []checks.CheckRun{
			{Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
		},
	}
	events := DiffCommit(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != CIAllPassed {
		t.Errorf("expected CIAllPassed, got %s", events[0].Event)
	}
}

func TestDiffCommit_CIFailed_Immediate(t *testing.T) {
	old := &commit.State{
		SHA: "abc1234",
		CheckRuns: []checks.CheckRun{
			{Name: "test", Status: "IN_PROGRESS", Conclusion: ""},
			{Name: "lint", Status: "IN_PROGRESS", Conclusion: ""},
		},
	}
	new := &commit.State{
		SHA: "abc1234",
		CheckRuns: []checks.CheckRun{
			{Name: "test", Status: "IN_PROGRESS", Conclusion: ""},
			{Name: "lint", Status: "COMPLETED", Conclusion: "FAILURE", URL: "https://example.com/lint"},
		},
	}
	events := DiffCommit(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != CIFailed {
		t.Errorf("expected CIFailed, got %s", events[0].Event)
	}
	if events[0].Details["name"] != "lint" {
		t.Errorf("expected failed check name 'lint', got %v", events[0].Details["name"])
	}
}

func TestDiffCommit_CIFailed_AllCompleteWithPriorFailure(t *testing.T) {
	old := &commit.State{
		SHA: "abc1234",
		CheckRuns: []checks.CheckRun{
			{Name: "test", Status: "IN_PROGRESS", Conclusion: ""},
			{Name: "lint", Status: "COMPLETED", Conclusion: "FAILURE"},
		},
	}
	new := &commit.State{
		SHA: "abc1234",
		CheckRuns: []checks.CheckRun{
			{Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Name: "lint", Status: "COMPLETED", Conclusion: "FAILURE"},
		},
	}
	events := DiffCommit(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != CIFailed {
		t.Errorf("expected CIFailed, got %s", events[0].Event)
	}
}

func TestDiffCommit_NoChangeNoEvents(t *testing.T) {
	state := &commit.State{
		SHA: "abc1234",
		CheckRuns: []checks.CheckRun{
			{Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
		},
	}
	events := DiffCommit(state, state)
	if len(events) != 0 {
		t.Errorf("expected no events for identical state, got %d", len(events))
	}
}

func TestDiffCommit_NotReEmittedWhenAlreadyAllComplete(t *testing.T) {
	old := &commit.State{
		SHA: "abc1234",
		CheckRuns: []checks.CheckRun{
			{Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Name: "lint", Status: "COMPLETED", Conclusion: "FAILURE"},
		},
	}
	new := &commit.State{
		SHA: "abc1234",
		CheckRuns: []checks.CheckRun{
			{Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Name: "lint", Status: "COMPLETED", Conclusion: "FAILURE"},
		},
	}
	events := DiffCommit(old, new)
	if len(events) != 0 {
		t.Errorf("expected no events when all checks were already complete, got %d", len(events))
	}
}

func TestDiffCommit_ShortSHAInSummary(t *testing.T) {
	state := &commit.State{
		SHA: "abc1234567890def",
		CheckRuns: []checks.CheckRun{
			{Name: "test", Status: "IN_PROGRESS"},
		},
	}
	events := DiffCommit(nil, state)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	expected := "Commit abc1234: CI 0/1 passed, 1 pending"
	if events[0].Summary != expected {
		t.Errorf("expected summary %q, got %q", expected, events[0].Summary)
	}
}
