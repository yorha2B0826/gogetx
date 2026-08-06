package github

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

func TestSearchReadsModulePathFromGoMod(t *testing.T) {
	t.Parallel()

	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/repositories":
			gotQuery = r.URL.Query().Get("q")
			_, _ = w.Write([]byte(`{"items":[
				{"full_name":"spf13/cobra","description":"A Commander for modern Go","default_branch":"main"}
			]}`))
		case "/repos/spf13/cobra/contents/go.mod":
			_, _ = w.Write([]byte(`{"content":"` +
				base64.StdEncoding.EncodeToString([]byte("module github.com/spf13/cobra\n")) +
				`","encoding":"base64"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	results, err := client.Search(context.Background(), "cobra", packageinfo.SearchOptions{Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if gotQuery != "cobra language:go" {
		t.Fatalf("query = %q, want cobra language:go", gotQuery)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	got := results[0]
	if got.PackagePath != "github.com/spf13/cobra" || got.ModulePath != "github.com/spf13/cobra" {
		t.Fatalf("candidate = %#v, want github.com/spf13/cobra", got)
	}
	if got.Source != packageinfo.SourceGitHub {
		t.Fatalf("Source = %q, want github", got.Source)
	}
	if !strings.Contains(got.Synopsis, "Commander") {
		t.Fatalf("Synopsis = %q, want repo description", got.Synopsis)
	}
}

func TestSearchSkipsReposWithoutGoMod(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/repositories":
			_, _ = w.Write([]byte(`{"items":[
				{"full_name":"org/hasgomod","default_branch":"main"},
				{"full_name":"org/nogomod","default_branch":"main"}
			]}`))
		case "/repos/org/hasgomod/contents/go.mod":
			_, _ = w.Write([]byte(`{"content":"` +
				base64.StdEncoding.EncodeToString([]byte("module github.com/org/hasgomod\n")) +
				`","encoding":"base64"}`))
		case "/repos/org/nogomod/contents/go.mod":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	results, err := client.Search(context.Background(), "golang", packageinfo.SearchOptions{Limit: 5})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1 (repo without go.mod skipped)", len(results))
	}
	if results[0].ModulePath != "github.com/org/hasgomod" {
		t.Fatalf("candidate = %#v, want repo with go.mod only", results[0])
	}
}

func TestSearchSetsAuthorizationHeaderWithToken(t *testing.T) {
	t.Parallel()

	var auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()), WithToken("secret"))
	if _, err := client.Search(context.Background(), "cobra", packageinfo.SearchOptions{}); err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if auth != "Bearer secret" {
		t.Fatalf("Authorization = %q, want Bearer secret", auth)
	}
}

func TestSearchReturnsErrorOnNon2xx(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	_, err := client.Search(context.Background(), "cobra", packageinfo.SearchOptions{})
	if err == nil {
		t.Fatal("Search returned nil error, want non-2xx error")
	}
	if !strings.Contains(err.Error(), "github search failed") {
		t.Fatalf("error = %v, want github search failed", err)
	}
}
