package search

import (
	"context"
	"fmt"

	"github.com/yorha2B0826/gogetx/internal/cache"
	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

type SourceSearcher interface {
	Search(ctx context.Context, keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error)
}

type Service struct {
	Pkgsite SourceSearcher
	GitHub  SourceSearcher
	Cache   *cache.Store
}

func (s *Service) Search(ctx context.Context, keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error) {
	opts = packageinfo.NormalizeSearchOptions(opts)
	if !opts.NoCache && !opts.Refresh && s.Cache != nil {
		if results, ok, err := s.Cache.Get(keyword, opts); err != nil {
			return nil, err
		} else if ok {
			return results, nil
		}
	}

	results, err := s.searchFresh(ctx, keyword, opts)
	if err != nil {
		return nil, err
	}
	if !opts.NoCache && s.Cache != nil {
		if err := s.Cache.Set(keyword, opts, results); err != nil {
			return nil, err
		}
	}
	return results, nil
}

func (s *Service) searchFresh(ctx context.Context, keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error) {
	switch opts.Source {
	case packageinfo.SourcePkgsite:
		if s.Pkgsite == nil {
			return nil, fmt.Errorf("pkgsite searcher is not configured")
		}
		return s.Pkgsite.Search(ctx, keyword, opts)
	case packageinfo.SourceGitHub:
		if s.GitHub == nil {
			return nil, fmt.Errorf("github searcher is not configured")
		}
		return s.GitHub.Search(ctx, keyword, opts)
	case packageinfo.SourceAll:
		var combined []packageinfo.PackageCandidate
		var firstErr error
		if s.Pkgsite != nil {
			results, err := s.Pkgsite.Search(ctx, keyword, opts)
			if err != nil && firstErr == nil {
				firstErr = err
			}
			combined = append(combined, results...)
		}
		if s.GitHub != nil {
			results, err := s.GitHub.Search(ctx, keyword, opts)
			if err != nil && firstErr == nil {
				firstErr = err
			}
			combined = append(combined, results...)
		}
		if len(combined) == 0 && firstErr != nil {
			return nil, firstErr
		}
		return dedupe(combined), nil
	default:
		return nil, fmt.Errorf("unsupported source %q", opts.Source)
	}
}

func dedupe(results []packageinfo.PackageCandidate) []packageinfo.PackageCandidate {
	seen := map[string]bool{}
	out := make([]packageinfo.PackageCandidate, 0, len(results))
	for _, result := range results {
		key := result.ModulePath + "|" + result.PackagePath
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, result)
	}
	return out
}
