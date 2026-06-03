package packageinfo

import "testing"

func TestNormalizeSearchOptionsUsesDefaultLimit(t *testing.T) {
	t.Parallel()

	opts := NormalizeSearchOptions(SearchOptions{})
	if opts.Limit != DefaultSearchLimit {
		t.Fatalf("Limit = %d, want default %d", opts.Limit, DefaultSearchLimit)
	}
}
