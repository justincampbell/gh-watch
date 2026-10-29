package branch

// State represents a snapshot of a branch's head commit.
type State struct {
	Name            string
	SHA             string
	MessageHeadline string
	Author          string

	// Since and Commits list what landed between a previous head (Since) and
	// this one, oldest first. Empty when there was no previous head, nothing
	// changed, or the branch was rewritten (ForcePushed).
	Since       string
	Commits     []Commit
	ForcePushed bool
}

type Commit struct {
	SHA             string
	MessageHeadline string
	Author          string
}
