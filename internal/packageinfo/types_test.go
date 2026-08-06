package packageinfo

import "testing"

func TestNormalizeSearchOptionsUsesDefaultLimit(t *testing.T) {
	t.Parallel()

	opts := NormalizeSearchOptions(SearchOptions{})
	if opts.Limit != DefaultSearchLimit {
		t.Fatalf("Limit = %d, want default %d", opts.Limit, DefaultSearchLimit)
	}
}

func TestDedupeCandidatesRemovesRepeatsPreservingOrder(t *testing.T) {
	t.Parallel()

	zap := PackageCandidate{PackagePath: "go.uber.org/zap", ModulePath: "go.uber.org/zap"}
	air := PackageCandidate{PackagePath: "github.com/air-verse/air", ModulePath: "github.com/air-verse/air"}

	got := DedupeCandidates([]PackageCandidate{zap, air, zap, air})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0] != zap || got[1] != air {
		t.Fatalf("deduped = %#v, want [zap air] preserving first occurrence", got)
	}

	var empty []PackageCandidate
	if got := DedupeCandidates(empty); len(got) != 0 {
		t.Fatalf("dedupe of empty = %#v, want empty", got)
	}
}

func TestDedupeKeyDistinguishesModuleAndPackage(t *testing.T) {
	t.Parallel()

	a := PackageCandidate{PackagePath: "example.com/root/pkg", ModulePath: "example.com/root"}
	b := PackageCandidate{PackagePath: "example.com/root/pkg", ModulePath: "example.com/root/pkg"}
	if a.DedupeKey() == b.DedupeKey() {
		t.Fatalf("DedupeKey = %q for %#v and %#v, want distinct", a.DedupeKey(), a, b)
	}
}
