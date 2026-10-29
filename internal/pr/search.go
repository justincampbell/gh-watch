package pr

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

// searchLimit is GitHub's per-page maximum for search.
const searchLimit = 100

// SetState is a snapshot of every PR matching a search, keyed by number.
type SetState struct {
	PRs map[int]*State
}

// SearchFetcher snapshots the PRs matching a query in one repository.
type SearchFetcher struct {
	Owner, Repo, Query string
	Single             Fetcher
	Warn               io.Writer

	warned bool
}

// ValidateSearch rejects qualifiers that would widen the search beyond the
// current repository, where PR numbers stop being unique.
func ValidateSearch(query string) error {
	for _, field := range strings.Fields(query) {
		for _, q := range []string{"repo:", "org:", "user:"} {
			if strings.HasPrefix(strings.TrimPrefix(field, "-"), q) {
				return fmt.Errorf("--search is scoped to the current repository; remove %q", field)
			}
		}
	}
	return nil
}

// Fetch runs the search. PRs in prev that left the results while open are
// refetched once: a merged or closed one stays in the snapshot so the diff
// reports how it left, and one that still exists open is dropped.
func (f *SearchFetcher) Fetch(prev *SetState) (*SetState, error) {
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("creating GraphQL client: %w", err)
	}

	query := `query PRSearch($q: String!, $first: Int!, $last: Int!) {
		search(query: $q, type: ISSUE, first: $first) {
			issueCount
			nodes { ... on PullRequest { ...prFields } }
		}
	}
	` + fragment

	variables := map[string]interface{}{
		"q":     fmt.Sprintf("repo:%s/%s is:pr %s", f.Owner, f.Repo, f.Query),
		"first": searchLimit,
		"last":  20,
	}

	var resp struct {
		Search struct {
			IssueCount int
			Nodes      []node
		}
	}
	if err := client.Do(query, variables, &resp); err != nil {
		return nil, fmt.Errorf("searching PRs: %w", err)
	}

	if resp.Search.IssueCount > searchLimit && !f.warned {
		f.warned = true
		fmt.Fprintf(f.warnOut(), "warning: search matched %d PRs; watching the first %d\n", resp.Search.IssueCount, searchLimit)
	}

	set := &SetState{PRs: make(map[int]*State, len(resp.Search.Nodes))}
	for _, n := range resp.Search.Nodes {
		set.PRs[n.Number] = n.toState()
	}

	if prev == nil {
		return set, nil
	}
	for number, old := range prev.PRs {
		if _, ok := set.PRs[number]; ok || old.Status != "open" {
			continue
		}
		state, err := f.Single.Fetch(f.Owner, f.Repo, number)
		if err != nil {
			return nil, err
		}
		if state.Status != "open" {
			set.PRs[number] = state
		}
	}
	return set, nil
}

func (f *SearchFetcher) warnOut() io.Writer {
	if f.Warn != nil {
		return f.Warn
	}
	return os.Stderr
}
