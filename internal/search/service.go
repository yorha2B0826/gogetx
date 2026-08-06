package search

import (
	"context"
	"fmt"
	"sync"

	"github.com/yorha2B0826/gogetx/internal/cache"
	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

type SourceSearcher interface {
	Search(ctx context.Context, keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error)
}

type SourcePageSearcher interface {
	SearchPage(ctx context.Context, keyword string, opts packageinfo.SearchOptions) (packageinfo.SearchPage, error)
}

type Service struct {
	Pkgsite SourceSearcher
	GitHub  SourceSearcher
	Cache   *cache.Store
}

func (s *Service) Search(ctx context.Context, keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error) {
	page, err := s.SearchPage(ctx, keyword, opts)
	if err != nil {
		return nil, err
	}
	return page.Results, nil
}

func (s *Service) SearchPage(ctx context.Context, keyword string, opts packageinfo.SearchOptions) (packageinfo.SearchPage, error) {
	opts = packageinfo.NormalizeSearchOptions(opts)
	if !opts.NoCache && !opts.Refresh && s.Cache != nil {
		if page, ok, err := s.Cache.GetPage(keyword, opts); err != nil {
			return packageinfo.SearchPage{}, err
		} else if ok {
			return page, nil
		}
	}

	page, err := s.searchFreshPage(ctx, keyword, opts)
	if err != nil {
		return packageinfo.SearchPage{}, err
	}
	if !opts.NoCache && s.Cache != nil {
		if err := s.Cache.SetPage(keyword, opts, page); err != nil {
			return packageinfo.SearchPage{}, err
		}
	}
	return page, nil
}

func (s *Service) searchFresh(ctx context.Context, keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error) {
	page, err := s.searchFreshPage(ctx, keyword, opts)
	if err != nil {
		return nil, err
	}
	return page.Results, nil
}

func (s *Service) searchFreshPage(ctx context.Context, keyword string, opts packageinfo.SearchOptions) (packageinfo.SearchPage, error) {
	switch opts.Source {
	case packageinfo.SourcePkgsite:
		if s.Pkgsite == nil {
			return packageinfo.SearchPage{}, fmt.Errorf("pkgsite searcher is not configured")
		}
		return searchPage(ctx, s.Pkgsite, keyword, opts)
	case packageinfo.SourceGitHub:
		if s.GitHub == nil {
			return packageinfo.SearchPage{}, fmt.Errorf("github searcher is not configured")
		}
		return searchPage(ctx, s.GitHub, keyword, opts)
	case packageinfo.SourceAll:
		var pkgsitePage packageinfo.SearchPage
		var pkgsiteErr error
		var githubResults []packageinfo.PackageCandidate
		var githubErr error

		var wg sync.WaitGroup
		if s.Pkgsite != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				pkgsitePage, pkgsiteErr = searchPage(ctx, s.Pkgsite, keyword, opts)
			}()
		}
		if opts.PageToken == "" && s.GitHub != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				githubResults, githubErr = s.GitHub.Search(ctx, keyword, opts)
			}()
		}
		wg.Wait()

		var firstErr error
		if pkgsiteErr != nil {
			firstErr = pkgsiteErr
		} else if githubErr != nil {
			firstErr = githubErr
		}
		combined := make([]packageinfo.PackageCandidate, 0, len(pkgsitePage.Results)+len(githubResults))
		combined = append(combined, pkgsitePage.Results...)
		combined = append(combined, githubResults...)
		if len(combined) == 0 && firstErr != nil {
			return packageinfo.SearchPage{}, firstErr
		}
		return packageinfo.SearchPage{
			Results:       packageinfo.DedupeCandidates(combined),
			NextPageToken: pkgsitePage.NextPageToken,
			Total:         pkgsitePage.Total,
		}, nil
	default:
		return packageinfo.SearchPage{}, fmt.Errorf("unsupported source %q", opts.Source)
	}
}

func searchPage(ctx context.Context, searcher SourceSearcher, keyword string, opts packageinfo.SearchOptions) (packageinfo.SearchPage, error) {
	if paged, ok := searcher.(SourcePageSearcher); ok {
		return paged.SearchPage(ctx, keyword, opts)
	}
	results, err := searcher.Search(ctx, keyword, opts)
	if err != nil {
		return packageinfo.SearchPage{}, err
	}
	return packageinfo.SearchPage{Results: results}, nil
}
