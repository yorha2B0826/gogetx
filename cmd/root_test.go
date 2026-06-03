package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/yorha2B0826/gogetx/internal/goproxy"
	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

type fakeSearcher struct {
	results []packageinfo.PackageCandidate
	called  bool
	keyword string
	opts    packageinfo.SearchOptions
	search  func(keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error)
}

func (f *fakeSearcher) Search(_ context.Context, keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error) {
	f.called = true
	f.keyword = keyword
	f.opts = opts
	if f.search != nil {
		return f.search(keyword, opts)
	}
	return f.results, nil
}

type fakeResolver struct {
	modulePath    string
	called        bool
	lastCandidate packageinfo.PackageCandidate
}

func (f *fakeResolver) Resolve(_ context.Context, candidate packageinfo.PackageCandidate) (string, error) {
	f.called = true
	f.lastCandidate = candidate
	return f.modulePath, nil
}

type fakeRunner struct {
	inside       bool
	getCalled    bool
	tidyCalled   bool
	listVersions []string
	getModule    string
	getVersion   string
	listModule   string
}

func (f *fakeRunner) Get(_ context.Context, modulePath, version string) error {
	f.getCalled = true
	f.getModule = modulePath
	f.getVersion = version
	return nil
}

func (f *fakeRunner) ModTidy(_ context.Context) error {
	f.tidyCalled = true
	return nil
}

func (f *fakeRunner) ListVersions(_ context.Context, modulePath string) ([]string, error) {
	f.listModule = modulePath
	return f.listVersions, nil
}

func (f *fakeRunner) IsInsideModule(_ context.Context) (bool, error) {
	return f.inside, nil
}

type fakeFavorites struct {
	values map[string]string
}

func (f fakeFavorites) Favorite(alias string) (string, bool, error) {
	value, ok := f.values[alias]
	return value, ok, nil
}

func (f fakeFavorites) Favorites() (map[string]string, error) {
	out := make(map[string]string, len(f.values))
	for key, value := range f.values {
		out[key] = value
	}
	return out, nil
}

func (f fakeFavorites) AddFavorite(_, _ string) error { return nil }
func (f fakeFavorites) RemoveFavorite(_ string) error { return nil }

type fakeSelector struct {
	selectedIndex  int
	selectCalled   bool
	confirmCalled  bool
	confirmMessage string
	selectFunc     func(results []packageinfo.PackageCandidate) (packageinfo.PackageCandidate, error)
}

func (f *fakeSelector) Select(results []packageinfo.PackageCandidate) (packageinfo.PackageCandidate, error) {
	if len(results) == 0 {
		return packageinfo.PackageCandidate{}, errors.New("no results")
	}
	f.selectCalled = true
	if f.selectFunc != nil {
		return f.selectFunc(results)
	}
	if f.selectedIndex >= len(results) {
		return packageinfo.PackageCandidate{}, errors.New("selected index out of range")
	}
	return results[f.selectedIndex], nil
}

func (f *fakeSelector) Confirm(message string) (bool, error) {
	f.confirmCalled = true
	f.confirmMessage = message
	return true, nil
}

type fakeLatest struct {
	module string
}

func (f *fakeLatest) Latest(_ context.Context, modulePath string) (goproxy.VersionInfo, error) {
	f.module = modulePath
	return goproxy.VersionInfo{Version: "v1.28.0"}, nil
}

type fakeOpener struct {
	url string
}

func (f *fakeOpener) Open(_ context.Context, url string) error {
	f.url = url
	return nil
}

func executeCommand(root *Command, args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return stdout.String(), stderr.String(), err
}

func testApp(searcher *fakeSearcher, runner *fakeRunner) *App {
	return testAppWithSelector(searcher, runner, &fakeSelector{})
}

func testAppWithSelector(searcher *fakeSearcher, runner *fakeRunner, selector *fakeSelector) *App {
	resolver := &fakeResolver{modulePath: "go.uber.org/zap"}
	return &App{
		Searcher:  searcher,
		Resolver:  resolver,
		Runner:    runner,
		Favorites: fakeFavorites{values: map[string]string{}},
		Selector:  selector,
		Latest:    &fakeLatest{},
		Opener:    &fakeOpener{},
	}
}

func TestSearchJSON(t *testing.T) {
	t.Parallel()

	searcher := &fakeSearcher{results: []packageinfo.PackageCandidate{{
		PackagePath: "go.uber.org/zap",
		ModulePath:  "go.uber.org/zap",
		Version:     "v1.28.0",
		Source:      "pkgsite",
	}}}
	root := NewRootCommand(testApp(searcher, &fakeRunner{inside: true}))

	stdout, _, err := executeCommand(root, "search", "zap", "--json")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	var got []packageinfo.PackageCandidate
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not JSON: %q", stdout)
	}
	if len(got) != 1 || got[0].ModulePath != "go.uber.org/zap" {
		t.Fatalf("results = %#v, want zap result", got)
	}
}

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	root := NewRootCommand(testApp(&fakeSearcher{}, &fakeRunner{inside: true}))

	stdout, _, err := executeCommand(root, "version")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.HasPrefix(stdout, "gogetx ") {
		t.Fatalf("stdout = %q, want gogetx version output", stdout)
	}
	if !strings.HasSuffix(stdout, "\n") {
		t.Fatalf("stdout = %q, want trailing newline", stdout)
	}
}

func TestRootVersionFlag(t *testing.T) {
	t.Parallel()

	root := NewRootCommand(testApp(&fakeSearcher{}, &fakeRunner{inside: true}))

	stdout, _, err := executeCommand(root, "--version")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.HasPrefix(stdout, "gogetx ") {
		t.Fatalf("stdout = %q, want gogetx version output", stdout)
	}
}

func TestSearchFlagsArePassedToSearcher(t *testing.T) {
	t.Parallel()

	searcher := &fakeSearcher{results: []packageinfo.PackageCandidate{{
		PackagePath: "github.com/spf13/cobra",
		ModulePath:  "github.com/spf13/cobra",
		Source:      packageinfo.SourceGitHub,
	}}}
	root := NewRootCommand(testApp(searcher, &fakeRunner{inside: true}))

	_, _, err := executeCommand(root, "search", "cobra", "--limit", "3", "--source", "github", "--no-cache", "--refresh")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if searcher.keyword != "cobra" {
		t.Fatalf("keyword = %q, want cobra", searcher.keyword)
	}
	if searcher.opts.Limit != 3 || searcher.opts.Source != packageinfo.SourceGitHub || !searcher.opts.NoCache || !searcher.opts.Refresh {
		t.Fatalf("opts = %#v, want all search flags passed through", searcher.opts)
	}
}

func TestSearchUsesDefaultLimit(t *testing.T) {
	t.Parallel()

	searcher := &fakeSearcher{results: []packageinfo.PackageCandidate{{
		PackagePath: "go.uber.org/zap",
		ModulePath:  "go.uber.org/zap",
	}}}
	root := NewRootCommand(testApp(searcher, &fakeRunner{inside: true}))

	_, _, err := executeCommand(root, "search", "zap")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if searcher.opts.Limit != packageinfo.DefaultSearchLimit {
		t.Fatalf("limit = %d, want default %d", searcher.opts.Limit, packageinfo.DefaultSearchLimit)
	}
}

func TestAddDryRunDoesNotRunGoGet(t *testing.T) {
	t.Parallel()

	searcher := &fakeSearcher{results: []packageinfo.PackageCandidate{{
		PackagePath: "go.uber.org/zap",
		ModulePath:  "go.uber.org/zap",
	}}}
	runner := &fakeRunner{inside: true}
	root := NewRootCommand(testApp(searcher, runner))

	stdout, _, err := executeCommand(root, "add", "zap", "--dry-run", "--yes")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(stdout, "go get go.uber.org/zap@latest") {
		t.Fatalf("stdout = %q, want dry-run go get command", stdout)
	}
	if runner.getCalled || runner.tidyCalled {
		t.Fatalf("runner called during dry-run: get=%v tidy=%v", runner.getCalled, runner.tidyCalled)
	}
}

func TestAddCanLoadMoreInteractiveResults(t *testing.T) {
	t.Parallel()

	var limits []int
	target := packageinfo.PackageCandidate{
		PackagePath: "github.com/air-verse/air",
		ModulePath:  "github.com/air-verse/air",
	}
	searcher := &fakeSearcher{
		search: func(_ string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error) {
			limits = append(limits, opts.Limit)
			count := opts.Limit
			if count > 36 {
				count = 36
			}
			results := make([]packageinfo.PackageCandidate, 0, count)
			for i := 0; i < count; i++ {
				results = append(results, packageinfo.PackageCandidate{
					PackagePath: fmt.Sprintf("example.com/pkg%d", i+1),
					ModulePath:  fmt.Sprintf("example.com/pkg%d", i+1),
				})
			}
			if opts.Limit >= 50 {
				results[len(results)-1] = target
			}
			return results, nil
		},
	}
	selector := &fakeSelector{
		selectFunc: func(results []packageinfo.PackageCandidate) (packageinfo.PackageCandidate, error) {
			for _, result := range results {
				if isLoadMoreCandidate(result) {
					return result, nil
				}
			}
			for _, result := range results {
				if result.PackagePath == target.PackagePath {
					return result, nil
				}
			}
			return packageinfo.PackageCandidate{}, errors.New("target result not found")
		},
	}
	root := NewRootCommand(&App{
		Searcher:  searcher,
		Resolver:  &fakeResolver{modulePath: target.ModulePath},
		Runner:    &fakeRunner{inside: true},
		Favorites: fakeFavorites{values: map[string]string{}},
		Selector:  selector,
		Latest:    &fakeLatest{},
		Opener:    &fakeOpener{},
	})

	stdout, _, err := executeCommand(root, "add", "air", "--dry-run", "--yes", "--no-cache")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(limits) != 2 || limits[0] != packageinfo.DefaultSearchLimit || limits[1] != packageinfo.DefaultSearchLimit+searchLimitStep {
		t.Fatalf("limits = %v, want [30 50]", limits)
	}
	if !strings.Contains(stdout, "Loading up to 50 results") {
		t.Fatalf("stdout = %q, want load more message", stdout)
	}
	if !strings.Contains(stdout, "go get github.com/air-verse/air@latest") {
		t.Fatalf("stdout = %q, want selected air command", stdout)
	}
}

func TestAddSearchFlagsArePassedToSearcher(t *testing.T) {
	t.Parallel()

	searcher := &fakeSearcher{results: []packageinfo.PackageCandidate{{
		PackagePath: "github.com/spf13/cobra",
		ModulePath:  "github.com/spf13/cobra",
		Source:      packageinfo.SourceGitHub,
	}}}
	runner := &fakeRunner{inside: true}
	root := NewRootCommand(testApp(searcher, runner))

	_, _, err := executeCommand(root, "add", "cobra", "--dry-run", "--first", "--yes", "--limit", "4", "--source", "all", "--no-cache", "--refresh")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if searcher.keyword != "cobra" {
		t.Fatalf("keyword = %q, want cobra", searcher.keyword)
	}
	if searcher.opts.Limit != 4 || searcher.opts.Source != packageinfo.SourceAll || !searcher.opts.NoCache || !searcher.opts.Refresh {
		t.Fatalf("opts = %#v, want all add search flags passed through", searcher.opts)
	}
}

func TestAddYesDoesNotSelectFirstResult(t *testing.T) {
	t.Parallel()

	searcher := &fakeSearcher{results: []packageinfo.PackageCandidate{
		{PackagePath: "example.com/first", ModulePath: "example.com/first"},
		{PackagePath: "go.uber.org/zap", ModulePath: "go.uber.org/zap"},
	}}
	runner := &fakeRunner{inside: true}
	selector := &fakeSelector{selectedIndex: 1}
	root := NewRootCommand(testAppWithSelector(searcher, runner, selector))

	stdout, _, err := executeCommand(root, "add", "zap", "--dry-run", "--yes")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !selector.selectCalled {
		t.Fatal("selector was not called; --yes should not choose the first result")
	}
	if selector.confirmCalled {
		t.Fatal("confirm was called; --yes should skip confirmation")
	}
	if strings.Contains(stdout, "example.com/first") {
		t.Fatalf("stdout = %q, want selected package instead of first result", stdout)
	}
	if !strings.Contains(stdout, "go get go.uber.org/zap@latest") {
		t.Fatalf("stdout = %q, want selected dry-run command", stdout)
	}
}

func TestAddFirstYesSelectsFirstResultAndSkipsConfirmation(t *testing.T) {
	t.Parallel()

	searcher := &fakeSearcher{results: []packageinfo.PackageCandidate{
		{PackagePath: "go.uber.org/zap", ModulePath: "go.uber.org/zap"},
		{PackagePath: "example.com/second", ModulePath: "example.com/second"},
	}}
	runner := &fakeRunner{inside: true}
	selector := &fakeSelector{selectedIndex: 1}
	root := NewRootCommand(testAppWithSelector(searcher, runner, selector))

	stdout, _, err := executeCommand(root, "add", "zap", "--dry-run", "--first", "--yes")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if selector.selectCalled {
		t.Fatal("selector was called; --first should choose the first result")
	}
	if selector.confirmCalled {
		t.Fatal("confirm was called; --yes should skip confirmation")
	}
	if !strings.Contains(stdout, "Selected: go.uber.org/zap") {
		t.Fatalf("stdout = %q, want first result selection", stdout)
	}
	if runner.getCalled || runner.tidyCalled {
		t.Fatalf("runner called during dry-run: get=%v tidy=%v", runner.getCalled, runner.tidyCalled)
	}
}

func TestAddDoesNotRunTidyByDefault(t *testing.T) {
	t.Parallel()

	searcher := &fakeSearcher{results: []packageinfo.PackageCandidate{{
		PackagePath: "go.uber.org/zap",
		ModulePath:  "go.uber.org/zap",
	}}}
	runner := &fakeRunner{inside: true}
	root := NewRootCommand(testApp(searcher, runner))

	_, _, err := executeCommand(root, "add", "zap", "--first", "--yes")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !runner.getCalled {
		t.Fatal("runner Get was not called")
	}
	if runner.tidyCalled {
		t.Fatal("runner ModTidy was called; add should preserve newly added dependencies by default")
	}
}

func TestAddRunsTidyWhenRequested(t *testing.T) {
	t.Parallel()

	searcher := &fakeSearcher{results: []packageinfo.PackageCandidate{{
		PackagePath: "go.uber.org/zap",
		ModulePath:  "go.uber.org/zap",
	}}}
	runner := &fakeRunner{inside: true}
	root := NewRootCommand(testApp(searcher, runner))

	_, _, err := executeCommand(root, "add", "zap", "--first", "--yes", "--tidy")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !runner.getCalled {
		t.Fatal("runner Get was not called")
	}
	if !runner.tidyCalled {
		t.Fatal("runner ModTidy was not called")
	}
}

func TestAddConfirmationMessageDoesNotIncludeTrailingQuestionMark(t *testing.T) {
	t.Parallel()

	searcher := &fakeSearcher{results: []packageinfo.PackageCandidate{{
		PackagePath: "go.uber.org/zap",
		ModulePath:  "go.uber.org/zap",
	}}}
	runner := &fakeRunner{inside: true}
	selector := &fakeSelector{}
	root := NewRootCommand(testAppWithSelector(searcher, runner, selector))

	_, _, err := executeCommand(root, "add", "zap", "--first", "--no-cache")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if selector.confirmMessage != "Run go get go.uber.org/zap@latest" {
		t.Fatalf("confirmMessage = %q, want command without trailing question mark", selector.confirmMessage)
	}
}

func TestAddRequiresGoModule(t *testing.T) {
	t.Parallel()

	searcher := &fakeSearcher{results: []packageinfo.PackageCandidate{{PackagePath: "go.uber.org/zap"}}}
	root := NewRootCommand(testApp(searcher, &fakeRunner{inside: false}))

	_, _, err := executeCommand(root, "add", "zap", "--dry-run", "--yes")
	if err == nil {
		t.Fatal("Execute returned nil error, want module error")
	}
	if !strings.Contains(err.Error(), "current directory is not inside a Go module") {
		t.Fatalf("error = %v, want Go module error", err)
	}
	if searcher.called {
		t.Fatal("searcher was called before module check")
	}
}

func TestVersionsResolvesPackagePathBeforeListing(t *testing.T) {
	t.Parallel()

	resolver := &fakeResolver{modulePath: "google.golang.org/grpc"}
	runner := &fakeRunner{listVersions: []string{"v1.80.0", "v1.81.0"}}
	root := NewRootCommand(&App{
		Searcher:  &fakeSearcher{},
		Resolver:  resolver,
		Runner:    runner,
		Favorites: fakeFavorites{values: map[string]string{}},
		Selector:  &fakeSelector{},
		Latest:    &fakeLatest{},
		Opener:    &fakeOpener{},
	})

	stdout, _, err := executeCommand(root, "versions", "google.golang.org/grpc/status")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !resolver.called {
		t.Fatal("resolver was not called")
	}
	if resolver.lastCandidate.PackagePath != "google.golang.org/grpc/status" {
		t.Fatalf("resolver candidate = %#v, want package path input", resolver.lastCandidate)
	}
	if runner.listModule != "google.golang.org/grpc" {
		t.Fatalf("ListVersions module = %q, want google.golang.org/grpc", runner.listModule)
	}
	if stdout != "v1.80.0\nv1.81.0\n" {
		t.Fatalf("stdout = %q, want listed versions", stdout)
	}
}

func TestVersionsReturnsErrorWhenNoVersionsFound(t *testing.T) {
	t.Parallel()

	root := NewRootCommand(&App{
		Searcher:  &fakeSearcher{},
		Resolver:  &fakeResolver{modulePath: "example.com/module"},
		Runner:    &fakeRunner{},
		Favorites: fakeFavorites{values: map[string]string{}},
		Selector:  &fakeSelector{},
		Latest:    &fakeLatest{},
		Opener:    &fakeOpener{},
	})

	_, _, err := executeCommand(root, "versions", "example.com/module")
	if err == nil {
		t.Fatal("Execute returned nil error, want no versions error")
	}
	if !strings.Contains(err.Error(), "no versions found") {
		t.Fatalf("error = %v, want no versions message", err)
	}
}

func TestVersionsRequiresModuleOrPackageArgument(t *testing.T) {
	t.Parallel()

	root := NewRootCommand(testApp(&fakeSearcher{}, &fakeRunner{inside: true}))

	_, _, err := executeCommand(root, "versions")
	if err == nil {
		t.Fatal("Execute returned nil error, want helpful missing argument error")
	}
	for _, want := range []string{
		"missing module or package path",
		"gogetx versions <module-or-package>",
		"gogetx versions go.uber.org/zap",
		"gogetx versions google.golang.org/grpc/status",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, missing %q", err.Error(), want)
		}
	}
	if strings.Contains(err.Error(), "accepts 1 arg") {
		t.Fatalf("error = %q, should not expose generic Cobra arg message", err.Error())
	}
}

func TestLatestResolvesPackagePathBeforeLookup(t *testing.T) {
	t.Parallel()

	resolver := &fakeResolver{modulePath: "google.golang.org/grpc"}
	latest := &fakeLatest{}
	root := NewRootCommand(&App{
		Searcher:  &fakeSearcher{},
		Resolver:  resolver,
		Runner:    &fakeRunner{},
		Favorites: fakeFavorites{values: map[string]string{}},
		Selector:  &fakeSelector{},
		Latest:    latest,
		Opener:    &fakeOpener{},
	})

	stdout, _, err := executeCommand(root, "latest", "google.golang.org/grpc/status")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !resolver.called {
		t.Fatal("resolver was not called")
	}
	if latest.module != "google.golang.org/grpc" {
		t.Fatalf("Latest module = %q, want google.golang.org/grpc", latest.module)
	}
	if !strings.Contains(stdout, "v1.28.0") {
		t.Fatalf("stdout = %q, want latest version", stdout)
	}
}

func TestLatestRequiresModuleOrPackageArgument(t *testing.T) {
	t.Parallel()

	root := NewRootCommand(testApp(&fakeSearcher{}, &fakeRunner{inside: true}))

	_, _, err := executeCommand(root, "latest")
	if err == nil {
		t.Fatal("Execute returned nil error, want helpful missing argument error")
	}
	for _, want := range []string{
		"missing module or package path",
		"gogetx latest <module-or-package>",
		"gogetx latest go.uber.org/zap",
		"gogetx latest google.golang.org/grpc/status",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, missing %q", err.Error(), want)
		}
	}
}

func TestDocCommandOpensResolvedURL(t *testing.T) {
	t.Parallel()

	opener := &fakeOpener{}
	root := NewRootCommand(&App{
		Searcher:  &fakeSearcher{},
		Resolver:  &fakeResolver{modulePath: "go.uber.org/zap"},
		Runner:    &fakeRunner{},
		Favorites: fakeFavorites{values: map[string]string{}},
		Selector:  &fakeSelector{},
		Latest:    &fakeLatest{},
		Opener:    opener,
	})

	stdout, _, err := executeCommand(root, "doc", "go.uber.org/zap")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if opener.url != "https://pkg.go.dev/go.uber.org/zap" {
		t.Fatalf("opened url = %q, want pkg.go.dev URL", opener.url)
	}
	if !strings.Contains(stdout, opener.url) {
		t.Fatalf("stdout = %q, want opened URL", stdout)
	}
}

func TestFavoriteCommands(t *testing.T) {
	t.Parallel()

	root := NewRootCommand(&App{
		Searcher:  &fakeSearcher{},
		Resolver:  &fakeResolver{modulePath: "go.uber.org/zap"},
		Runner:    &fakeRunner{},
		Favorites: fakeFavorites{values: map[string]string{"logger": "go.uber.org/zap"}},
		Selector:  &fakeSelector{},
		Latest:    &fakeLatest{},
		Opener:    &fakeOpener{},
	})

	stdout, _, err := executeCommand(root, "fav")
	if err != nil {
		t.Fatalf("fav returned error: %v", err)
	}
	if !strings.Contains(stdout, "logger\tgo.uber.org/zap") {
		t.Fatalf("stdout = %q, want favorite listing", stdout)
	}

	stdout, _, err = executeCommand(root, "addfav", "zap", "go.uber.org/zap")
	if err != nil {
		t.Fatalf("addfav returned error: %v", err)
	}
	if !strings.Contains(stdout, "Added favorite zap -> go.uber.org/zap") {
		t.Fatalf("stdout = %q, want add message", stdout)
	}

	stdout, _, err = executeCommand(root, "rmfav", "zap")
	if err != nil {
		t.Fatalf("rmfav returned error: %v", err)
	}
	if !strings.Contains(stdout, "Removed favorite zap") {
		t.Fatalf("stdout = %q, want remove message", stdout)
	}
}
