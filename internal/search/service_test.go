package search

import (
	"context"
	"testing"
	"time"

	"github.com/yorha2B0826/gogetx/internal/cache"
	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

type fakePageSource struct {
	pages []packageinfo.SearchPage
	opts  []packageinfo.SearchOptions
}

func (f *fakePageSource) Search(ctx context.Context, keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error) {
	page, err := f.SearchPage(ctx, keyword, opts)
	if err != nil {
		return nil, err
	}
	return page.Results, nil
}

func (f *fakePageSource) SearchPage(_ context.Context, _ string, opts packageinfo.SearchOptions) (packageinfo.SearchPage, error) {
	f.opts = append(f.opts, opts)
	if len(f.pages) == 0 {
		return packageinfo.SearchPage{}, nil
	}
	page := f.pages[0]
	f.pages = f.pages[1:]
	return page, nil
}

type fakeSingleSource struct {
	results []packageinfo.PackageCandidate
	calls   int
}

func (f *fakeSingleSource) Search(_ context.Context, _ string, _ packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error) {
	f.calls++
	return f.results, nil
}

func TestServiceSearchPageCachesByToken(t *testing.T) {
	t.Parallel()

	store := cache.NewStore(t.TempDir()+"/search.json", time.Hour)
	source := &fakePageSource{pages: []packageinfo.SearchPage{
		{
			Results:       []packageinfo.PackageCandidate{{PackagePath: "example.com/first", ModulePath: "example.com/first"}},
			NextPageToken: "page-2",
			Total:         2,
		},
		{
			Results: []packageinfo.PackageCandidate{{PackagePath: "example.com/second", ModulePath: "example.com/second"}},
			Total:   2,
		},
	}}
	service := &Service{Pkgsite: source, Cache: store}

	first, err := service.SearchPage(context.Background(), "air", packageinfo.SearchOptions{Source: packageinfo.SourcePkgsite, Limit: 1})
	if err != nil {
		t.Fatalf("SearchPage first returned error: %v", err)
	}
	second, err := service.SearchPage(context.Background(), "air", packageinfo.SearchOptions{Source: packageinfo.SourcePkgsite, Limit: 1, PageToken: first.NextPageToken})
	if err != nil {
		t.Fatalf("SearchPage second returned error: %v", err)
	}
	if second.Results[0].PackagePath != "example.com/second" {
		t.Fatalf("second page = %#v, want second result", second)
	}

	source.pages = nil
	cachedSecond, err := service.SearchPage(context.Background(), "air", packageinfo.SearchOptions{Source: packageinfo.SourcePkgsite, Limit: 1, PageToken: "page-2"})
	if err != nil {
		t.Fatalf("SearchPage cached second returned error: %v", err)
	}
	if cachedSecond.Results[0].PackagePath != "example.com/second" {
		t.Fatalf("cached second = %#v, want second result", cachedSecond)
	}
	if len(source.opts) != 2 {
		t.Fatalf("source calls = %d, want 2 before cache hit", len(source.opts))
	}
}

func TestServiceAllSourceUsesGitHubOnlyOnFirstPage(t *testing.T) {
	t.Parallel()

	pkgsite := &fakePageSource{pages: []packageinfo.SearchPage{
		{
			Results:       []packageinfo.PackageCandidate{{PackagePath: "example.com/pkgsite-1", ModulePath: "example.com/pkgsite-1"}},
			NextPageToken: "page-2",
		},
		{
			Results: []packageinfo.PackageCandidate{{PackagePath: "example.com/pkgsite-2", ModulePath: "example.com/pkgsite-2"}},
		},
	}}
	github := &fakeSingleSource{results: []packageinfo.PackageCandidate{{PackagePath: "github.com/example/repo", ModulePath: "github.com/example/repo"}}}
	service := &Service{Pkgsite: pkgsite, GitHub: github}

	first, err := service.SearchPage(context.Background(), "air", packageinfo.SearchOptions{Source: packageinfo.SourceAll, Limit: 1})
	if err != nil {
		t.Fatalf("SearchPage first returned error: %v", err)
	}
	if len(first.Results) != 2 {
		t.Fatalf("first results = %#v, want pkgsite plus github", first.Results)
	}
	if github.calls != 1 {
		t.Fatalf("github calls = %d, want 1 after first page", github.calls)
	}

	second, err := service.SearchPage(context.Background(), "air", packageinfo.SearchOptions{Source: packageinfo.SourceAll, Limit: 1, PageToken: first.NextPageToken})
	if err != nil {
		t.Fatalf("SearchPage second returned error: %v", err)
	}
	if len(second.Results) != 1 || second.Results[0].PackagePath != "example.com/pkgsite-2" {
		t.Fatalf("second results = %#v, want pkgsite second page only", second.Results)
	}
	if github.calls != 1 {
		t.Fatalf("github calls = %d, want no second-page github call", github.calls)
	}
}
