package branch

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Fetcher retrieves branch state from GitHub. When since is a previous head
// SHA that differs from the current one, the commits in between are listed.
type Fetcher interface {
	Fetch(owner, repo, branch, since string) (*State, error)
}

// GraphQLFetcher uses the GitHub GraphQL API via go-gh.
type GraphQLFetcher struct{}

func NewFetcher() Fetcher {
	return &GraphQLFetcher{}
}

type graphQLResponse struct {
	Repository struct {
		Ref *struct {
			Target struct {
				TypeName        string `json:"__typename"`
				OID             string `json:"oid"`
				MessageHeadline string `json:"messageHeadline"`
				Author          struct {
					User *struct {
						Login string `json:"login"`
					} `json:"user"`
				} `json:"author"`
			} `json:"target"`
		} `json:"ref"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

func (f *GraphQLFetcher) Fetch(owner, repo, branch, since string) (*State, error) {
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("creating GraphQL client: %w", err)
	}

	query := `query BranchState($owner: String!, $repo: String!, $ref: String!) {
		repository(owner: $owner, name: $repo) {
			ref(qualifiedName: $ref) {
				target {
					__typename
					... on Commit {
						oid
						messageHeadline
						author {
							user {
								login
							}
						}
					}
				}
			}
		}
	}`

	variables := map[string]interface{}{
		"owner": owner,
		"repo":  repo,
		"ref":   "refs/heads/" + branch,
	}

	var resp graphQLResponse
	if err := client.Do(query, variables, &resp); err != nil {
		return nil, fmt.Errorf("querying branch state: %w", err)
	}

	if resp.Repository.Ref == nil {
		return nil, fmt.Errorf("branch %q not found", branch)
	}

	if resp.Repository.Ref.Target.TypeName != "Commit" {
		return nil, fmt.Errorf("branch %q target is not a commit (got %s)", branch, resp.Repository.Ref.Target.TypeName)
	}

	state := &State{
		Name:            branch,
		SHA:             resp.Repository.Ref.Target.OID,
		MessageHeadline: resp.Repository.Ref.Target.MessageHeadline,
	}

	if resp.Repository.Ref.Target.Author.User != nil {
		state.Author = resp.Repository.Ref.Target.Author.User.Login
	}

	if since != "" && since != state.SHA {
		if err := listCommitsSince(owner, repo, state, since); err != nil {
			return nil, err
		}
	}

	return state, nil
}

// listCommitsSince fills in the commits between since and the state's head
// via the REST compare API, which returns exactly head ^since, oldest first.
func listCommitsSince(owner, repo string, state *State, since string) error {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("creating REST client: %w", err)
	}

	var resp struct {
		Status       string `json:"status"`
		TotalCommits int    `json:"total_commits"`
		Commits      []struct {
			SHA    string `json:"sha"`
			Commit struct {
				Message string `json:"message"`
			} `json:"commit"`
			Author *struct {
				Login string `json:"login"`
			} `json:"author"`
		} `json:"commits"`
	}
	endpoint := fmt.Sprintf("repos/%s/%s/compare/%s...%s", owner, repo, since, state.SHA)
	if err := client.Get(endpoint, &resp); err != nil {
		return fmt.Errorf("comparing %s...%s: %w", since, state.SHA, err)
	}

	state.Since = since
	// compare caps its commit list at 250; past that, report the head alone.
	if resp.Status != "ahead" || resp.TotalCommits > len(resp.Commits) {
		state.ForcePushed = resp.Status != "ahead"
		return nil
	}
	for _, c := range resp.Commits {
		commit := Commit{SHA: c.SHA, MessageHeadline: strings.SplitN(c.Commit.Message, "\n", 2)[0]}
		if c.Author != nil {
			commit.Author = c.Author.Login
		}
		state.Commits = append(state.Commits, commit)
	}
	return nil
}
