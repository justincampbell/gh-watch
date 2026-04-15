package branch

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Fetcher retrieves branch state from GitHub.
type Fetcher interface {
	Fetch(owner, repo, branch string) (*State, error)
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

func (f *GraphQLFetcher) Fetch(owner, repo, branch string) (*State, error) {
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

	return state, nil
}
