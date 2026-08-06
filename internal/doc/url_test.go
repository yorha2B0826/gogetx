package doc

import "testing"

func TestURLForPackagePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "module root", target: "go.uber.org/zap", want: "https://pkg.go.dev/go.uber.org/zap"},
		{name: "subpackage", target: "google.golang.org/grpc/status", want: "https://pkg.go.dev/google.golang.org/grpc/status"},
		{name: "trims whitespace", target: "  go.uber.org/zap  ", want: "https://pkg.go.dev/go.uber.org/zap"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := URLFor(tt.target); got != tt.want {
				t.Fatalf("URLFor(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestURLForKeywordSearch(t *testing.T) {
	t.Parallel()

	if got := URLFor("zap"); got != "https://pkg.go.dev/search?q=zap" {
		t.Fatalf("URLFor(zap) = %q, want search URL", got)
	}
}
