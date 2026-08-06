package cmd

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/yorha2B0826/gogetx/internal/browser"
	"github.com/yorha2B0826/gogetx/internal/cache"
	"github.com/yorha2B0826/gogetx/internal/config"
	"github.com/yorha2B0826/gogetx/internal/github"
	"github.com/yorha2B0826/gogetx/internal/goproxy"
	"github.com/yorha2B0826/gogetx/internal/packageinfo"
	"github.com/yorha2B0826/gogetx/internal/pkgsite"
	"github.com/yorha2B0826/gogetx/internal/resolver"
	"github.com/yorha2B0826/gogetx/internal/runner"
	searchsvc "github.com/yorha2B0826/gogetx/internal/search"
	"github.com/yorha2B0826/gogetx/internal/ui"
)

type Command = cobra.Command

type Searcher interface {
	Search(ctx context.Context, keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error)
}

type PagedSearcher interface {
	SearchPage(ctx context.Context, keyword string, opts packageinfo.SearchOptions) (packageinfo.SearchPage, error)
}

type Resolver interface {
	Resolve(ctx context.Context, candidate packageinfo.PackageCandidate) (string, error)
}

type Runner interface {
	Get(ctx context.Context, modulePath string, version string) error
	ModTidy(ctx context.Context) error
	IsInsideModule(ctx context.Context) (bool, error)
}

type Favorites interface {
	Favorite(alias string) (string, bool, error)
	Favorites() (map[string]string, error)
	AddFavorite(alias string, modulePath string) error
	RemoveFavorite(alias string) error
}

type Selector interface {
	Select(results []packageinfo.PackageCandidate) (packageinfo.PackageCandidate, error)
	Confirm(message string) (bool, error)
}

type VersionSource interface {
	Latest(ctx context.Context, modulePath string) (goproxy.VersionInfo, error)
	ListVersions(ctx context.Context, modulePath string) ([]string, error)
}

type Opener interface {
	Open(ctx context.Context, url string) error
}

type App struct {
	Searcher  Searcher
	Resolver  Resolver
	Runner    Runner
	Favorites Favorites
	Selector  Selector
	Latest    VersionSource
	Opener    Opener
}

func NewDefaultApp() *App {
	goRunner := runner.New()
	proxyClient := goproxy.NewClient()
	return &App{
		Searcher: &searchsvc.Service{
			Pkgsite: pkgsite.NewClient(),
			GitHub:  github.NewClient(),
			Cache:   cache.NewStore(defaultCachePath(), 24*time.Hour),
		},
		Resolver:  resolver.New(proxyClient),
		Runner:    goRunner,
		Favorites: config.NewManager(defaultConfigPath()),
		Selector:  ui.New(),
		Latest:    proxyClient,
		Opener:    browser.New(),
	}
}

func defaultCachePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(".gogetx", "cache", "search.json")
	}
	return filepath.Join(dir, "gogetx", "search.json")
}

func defaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".gogetx", "config.yaml")
	}
	return filepath.Join(dir, "gogetx", "config.yaml")
}
