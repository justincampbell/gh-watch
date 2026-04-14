package commit

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/justincampbell/gh-watch/internal/checks"
)

// Fetcher retrieves commit CI state from GitHub.
type Fetcher interface {
	Fetch(owner, repo, sha string) (*State, error)
}

// GraphQLFetcher uses the GitHub GraphQL API via go-gh.
type GraphQLFetcher struct{}

func NewFetcher() Fetcher {
	return &GraphQLFetcher{}
}

type graphQLResponse struct {
	Repository struct {
		Object struct {
			TypeName           string `json:"__typename"`
			OID                string `json:"oid"`
			StatusCheckRollup *struct {
				Contexts struct {
					Nodes []struct {
						TypeName   string `json:"__typename"`
						Name       string
						Status     string
						Conclusion string
						DetailsURL string `json:"detailsUrl"`
						Context    string
						State      string
						TargetURL  string `json:"targetUrl"`
					}
				} `graphql:"contexts(first: 100)"`
			}
		} `graphql:"object(expression: $oid)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

func (f *GraphQLFetcher) Fetch(owner, repo, sha string) (*State, error) {
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("creating GraphQL client: %w", err)
	}

	query := `query CommitStatus($owner: String!, $repo: String!, $oid: String!) {
		repository(owner: $owner, name: $repo) {
			object(expression: $oid) {
				__typename
				... on Commit {
					oid
					statusCheckRollup {
						contexts(first: 100) {
							nodes {
								__typename
								... on CheckRun {
									name
									status
									conclusion
									detailsUrl
								}
								... on StatusContext {
									context
									state
									targetUrl
								}
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
		"oid":   sha,
	}

	var resp graphQLResponse
	if err := client.Do(query, variables, &resp); err != nil {
		return nil, fmt.Errorf("querying commit status: %w", err)
	}

	if resp.Repository.Object.TypeName != "Commit" {
		return nil, fmt.Errorf("object %s is not a commit (got %s)", sha, resp.Repository.Object.TypeName)
	}

	state := &State{
		SHA: resp.Repository.Object.OID,
	}

	if rollup := resp.Repository.Object.StatusCheckRollup; rollup != nil {
		for _, ctx := range rollup.Contexts.Nodes {
			if ctx.TypeName == "CheckRun" {
				state.CheckRuns = append(state.CheckRuns, checks.CheckRun{
					Name:       ctx.Name,
					Status:     ctx.Status,
					Conclusion: ctx.Conclusion,
					URL:        ctx.DetailsURL,
				})
			} else if ctx.TypeName == "StatusContext" {
				state.CheckRuns = append(state.CheckRuns, checks.CheckRun{
					Name:       ctx.Context,
					Status:     checks.StatusContextStateToStatus(ctx.State),
					Conclusion: checks.StatusContextStateToConclusion(ctx.State),
					URL:        ctx.TargetURL,
				})
			}
		}
	}

	return state, nil
}

