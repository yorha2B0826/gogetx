package goproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLatestReturnsVersionInfo(t *testing.T) {
	t.Parallel()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Version":"v1.28.0","Time":"2024-01-01T00:00:00Z"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	info, err := client.Latest(context.Background(), "go.uber.org/zap")
	if err != nil {
		t.Fatalf("Latest returned error: %v", err)
	}
	if gotPath != "/go.uber.org/zap/@latest" {
		t.Fatalf("path = %q, want /go.uber.org/zap/@latest", gotPath)
	}
	if info.Version != "v1.28.0" {
		t.Fatalf("Version = %q, want v1.28.0", info.Version)
	}
	if !info.Time.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("Time = %v, want 2024-01-01T00:00:00Z", info.Time)
	}
}

func TestLatestReturnsErrorOnMissingModule(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	_, err := client.Latest(context.Background(), "example.com/missing")
	if err == nil {
		t.Fatal("Latest returned nil error, want 404 error")
	}
	if !strings.Contains(err.Error(), "latest lookup failed") {
		t.Fatalf("error = %v, want proxy latest lookup failed", err)
	}
}

func TestListVersionsParsesLines(t *testing.T) {
	t.Parallel()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte("v1.27.0\nv1.28.0\n"))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	versions, err := client.ListVersions(context.Background(), "go.uber.org/zap")
	if err != nil {
		t.Fatalf("ListVersions returned error: %v", err)
	}
	if gotPath != "/go.uber.org/zap/@v/list" {
		t.Fatalf("path = %q, want /go.uber.org/zap/@v/list", gotPath)
	}
	if strings.Join(versions, ",") != "v1.27.0,v1.28.0" {
		t.Fatalf("versions = %v, want [v1.27.0 v1.28.0]", versions)
	}
}

func TestListVersionsReturnsErrorWhenEmpty(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(nil)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	_, err := client.ListVersions(context.Background(), "example.com/empty")
	if err == nil {
		t.Fatal("ListVersions returned nil error, want no versions error")
	}
	if !strings.Contains(err.Error(), "no versions found") {
		t.Fatalf("error = %v, want no versions message", err)
	}
}

func TestListVersionsReturnsErrorOn404(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if _, err := client.ListVersions(context.Background(), "example.com/missing"); err == nil {
		t.Fatal("ListVersions returned nil error, want 404 error")
	}
}

func TestNewClientUsesGOPROXYEnvironment(t *testing.T) {
	t.Setenv("GOPROXY", "https://goproxy.cn,direct")
	client := NewClient()
	if client.baseURL != "https://goproxy.cn" {
		t.Fatalf("baseURL = %q, want https://goproxy.cn", client.baseURL)
	}
}

func TestNewClientFallsBackWhenGOPROXYIsOff(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	client := NewClient()
	if client.baseURL != defaultProxyBaseURL {
		t.Fatalf("baseURL = %q, want %q", client.baseURL, defaultProxyBaseURL)
	}
}

func TestNewClientFallsBackWhenGOPROXYIsUnset(t *testing.T) {
	t.Setenv("GOPROXY", "")
	client := NewClient()
	if client.baseURL != defaultProxyBaseURL {
		t.Fatalf("baseURL = %q, want %q", client.baseURL, defaultProxyBaseURL)
	}
}
