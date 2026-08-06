package cache

import (
	"encoding/json"
	"os"
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

func TestStoreCachesSearchPagesByPageToken(t *testing.T) {
	t.Parallel()

	store := NewStore(t.TempDir()+"/search.json", time.Hour)
	opts := packageinfo.SearchOptions{Source: "pkgsite", Limit: 10}
	firstPage := packageinfo.SearchPage{
		Results:       []packageinfo.PackageCandidate{{PackagePath: "example.com/first", ModulePath: "example.com/first"}},
		NextPageToken: "page-2",
		Total:         20,
	}
	secondPageOpts := opts
	secondPageOpts.PageToken = "page-2"
	secondPage := packageinfo.SearchPage{
		Results: []packageinfo.PackageCandidate{{PackagePath: "example.com/second", ModulePath: "example.com/second"}},
		Total:   20,
	}

	if err := store.SetPage("air", opts, firstPage); err != nil {
		t.Fatalf("SetPage first returned error: %v", err)
	}
	if err := store.SetPage("air", secondPageOpts, secondPage); err != nil {
		t.Fatalf("SetPage second returned error: %v", err)
	}

	gotFirst, ok, err := store.GetPage("air", opts)
	if err != nil {
		t.Fatalf("GetPage first returned error: %v", err)
	}
	if !ok {
		t.Fatal("ok = false, want cached first page")
	}
	if gotFirst.NextPageToken != "page-2" || gotFirst.Total != 20 {
		t.Fatalf("first page metadata = %#v, want next token and total", gotFirst)
	}
	if gotFirst.Results[0].PackagePath != "example.com/first" {
		t.Fatalf("first result = %#v, want first page", gotFirst.Results)
	}

	gotSecond, ok, err := store.GetPage("air", secondPageOpts)
	if err != nil {
		t.Fatalf("GetPage second returned error: %v", err)
	}
	if !ok {
		t.Fatal("ok = false, want cached second page")
	}
	if gotSecond.NextPageToken != "" {
		t.Fatalf("second NextPageToken = %q, want empty", gotSecond.NextPageToken)
	}
	if gotSecond.Results[0].PackagePath != "example.com/second" {
		t.Fatalf("second result = %#v, want second page", gotSecond.Results)
	}
}

func TestStoreTreatsCorruptCacheAsMissAndSelfHeals(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/search.json"
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	store := NewStore(path, time.Hour)
	opts := packageinfo.SearchOptions{Source: "pkgsite", Limit: 10}

	_, ok, err := store.Get("zap", opts)
	if err != nil {
		t.Fatalf("Get returned error on corrupt cache: %v", err)
	}
	if ok {
		t.Fatal("ok = true, want miss for corrupt cache")
	}

	results := []packageinfo.PackageCandidate{{PackagePath: "go.uber.org/zap", ModulePath: "go.uber.org/zap"}}
	if err := store.Set("zap", opts, results); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	got, ok, err := store.Get("zap", opts)
	if err != nil {
		t.Fatalf("Get returned error after self-heal: %v", err)
	}
	if !ok {
		t.Fatal("ok = false, want hit after self-heal")
	}
	if len(got) != 1 || got[0].ModulePath != "go.uber.org/zap" {
		t.Fatalf("results = %#v, want recovered zap result", got)
	}
}

func TestStorePrunesExpiredEntriesOnWrite(t *testing.T) {
	t.Parallel()

	store := NewStore(t.TempDir()+"/search.json", time.Hour)
	staleOpts := packageinfo.SearchOptions{Source: "pkgsite", Limit: 10}
	freshOpts := packageinfo.SearchOptions{Source: "github", Limit: 10}

	// Seed an expired entry directly, as if written by an older run.
	stale := cacheFile{Entries: map[string]SearchCacheEntry{
		cacheKey("old", staleOpts): {
			Keyword:   "old",
			Results:   []packageinfo.PackageCandidate{{PackagePath: "example.com/old"}},
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
	}}
	if err := store.write(stale); err != nil {
		t.Fatalf("write returned error: %v", err)
	}

	if err := store.Set("fresh", freshOpts, []packageinfo.PackageCandidate{{PackagePath: "example.com/fresh"}}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	content, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	var got cacheFile
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if _, ok := got.Entries[cacheKey("old", staleOpts)]; ok {
		t.Fatal("expired entry was not pruned from cache")
	}
	if _, ok := got.Entries[cacheKey("fresh", freshOpts)]; !ok {
		t.Fatal("fresh entry is missing after prune")
	}
}
