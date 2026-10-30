package checks

import (
	"fmt"
	"regexp"
	"strings"
)

// CheckRun represents a CI check or status context on a commit.
type CheckRun struct {
	Name       string
	Status     string // "QUEUED", "IN_PROGRESS", "COMPLETED"
	Conclusion string // "SUCCESS", "FAILURE", "NEUTRAL", "CANCELLED", "TIMED_OUT", "ACTION_REQUIRED", ""
	URL        string

	// FailureConfirmed is set once the same failure has been seen on two
	// consecutive polls of the same head commit.
	FailureConfirmed bool
}

// ConfirmFailures marks every current failure as confirmed. Used when a
// snapshot is first reported, since its failures are already in that report.
func ConfirmFailures(runs []CheckRun) {
	for i := range runs {
		if runs[i].IsFailed() {
			runs[i].FailureConfirmed = true
		}
	}
}

// IsFailed returns true if the check concluded with a failure state.
func (c CheckRun) IsFailed() bool {
	return c.Conclusion == "FAILURE" || c.Conclusion == "TIMED_OUT" || c.Conclusion == "CANCELLED"
}

// Summary holds aggregate CI check counts.
type Summary struct {
	Total   int
	Passed  int
	Failed  int
	Pending int
}

// Summarize counts check states across a slice of CheckRuns.
func Summarize(runs []CheckRun) Summary {
	s := Summary{Total: len(runs)}
	for _, c := range runs {
		if c.Status == "COMPLETED" {
			if c.IsFailed() {
				s.Failed++
			} else {
				s.Passed++
			}
		} else {
			s.Pending++
		}
	}
	return s
}

// BuildMap creates a lookup map from check name to CheckRun.
func BuildMap(runs []CheckRun) map[string]CheckRun {
	m := make(map[string]CheckRun, len(runs))
	for _, c := range runs {
		m[c.Name] = c
	}
	return m
}

// StatusContextStateToStatus maps a GitHub StatusContext state to a check status.
func StatusContextStateToStatus(s string) string {
	switch s {
	case "PENDING", "EXPECTED":
		return "IN_PROGRESS"
	default:
		return "COMPLETED"
	}
}

// StatusContextStateToConclusion maps a GitHub StatusContext state to a check conclusion.
func StatusContextStateToConclusion(s string) string {
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

// Ignore matches check names to drop from snapshots.
type Ignore []*regexp.Regexp

// ParseIgnore compiles globs where `*` matches any run of characters,
// including the `/` in "Workflow / job" names, and `?` matches one.
func ParseIgnore(globs []string) (Ignore, error) {
	var ig Ignore
	for _, g := range globs {
		pattern := regexp.QuoteMeta(g)
		pattern = strings.ReplaceAll(pattern, `\*`, `.*`)
		pattern = strings.ReplaceAll(pattern, `\?`, `.`)
		re, err := regexp.Compile("^" + pattern + "$")
		if err != nil {
			return nil, fmt.Errorf("invalid --ignore-check %q: %w", g, err)
		}
		ig = append(ig, re)
	}
	return ig, nil
}

// Filter returns runs whose names match no ignore pattern.
func (ig Ignore) Filter(runs []CheckRun) []CheckRun {
	if len(ig) == 0 {
		return runs
	}
	var kept []CheckRun
	for _, r := range runs {
		if !ig.matches(r.Name) {
			kept = append(kept, r)
		}
	}
	return kept
}

func (ig Ignore) matches(name string) bool {
	for _, re := range ig {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}
