package events

import (
	"testing"

	"github.com/justincampbell/gh-watch/internal/tag"
)

func TestDiffTag_NilOld(t *testing.T) {
	state := &tag.State{
		Match: "v*",
		Tags: []tag.Tag{
			{Name: "v1.2.0", SHA: "abc1234567890def", MessageHeadline: "Release 1.2.0"},
			{Name: "v1.1.0", SHA: "111111111111", MessageHeadline: "Release 1.1.0"},
		},
	}
	events := DiffTag(nil, state)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Event != InitialState {
		t.Errorf("expected InitialState, got %s", e.Event)
	}
	if e.Details["tag_count"] != 2 {
		t.Errorf("expected tag_count 2, got %v", e.Details["tag_count"])
	}
	if e.Details["latest_tag"] != "v1.2.0" {
		t.Errorf("expected latest_tag v1.2.0, got %v", e.Details["latest_tag"])
	}
	if e.Details["match"] != "v*" {
		t.Errorf("expected match v*, got %v", e.Details["match"])
	}
}

func TestDiffTag_NilOld_ContainsPreexisting(t *testing.T) {
	state := &tag.State{
		ContainsTarget: "deadbeef",
		Tags: []tag.Tag{
			{Name: "v2.0.0", SHA: "aaa1111", MessageHeadline: "Release 2.0.0", Contains: false},
			{Name: "v1.9.0", SHA: "bbb2222", MessageHeadline: "Release 1.9.0", Contains: true},
		},
	}
	events := DiffTag(nil, state)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Event != InitialState {
		t.Errorf("expected first event InitialState, got %s", events[0].Event)
	}
	tc := events[1]
	if tc.Event != TagCreated {
		t.Fatalf("expected second event TagCreated, got %s", tc.Event)
	}
	if tc.Details["tag"] != "v1.9.0" {
		t.Errorf("expected newest containing tag v1.9.0, got %v", tc.Details["tag"])
	}
	if tc.Details["preexisting"] != true {
		t.Errorf("expected preexisting true, got %v", tc.Details["preexisting"])
	}
	if tc.Details["contains"] != "deadbeef" {
		t.Errorf("expected contains deadbeef, got %v", tc.Details["contains"])
	}
}

func TestDiffTag_NilOld_ContainsWaiting(t *testing.T) {
	state := &tag.State{
		ContainsTarget: "deadbeef",
		Tags: []tag.Tag{
			{Name: "v2.0.0", SHA: "aaa1111", MessageHeadline: "Release 2.0.0", Contains: false},
		},
	}
	events := DiffTag(nil, state)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != InitialState {
		t.Errorf("expected InitialState, got %s", events[0].Event)
	}
}

func TestDiffTag_NewTag(t *testing.T) {
	old := &tag.State{
		Tags: []tag.Tag{
			{Name: "v1.1.0", SHA: "111111111111", MessageHeadline: "Release 1.1.0"},
		},
	}
	new := &tag.State{
		Tags: []tag.Tag{
			{Name: "v1.2.0", SHA: "abc1234567890def", MessageHeadline: "Release 1.2.0"},
			{Name: "v1.1.0", SHA: "111111111111", MessageHeadline: "Release 1.1.0"},
		},
	}
	events := DiffTag(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Event != TagCreated {
		t.Errorf("expected TagCreated, got %s", e.Event)
	}
	if e.Details["tag"] != "v1.2.0" {
		t.Errorf("expected tag v1.2.0, got %v", e.Details["tag"])
	}
	if e.Details["sha"] != "abc1234567890def" {
		t.Errorf("expected full sha, got %v", e.Details["sha"])
	}
	if e.Details["message_headline"] != "Release 1.2.0" {
		t.Errorf("expected message_headline, got %v", e.Details["message_headline"])
	}
	expected := "New tag v1.2.0 at abc1234: Release 1.2.0"
	if e.Summary != expected {
		t.Errorf("expected summary %q, got %q", expected, e.Summary)
	}
}

func TestDiffTag_NoChange(t *testing.T) {
	state := &tag.State{
		Tags: []tag.Tag{
			{Name: "v1.1.0", SHA: "111111111111", MessageHeadline: "Release 1.1.0"},
		},
	}
	events := DiffTag(state, state)
	if len(events) != 0 {
		t.Errorf("expected no events for identical state, got %d", len(events))
	}
}

func TestDiffTag_ContainsFiltersOut(t *testing.T) {
	old := &tag.State{ContainsTarget: "deadbeef"}
	new := &tag.State{
		ContainsTarget: "deadbeef",
		Tags: []tag.Tag{
			{Name: "v2.0.0", SHA: "aaa1111", MessageHeadline: "Release 2.0.0", Contains: false},
		},
	}
	events := DiffTag(old, new)
	if len(events) != 0 {
		t.Errorf("expected no events when new tag does not contain target, got %d", len(events))
	}
}

func TestDiffTag_ContainsMatches(t *testing.T) {
	old := &tag.State{ContainsTarget: "deadbeef"}
	new := &tag.State{
		ContainsTarget: "deadbeef",
		Tags: []tag.Tag{
			{Name: "v2.0.0", SHA: "aaa1111", MessageHeadline: "Release 2.0.0", Contains: true},
		},
	}
	events := DiffTag(old, new)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != TagCreated {
		t.Errorf("expected TagCreated, got %s", events[0].Event)
	}
	if events[0].Details["preexisting"] != nil {
		t.Errorf("expected no preexisting key for a newly-appeared tag, got %v", events[0].Details["preexisting"])
	}
}
