package events

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/justincampbell/gh-watch/internal/branch"
)

// DiffBranch compares old and new branch states and returns detected events:
// one new-commit per commit that landed, oldest first. If old is nil, this is
// the initial fetch.
func DiffBranch(old, new *branch.State) []Event {
	if old == nil {
		return []Event{branchInitialStateEvent(new, time.Now())}
	}
	if old.SHA == new.SHA {
		return nil
	}

	now := time.Now()
	commits := new.Commits
	if new.Since != old.SHA || len(commits) == 0 {
		commits = []branch.Commit{{SHA: new.SHA, MessageHeadline: new.MessageHeadline, Author: new.Author}}
	}

	var out []Event
	previous := old.SHA
	for _, c := range commits {
		details := commitDetails(new.Name, c)
		details["previous_sha"] = previous
		if new.ForcePushed {
			details["force_pushed"] = true
		}
		out = append(out, Event{
			Timestamp: now,
			Event:     NewCommit,
			Summary:   fmt.Sprintf("New commit on %s: %s %s", new.Name, shortSHA(c.SHA), c.MessageHeadline),
			Details:   details,
		})
		previous = c.SHA
	}
	return out
}

// SkipAuthors drops new-commit events authored by any of logins.
func SkipAuthors(evts []Event, logins []string) []Event {
	if len(logins) == 0 {
		return evts
	}
	var kept []Event
	for _, e := range evts {
		author, _ := e.Details["author"].(string)
		if e.Event == NewCommit && slices.Contains(logins, author) {
			continue
		}
		kept = append(kept, e)
	}
	return kept
}

func branchInitialStateEvent(state *branch.State, now time.Time) Event {
	c := branch.Commit{SHA: state.SHA, MessageHeadline: state.MessageHeadline, Author: state.Author}
	return Event{
		Timestamp: now,
		Event:     InitialState,
		Summary:   fmt.Sprintf("Branch %s at %s: %s", state.Name, shortSHA(state.SHA), state.MessageHeadline),
		Details:   commitDetails(state.Name, c),
	}
}

func commitDetails(branchName string, c branch.Commit) map[string]interface{} {
	details := map[string]interface{}{
		"branch":           branchName,
		"sha":              c.SHA,
		"message_headline": c.MessageHeadline,
	}
	if c.Author != "" {
		details["author"] = c.Author
	}
	if n, ok := prNumberFromHeadline(c.MessageHeadline); ok {
		details["pr_number"] = n
	}
	return details
}

var (
	squashPRSuffix = regexp.MustCompile(`\(#(\d+)\)\s*$`)
	mergePRPrefix  = regexp.MustCompile(`^Merge pull request #(\d+) `)
)

func prNumberFromHeadline(headline string) (int, bool) {
	for _, re := range []*regexp.Regexp{squashPRSuffix, mergePRPrefix} {
		if m := re.FindStringSubmatch(headline); m != nil {
			n, err := strconv.Atoi(m[1])
			return n, err == nil
		}
	}
	return 0, false
}
