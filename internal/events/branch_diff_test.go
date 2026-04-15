package events

import (
	"testing"

	"github.com/justincampbell/gh-watch/internal/branch"
)

func TestDiffBranch_NilOld(t *testing.T) {
	state := &branch.State{
		Name:            "main",
		SHA:             "abc1234567890def",
		MessageHeadline: "Initial commit",
		Author:          "octocat",
	}
	events := DiffBranch(nil, state)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Event != InitialState {
		t.Errorf("expected InitialState, got %s", e.Event)
	}
	if e.Details["branch"] != "main" {
		t.Errorf("expected branch main, got %v", e.Details["branch"])
	}
	if e.Details["sha"] != "abc1234567890def" {
		t.Errorf("expected full sha in details, got %v", e.Details["sha"])
	}
	if e.Details["author"] != "octocat" {
		t.Errorf("expected author octocat, got %v", e.Details["author"])
	}
	expected := "Branch main at abc1234: Initial commit"
	if e.Summary != expected {
		t.Errorf("expected summary %q, got %q", expected, e.Summary)
	}
}

func TestDiffBranch_NewCommit(t *testing.T) {
	old := &branch.State{
		Name:            "main",
		SHA:             "abc1234567890def",
		MessageHeadline: "Initial commit",
		Author:          "octocat",
	}
	new := &branch.State{
		Name:            "main",
		SHA:             "def4567890123abc",
		MessageHeadline: "Add feature",
		Author:          "monalisa",
	}
	events := DiffBranch(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Event != NewCommit {
		t.Errorf("expected NewCommit, got %s", e.Event)
	}
	if e.Details["sha"] != "def4567890123abc" {
		t.Errorf("expected new sha, got %v", e.Details["sha"])
	}
	if e.Details["previous_sha"] != "abc1234567890def" {
		t.Errorf("expected previous_sha, got %v", e.Details["previous_sha"])
	}
	if e.Details["author"] != "monalisa" {
		t.Errorf("expected author monalisa, got %v", e.Details["author"])
	}
}

func TestDiffBranch_NoChange(t *testing.T) {
	state := &branch.State{
		Name:            "main",
		SHA:             "abc1234567890def",
		MessageHeadline: "Initial commit",
	}
	events := DiffBranch(state, state)
	if len(events) != 0 {
		t.Errorf("expected no events for identical state, got %d", len(events))
	}
}

func TestDiffBranch_NoAuthor(t *testing.T) {
	state := &branch.State{
		Name:            "main",
		SHA:             "abc1234567890def",
		MessageHeadline: "Bot commit",
	}
	events := DiffBranch(nil, state)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if _, ok := events[0].Details["author"]; ok {
		t.Errorf("expected no author key when author is empty")
	}
}
