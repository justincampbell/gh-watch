package checks

// CheckRun represents a CI check or status context on a commit.
type CheckRun struct {
	Name       string
	Status     string // "QUEUED", "IN_PROGRESS", "COMPLETED"
	Conclusion string // "SUCCESS", "FAILURE", "NEUTRAL", "CANCELLED", "TIMED_OUT", "ACTION_REQUIRED", ""
	URL        string
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
