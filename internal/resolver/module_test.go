package resolver

import (
	"context"
	"errors"
	"testing"

	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

type fakeVersionLister map[string]bool

func (f fakeVersionLister) ListVersions(_ context.Context, modulePath string) ([]string, error) {
	if f[modulePath] {
		return []string{"v1.0.0"}, nil
	}
	return nil, errors.New("module not found")
}

func TestResolveUsesCandidateModulePath(t *testing.T) {
	t.Parallel()

	resolved, err := New(nil).Resolve(context.Background(), packageinfo.PackageCandidate{
		PackagePath: "go.uber.org/zap/zapcore",
		ModulePath:  "go.uber.org/zap",
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if resolved != "go.uber.org/zap" {
		t.Fatalf("resolved = %q, want go.uber.org/zap", resolved)
	}
}

func TestResolveFindsContainingModule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		packagePath string
		found       fakeVersionLister
		want        string
	}{
		{
			name:        "google grpc subpackage",
			packagePath: "google.golang.org/grpc/status",
			found:       fakeVersionLister{"google.golang.org/grpc": true},
			want:        "google.golang.org/grpc",
		},
		{
			name:        "github major version module",
			packagePath: "github.com/labstack/echo/v4",
			found:       fakeVersionLister{"github.com/labstack/echo/v4": true},
			want:        "github.com/labstack/echo/v4",
		},
		{
			name:        "github subpackage",
			packagePath: "github.com/gin-gonic/gin/binding",
			found:       fakeVersionLister{"github.com/gin-gonic/gin": true},
			want:        "github.com/gin-gonic/gin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := New(tt.found).Resolve(context.Background(), packageinfo.PackageCandidate{
				PackagePath: tt.packagePath,
			})
			if err != nil {
				t.Fatalf("Resolve returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolved = %q, want %q", got, tt.want)
			}
		})
	}
}
