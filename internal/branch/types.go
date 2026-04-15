package branch

// State represents a snapshot of a branch's head commit.
type State struct {
	Name            string
	SHA             string
	MessageHeadline string
	Author          string
}
