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

func TestDiffBranch_EveryCommitSinceLastHead(t *testing.T) {
	old := &branch.State{Name: "main", SHA: "aaa"}
	new := &branch.State{
		Name: "main", SHA: "ccc", MessageHeadline: "Merge pull request #12 from x/y",
		Since: "aaa",
		Commits: []branch.Commit{
			{SHA: "bbb", MessageHeadline: "Fix thing (#11)", Author: "alice"},
			{SHA: "ccc", MessageHeadline: "Merge pull request #12 from x/y", Author: "bob"},
		},
	}
	events := DiffBranch(old, new)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Details["sha"] != "bbb" || events[0].Details["previous_sha"] != "aaa" || events[0].Details["pr_number"] != 11 {
		t.Errorf("unexpected first commit details: %v", events[0].Details)
	}
	if events[1].Details["previous_sha"] != "bbb" || events[1].Details["pr_number"] != 12 {
		t.Errorf("unexpected second commit details: %v", events[1].Details)
	}
}

func TestDiffBranch_StaleCommitListFallsBackToHead(t *testing.T) {
	old := &branch.State{Name: "main", SHA: "zzz"}
	new := &branch.State{Name: "main", SHA: "ccc", Since: "aaa", Commits: []branch.Commit{{SHA: "bbb"}, {SHA: "ccc"}}}
	events := DiffBranch(old, new)
	if len(events) != 1 || events[0].Details["sha"] != "ccc" {
		t.Fatalf("expected head-only event, got %v", events)
	}
}

func TestDiffBranch_ForcePushed(t *testing.T) {
	old := &branch.State{Name: "main", SHA: "aaa"}
	new := &branch.State{Name: "main", SHA: "ccc", Since: "aaa", ForcePushed: true}
	events := DiffBranch(old, new)
	if len(events) != 1 || events[0].Details["force_pushed"] != true {
		t.Fatalf("expected one force-pushed event, got %v", events)
	}
}

func TestSkipAuthors(t *testing.T) {
	evts := []Event{
		{Event: InitialState, Details: map[string]interface{}{"author": "me"}},
		{Event: NewCommit, Details: map[string]interface{}{"author": "me"}},
		{Event: NewCommit, Details: map[string]interface{}{"author": "alice"}},
	}
	kept := SkipAuthors(evts, []string{"me"})
	if len(kept) != 2 || kept[1].Details["author"] != "alice" {
		t.Errorf("unexpected kept events: %v", kept)
	}
}
