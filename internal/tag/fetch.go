package tag

import (
	"fmt"
	"path"

	"github.com/cli/go-gh/v2/pkg/api"
)

// maxTags bounds how many of the most recent tags (by tag commit date) are
// tracked per poll. If more than this many new tags land between polls, the
// oldest new ones could be missed — acceptable for realistic tagging cadence.
const maxTags = 100

// Fetcher retrieves tag state from GitHub.
type Fetcher interface {
	Fetch(owner, repo, match, contains string) (*State, error)
}

// GraphQLFetcher lists tags via GraphQL and, when a containment target is set,
// resolves ancestry via the REST compare API.
type GraphQLFetcher struct {
	// containsCache memoizes containment by tag commit SHA. The containment
	// target is fixed for the lifetime of a watch, so the SHA is a sufficient key.
	containsCache map[string]bool
}

func NewFetcher() Fetcher {
	return &GraphQLFetcher{containsCache: map[string]bool{}}
}

type graphQLResponse struct {
	Repository struct {
		Refs struct {
			Nodes []struct {
				Name   string `json:"name"`
				Target struct {
					TypeName        string `json:"__typename"`
					OID             string `json:"oid"`
					MessageHeadline string `json:"messageHeadline"`
					// Target is populated for annotated tags (Tag -> Commit).
					Target *struct {
						OID             string `json:"oid"`
						MessageHeadline string `json:"messageHeadline"`
					} `json:"target"`
				} `json:"target"`
			} `json:"nodes"`
		} `json:"refs"`
	} `json:"repository"`
}

func (f *GraphQLFetcher) Fetch(owner, repo, match, contains string) (*State, error) {
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("creating GraphQL client: %w", err)
	}

	query := `query TagState($owner: String!, $repo: String!, $count: Int!) {
		repository(owner: $owner, name: $repo) {
			refs(refPrefix: "refs/tags/", first: $count, orderBy: {field: TAG_COMMIT_DATE, direction: DESC}) {
				nodes {
					name
					target {
						__typename
						... on Commit {
							oid
							messageHeadline
						}
						... on Tag {
							target {
								... on Commit {
									oid
									messageHeadline
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
		"count": maxTags,
	}

	var resp graphQLResponse
	if err := client.Do(query, variables, &resp); err != nil {
		return nil, fmt.Errorf("querying tag state: %w", err)
	}

	state := &State{Match: match, ContainsTarget: contains}

	for _, node := range resp.Repository.Refs.Nodes {
		if match != "" {
			matched, err := path.Match(match, node.Name)
			if err != nil {
				return nil, fmt.Errorf("invalid --match pattern %q: %w", match, err)
			}
			if !matched {
				continue
			}
		}

		// Peel annotated tags (Tag -> Commit) to the underlying commit.
		oid := node.Target.OID
		headline := node.Target.MessageHeadline
		if node.Target.TypeName == "Tag" && node.Target.Target != nil {
			oid = node.Target.Target.OID
			headline = node.Target.Target.MessageHeadline
		}

		t := Tag{
			Name:            node.Name,
			SHA:             oid,
			MessageHeadline: headline,
		}

		if contains != "" {
			c, err := f.tagContains(owner, repo, contains, oid)
			if err != nil {
				return nil, err
			}
			t.Contains = c
		}

		state.Tags = append(state.Tags, t)
	}

	return state, nil
}

// tagContains reports whether tagSHA's history includes target, memoizing results.
func (f *GraphQLFetcher) tagContains(owner, repo, target, tagSHA string) (bool, error) {
	if c, ok := f.containsCache[tagSHA]; ok {
		return c, nil
	}

	client, err := api.DefaultRESTClient()
	if err != nil {
		return false, fmt.Errorf("creating REST client: %w", err)
	}

	var resp struct {
		Status string `json:"status"`
	}
	// base...head: base (target) is an ancestor of head (tag) iff status is
	// "ahead" or "identical". Using commit SHAs avoids slashes from tag names.
	endpoint := fmt.Sprintf("repos/%s/%s/compare/%s...%s", owner, repo, target, tagSHA)
	if err := client.Get(endpoint, &resp); err != nil {
		return false, fmt.Errorf("comparing %s...%s: %w", target, tagSHA, err)
	}

	c := containsStatus(resp.Status)
	f.containsCache[tagSHA] = c
	return c, nil
}

// containsStatus maps a compare API status to whether base is an ancestor of head.
func containsStatus(status string) bool {
	return status == "ahead" || status == "identical"
}
