package checks

import "testing"

func TestIgnoreFilter(t *testing.T) {
	ig, err := ParseIgnore([]string{"Lint*", "Label Check"})
	if err != nil {
		t.Fatal(err)
	}
	runs := []CheckRun{
		{Name: "Lint / golangci"},
		{Name: "Label Check"},
		{Name: "Label Checks"},
		{Name: "test (1.2)"},
	}
	kept := ig.Filter(runs)
	if len(kept) != 2 || kept[0].Name != "Label Checks" || kept[1].Name != "test (1.2)" {
		t.Errorf("unexpected kept runs: %v", kept)
	}
}
