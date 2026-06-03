package pkgsite

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

func TestClientSearchUsesV1BetaSearch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/search" {
			t.Fatalf("path = %q, want /v1beta/search", r.URL.Path)
		}
		if got := r.URL.Query().Get("q"); got != "zap" {
			t.Fatalf("q = %q, want zap", got)
		}
		if got := r.URL.Query().Get("limit"); got != "2" {
			t.Fatalf("limit = %q, want 2", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [
				{
					"packagePath": "go.uber.org/zap",
					"modulePath": "go.uber.org/zap",
					"version": "v1.28.0",
					"synopsis": "Package zap provides fast, structured, leveled logging."
				}
			],
			"total": 1
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	got, err := client.Search(context.Background(), "zap", packageinfo.SearchOptions{Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(got))
	}
	if got[0].PackagePath != "go.uber.org/zap" {
		t.Fatalf("PackagePath = %q, want go.uber.org/zap", got[0].PackagePath)
	}
	if got[0].ModulePath != "go.uber.org/zap" {
		t.Fatalf("ModulePath = %q, want go.uber.org/zap", got[0].ModulePath)
	}
	if got[0].Source != "pkgsite" {
		t.Fatalf("Source = %q, want pkgsite", got[0].Source)
	}
}

func TestClientSearchPageUsesPaginationToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/search" {
			t.Fatalf("path = %q, want /v1beta/search", r.URL.Path)
		}
		if got := r.URL.Query().Get("q"); got != "air" {
			t.Fatalf("q = %q, want air", got)
		}
		if got := r.URL.Query().Get("limit"); got != "2" {
			t.Fatalf("limit = %q, want 2", got)
		}
		if got := r.URL.Query().Get("token"); got != "page-2" {
			t.Fatalf("token = %q, want page-2", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [
				{
					"packagePath": "github.com/air-verse/air",
					"modulePath": "github.com/air-verse/air",
					"version": "v1.62.0",
					"synopsis": "Live reload for Go apps."
				}
			],
			"nextPageToken": "page-3",
			"total": 42
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	got, err := client.SearchPage(context.Background(), "air", packageinfo.SearchOptions{
		Limit:     2,
		PageToken: "page-2",
	})
	if err != nil {
		t.Fatalf("SearchPage returned error: %v", err)
	}
	if got.NextPageToken != "page-3" {
		t.Fatalf("NextPageToken = %q, want page-3", got.NextPageToken)
	}
	if got.Total != 42 {
		t.Fatalf("Total = %d, want 42", got.Total)
	}
	if len(got.Results) != 1 || got.Results[0].ModulePath != "github.com/air-verse/air" {
		t.Fatalf("results = %#v, want air result", got.Results)
	}
}
