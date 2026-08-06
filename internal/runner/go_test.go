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

func TestGetAndTidyUseGoCommands(t *testing.T) {
	t.Parallel()

	exec := &fakeExecutor{outputs: map[string]string{}}
	r := NewWithExecutor(exec)

	if err := r.Get(context.Background(), "go.uber.org/zap", "latest"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if err := r.ModTidy(context.Background()); err != nil {
		t.Fatalf("ModTidy returned error: %v", err)
	}

	gotCalls := strings.Join(exec.calls, "\n")
	for _, want := range []string{
		"go get go.uber.org/zap@latest",
		"go mod tidy",
	} {
		if !strings.Contains(gotCalls, want) {
			t.Fatalf("calls = %q, missing %q", gotCalls, want)
		}
	}
}
