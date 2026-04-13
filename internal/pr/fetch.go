package pr

import (
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Fetcher retrieves PR state from GitHub.
type Fetcher interface {
	Fetch(owner, repo string, number int) (*State, error)
}

// GraphQLFetcher uses the GitHub GraphQL API via go-gh.
type GraphQLFetcher struct{}

func NewFetcher() Fetcher {
	return &GraphQLFetcher{}
}

type graphQLResponse struct {
	Repository struct {
		PullRequest struct {
			Number    int
			Title     string
			State     string
			Mergeable string
			Commits   struct {
				Nodes []struct {
					Commit struct {
						StatusCheckRollup struct {
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
					}
				}
			} `graphql:"commits(last: 1)"`
			Reviews struct {
				Nodes []struct {
					Author struct {
						Login string
					}
					State string
					Body  string
				}
			} `graphql:"reviews(last: 50)"`
			Comments struct {
				Nodes []struct {
					ID        string
					Author    struct{ Login string }
					Body      string
					CreatedAt time.Time
				}
			} `graphql:"comments(last: 50)"`
		} `graphql:"pullRequest(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

func (f *GraphQLFetcher) Fetch(owner, repo string, number int) (*State, error) {
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("creating GraphQL client: %w", err)
	}

	query := `query PRState($owner: String!, $repo: String!, $number: Int!) {
		repository(owner: $owner, name: $repo) {
			pullRequest(number: $number) {
				number
				title
				state
				mergeable
				commits(last: 1) {
					nodes {
						commit {
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
				reviews(last: 50) {
					nodes {
						author { login }
						state
						body
					}
				}
				comments(last: 50) {
					nodes {
						id
						author { login }
						body
						createdAt
					}
				}
			}
		}
	}`

	variables := map[string]interface{}{
		"owner":  owner,
		"repo":   repo,
		"number": number,
	}

	var resp graphQLResponse
	if err := client.Do(query, variables, &resp); err != nil {
		return nil, fmt.Errorf("querying PR state: %w", err)
	}

	pr := resp.Repository.PullRequest

	state := &State{
		Number:    pr.Number,
		Title:     pr.Title,
		Status:    stateToStatus(pr.State),
		Mergeable: pr.Mergeable,
		FetchedAt: time.Now(),
	}

	if len(pr.Commits.Nodes) > 0 {
		contexts := pr.Commits.Nodes[0].Commit.StatusCheckRollup.Contexts.Nodes
		for _, ctx := range contexts {
			if ctx.TypeName == "CheckRun" {
				state.CheckRuns = append(state.CheckRuns, CheckRun{
					Name:       ctx.Name,
					Status:     ctx.Status,
					Conclusion: ctx.Conclusion,
					URL:        ctx.DetailsURL,
				})
			} else if ctx.TypeName == "StatusContext" {
				state.CheckRuns = append(state.CheckRuns, CheckRun{
					Name:       ctx.Context,
					Status:     statusContextStateToStatus(ctx.State),
					Conclusion: statusContextStateToConclusion(ctx.State),
					URL:        ctx.TargetURL,
				})
			}
		}
	}

	for _, r := range pr.Reviews.Nodes {
		state.Reviews = append(state.Reviews, Review{
			Author: r.Author.Login,
			State:  r.State,
			Body:   r.Body,
		})
	}

	for _, c := range pr.Comments.Nodes {
		state.Comments = append(state.Comments, Comment{
			ID:        c.ID,
			Author:    c.Author.Login,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
		})
	}

	return state, nil
}

func stateToStatus(s string) string {
	switch s {
	case "MERGED":
		return "merged"
	case "CLOSED":
		return "closed"
	default:
		return "open"
	}
}

func statusContextStateToStatus(s string) string {
	switch s {
	case "PENDING", "EXPECTED":
		return "IN_PROGRESS"
	default:
		return "COMPLETED"
	}
}

func statusContextStateToConclusion(s string) string {
	switch s {
	case "SUCCESS":
		return "SUCCESS"
	case "FAILURE", "ERROR":
		return "FAILURE"
	case "PENDING", "EXPECTED":
		return ""
	default:
		return s
	}
}
