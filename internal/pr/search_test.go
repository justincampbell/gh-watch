package pr

import "testing"

func TestValidateSearch(t *testing.T) {
	for _, q := range []string{"author:@me is:open", "head:feature-x label:bug", ""} {
		if err := ValidateSearch(q); err != nil {
			t.Errorf("ValidateSearch(%q) = %v, want nil", q, err)
		}
	}
	for _, q := range []string{"repo:cli/cli is:open", "is:open org:github", "-user:foo"} {
		if err := ValidateSearch(q); err == nil {
			t.Errorf("ValidateSearch(%q) = nil, want error", q)
		}
	}
}
