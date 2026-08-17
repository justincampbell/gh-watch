package tag

// State represents a snapshot of a repository's tags, most-recent-first.
type State struct {
	Match          string // glob filter; empty = all tags
	ContainsTarget string // commit SHA to filter by; empty = no filter
	Tags           []Tag
}

// Tag is a single tag and its target commit.
type Tag struct {
	Name            string
	SHA             string // target commit OID (peeled through annotated tags)
	MessageHeadline string
	Contains        bool // whether SHA contains ContainsTarget; meaningful only when ContainsTarget != ""
}
