package pr

import (
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/justincampbell/gh-watch/internal/checks"
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

// node mirrors the prFields fragment.
type node struct {
	Number          int
	Title           string
	State           string
	IsDraft         bool
	Mergeable       string
	IsInMergeQueue  bool
	MergeQueueEntry *struct {
		State    string
		Position int
	}
	AutoMergeRequest *struct {
		MergeMethod string
		EnabledBy   *struct{ Login string }
	}
	Commits struct {
		Nodes []struct {
			Commit struct {
				OID               string `json:"oid"`
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
					}
				}
			}
		}
	}
	Reviews struct {
		Nodes []struct {
			ID     string
			Author struct{ Login string }
			State  string
			Body   string
		}
	}
	Comments struct {
		Nodes []struct {
			ID        string
			Author    struct{ Login string }
			Body      string
			CreatedAt time.Time
		}
	}
}

// fragment selects every field needed to build a State. $last bounds reviews
// and comments so set-mode searches can keep responses small.
const fragment = `fragment prFields on PullRequest {
	number
	title
	state
	isDraft
	mergeable
	isInMergeQueue
	mergeQueueEntry {
		state
		position
	}
	autoMergeRequest {
		mergeMethod
		enabledBy { login }
	}
	commits(last: 1) {
		nodes {
			commit {
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
	reviews(last: $last) {
		nodes {
			id
			author { login }
			state
			body
		}
	}
	comments(last: $last) {
		nodes {
			id
			author { login }
			body
			createdAt
		}
	}
}`

func (f *GraphQLFetcher) Fetch(owner, repo string, number int) (*State, error) {
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("creating GraphQL client: %w", err)
	}

	query := `query PRState($owner: String!, $repo: String!, $number: Int!, $last: Int!) {
		repository(owner: $owner, name: $repo) {
			pullRequest(number: $number) { ...prFields }
		}
	}
	` + fragment

	variables := map[string]interface{}{
		"owner":  owner,
		"repo":   repo,
		"number": number,
		"last":   50,
	}

	var resp struct {
		Repository struct {
			PullRequest node
		}
	}
	if err := client.Do(query, variables, &resp); err != nil {
		return nil, fmt.Errorf("querying PR state: %w", err)
	}

	return resp.Repository.PullRequest.toState(), nil
}

func (pr node) toState() *State {
	state := &State{
		Number:       pr.Number,
		Title:        pr.Title,
		Status:       stateToStatus(pr.State),
		IsDraft:      pr.IsDraft,
		Mergeable:    pr.Mergeable,
		InMergeQueue: pr.IsInMergeQueue || pr.MergeQueueEntry != nil,
	}
	if pr.MergeQueueEntry != nil {
		state.MergeQueueState = pr.MergeQueueEntry.State
		state.MergeQueuePosition = pr.MergeQueueEntry.Position
	}
	if pr.AutoMergeRequest != nil {
		state.AutoMerge = &AutoMerge{MergeMethod: pr.AutoMergeRequest.MergeMethod}
		if pr.AutoMergeRequest.EnabledBy != nil {
			state.AutoMerge.EnabledBy = pr.AutoMergeRequest.EnabledBy.Login
		}
	}

	if len(pr.Commits.Nodes) > 0 {
		state.HeadSHA = pr.Commits.Nodes[0].Commit.OID
		contexts := pr.Commits.Nodes[0].Commit.StatusCheckRollup.Contexts.Nodes
		for _, ctx := range contexts {
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

	for _, r := range pr.Reviews.Nodes {
		state.Reviews = append(state.Reviews, Review{
			ID:     r.ID,
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

	return state
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
