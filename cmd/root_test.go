package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/yorha2B0826/gogetx/internal/goproxy"
	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

type fakeSearcher struct {
	results []packageinfo.PackageCandidate
	called  bool
}

func (f *fakeSearcher) Search(_ context.Context, _ string, _ packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error) {
	f.called = true
	return f.results, nil
}

type fakeResolver struct {
	modulePath string
}

func (f fakeResolver) Resolve(_ context.Context, _ packageinfo.PackageCandidate) (string, error) {
	return f.modulePath, nil
}

type fakeRunner struct {
	inside       bool
	getCalled    bool
	tidyCalled   bool
	listVersions []string
}

func (f *fakeRunner) Get(_ context.Context, _, _ string) error {
	f.getCalled = true
	return nil
}

func (f *fakeRunner) ModTidy(_ context.Context) error {
	f.tidyCalled = true
	return nil
}

func (f *fakeRunner) ListVersions(_ context.Context, _ string) ([]string, error) {
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

type fakeSelector struct{}

func (fakeSelector) Select(results []packageinfo.PackageCandidate) (packageinfo.PackageCandidate, error) {
	if len(results) == 0 {
		return packageinfo.PackageCandidate{}, errors.New("no results")
	}
	return results[0], nil
}

func (fakeSelector) Confirm(_ string) (bool, error) { return true, nil }

type fakeLatest struct{}

func (fakeLatest) Latest(_ context.Context, _ string) (goproxy.VersionInfo, error) {
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
	return &App{
		Searcher:  searcher,
		Resolver:  fakeResolver{modulePath: "go.uber.org/zap"},
		Runner:    runner,
		Favorites: fakeFavorites{values: map[string]string{}},
		Selector:  fakeSelector{},
		Latest:    fakeLatest{},
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
