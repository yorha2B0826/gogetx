package cache

import (
	"testing"
	"time"

	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

func TestStoreCachesSearchResultsBySourceKeywordAndLimit(t *testing.T) {
	t.Parallel()

	store := NewStore(t.TempDir()+"/search.json", time.Hour)
	opts := packageinfo.SearchOptions{Source: "pkgsite", Limit: 10}
	results := []packageinfo.PackageCandidate{{PackagePath: "go.uber.org/zap", ModulePath: "go.uber.org/zap"}}

	if err := store.Set("zap", opts, results); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	got, ok, err := store.Get("zap", opts)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if len(got) != 1 || got[0].ModulePath != "go.uber.org/zap" {
		t.Fatalf("results = %#v, want cached zap result", got)
	}
}

func TestStoreTreatsExpiredEntriesAsMisses(t *testing.T) {
	t.Parallel()

	store := NewStore(t.TempDir()+"/search.json", -time.Second)
	opts := packageinfo.SearchOptions{Source: "pkgsite", Limit: 10}
	if err := store.Set("zap", opts, []packageinfo.PackageCandidate{{PackagePath: "go.uber.org/zap"}}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	_, ok, err := store.Get("zap", opts)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if ok {
		t.Fatal("ok = true, want false for expired entry")
	}
}
