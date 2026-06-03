package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type ExecOptions struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type Executor interface {
	Execute(ctx context.Context, name string, args []string, opts ExecOptions) (string, error)
}

type OSExecutor struct{}

func (OSExecutor) Execute(ctx context.Context, name string, args []string, opts ExecOptions) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = opts.Stdin

	var stdout bytes.Buffer
	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	} else {
		cmd.Stdout = &stdout
	}

	var stderr bytes.Buffer
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	} else {
		cmd.Stderr = &stderr
	}

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return stdout.String(), fmt.Errorf("%w: %s", err, msg)
		}
		return stdout.String(), err
	}
	return stdout.String(), nil
}

type GoRunner struct {
	executor Executor
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
}

func New() *GoRunner {
	return &GoRunner{
		executor: OSExecutor{},
		stdin:    os.Stdin,
		stdout:   os.Stdout,
		stderr:   os.Stderr,
	}
}

func NewWithExecutor(executor Executor) *GoRunner {
	return &GoRunner{
		executor: executor,
		stdin:    os.Stdin,
		stdout:   os.Stdout,
		stderr:   os.Stderr,
	}
}

func (r *GoRunner) Get(ctx context.Context, modulePath string, version string) error {
	if version == "" {
		version = "latest"
	}
	_, err := r.executor.Execute(ctx, "go", []string{"get", modulePath + "@" + version}, ExecOptions{
		Stdin:  r.stdin,
		Stdout: r.stdout,
		Stderr: r.stderr,
	})
	return err
}

func (r *GoRunner) ModTidy(ctx context.Context) error {
	_, err := r.executor.Execute(ctx, "go", []string{"mod", "tidy"}, ExecOptions{
		Stdin:  r.stdin,
		Stdout: r.stdout,
		Stderr: r.stderr,
	})
	return err
}

func (r *GoRunner) ListVersions(ctx context.Context, modulePath string) ([]string, error) {
	output, err := r.executor.Execute(ctx, "go", []string{"list", "-m", "-versions", modulePath}, ExecOptions{})
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(output)
	if len(fields) <= 1 {
		return nil, fmt.Errorf("no versions found for module %q", modulePath)
	}
	return fields[1:], nil
}

func (r *GoRunner) IsInsideModule(ctx context.Context) (bool, error) {
	output, err := r.executor.Execute(ctx, "go", []string{"env", "GOMOD"}, ExecOptions{})
	if err != nil {
		return false, err
	}
	gomod := strings.TrimSpace(output)
	return gomod != "" && gomod != "/dev/null", nil
}
