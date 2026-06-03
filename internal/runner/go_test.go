package runner

import (
	"context"
	"strings"
	"testing"
)

type fakeExecutor struct {
	outputs map[string]string
	calls   []string
}

func (f *fakeExecutor) Execute(_ context.Context, name string, args []string, _ ExecOptions) (string, error) {
	call := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, call)
	return f.outputs[call], nil
}

func TestIsInsideModuleUsesGoEnvGOMOD(t *testing.T) {
	t.Parallel()

	r := NewWithExecutor(&fakeExecutor{
		outputs: map[string]string{
			"go env GOMOD": "/tmp/app/go.mod\n",
		},
	})

	inside, err := r.IsInsideModule(context.Background())
	if err != nil {
		t.Fatalf("IsInsideModule returned error: %v", err)
	}
	if !inside {
		t.Fatal("inside = false, want true")
	}
}

func TestGetTidyAndListVersionsUseGoCommands(t *testing.T) {
	t.Parallel()

	exec := &fakeExecutor{
		outputs: map[string]string{
			"go list -m -versions go.uber.org/zap": "go.uber.org/zap v1.27.0 v1.28.0\n",
		},
	}
	r := NewWithExecutor(exec)

	if err := r.Get(context.Background(), "go.uber.org/zap", "latest"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if err := r.ModTidy(context.Background()); err != nil {
		t.Fatalf("ModTidy returned error: %v", err)
	}
	versions, err := r.ListVersions(context.Background(), "go.uber.org/zap")
	if err != nil {
		t.Fatalf("ListVersions returned error: %v", err)
	}
	if strings.Join(versions, ",") != "v1.27.0,v1.28.0" {
		t.Fatalf("versions = %v, want [v1.27.0 v1.28.0]", versions)
	}

	gotCalls := strings.Join(exec.calls, "\n")
	for _, want := range []string{
		"go get go.uber.org/zap@latest",
		"go mod tidy",
		"go list -m -versions go.uber.org/zap",
	} {
		if !strings.Contains(gotCalls, want) {
			t.Fatalf("calls = %q, missing %q", gotCalls, want)
		}
	}
}

func TestListVersionsReturnsErrorWhenOutputHasNoVersions(t *testing.T) {
	t.Parallel()

	r := NewWithExecutor(&fakeExecutor{
		outputs: map[string]string{
			"go list -m -versions google.golang.org/grpc/status": "google.golang.org/grpc/status\n",
		},
	})

	_, err := r.ListVersions(context.Background(), "google.golang.org/grpc/status")
	if err == nil {
		t.Fatal("ListVersions returned nil error, want no versions error")
	}
	if !strings.Contains(err.Error(), "no versions found") {
		t.Fatalf("error = %v, want no versions message", err)
	}
}
