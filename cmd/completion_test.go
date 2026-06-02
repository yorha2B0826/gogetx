package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionInstallTargetZsh(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	target, err := completionInstallTarget("zsh", home)
	if err != nil {
		t.Fatalf("completionInstallTarget returned error: %v", err)
	}

	if target.Shell != "zsh" {
		t.Fatalf("Shell = %q, want zsh", target.Shell)
	}
	wantScript := filepath.Join(home, ".zsh", "completions", "_gogetx")
	if target.ScriptPath != wantScript {
		t.Fatalf("ScriptPath = %q, want %q", target.ScriptPath, wantScript)
	}
	wantRC := filepath.Join(home, ".zshrc")
	if target.RCPath != wantRC {
		t.Fatalf("RCPath = %q, want %q", target.RCPath, wantRC)
	}
	if !strings.Contains(target.RCBlock, "fpath=(\"$HOME/.zsh/completions\" $fpath)") {
		t.Fatalf("RCBlock = %q, want zsh fpath setup", target.RCBlock)
	}
}

func TestCompletionInstallTargetFish(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	target, err := completionInstallTarget("fish", home)
	if err != nil {
		t.Fatalf("completionInstallTarget returned error: %v", err)
	}

	wantScript := filepath.Join(home, ".config", "fish", "completions", "gogetx.fish")
	if target.ScriptPath != wantScript {
		t.Fatalf("ScriptPath = %q, want %q", target.ScriptPath, wantScript)
	}
	if target.RCPath != "" || target.RCBlock != "" {
		t.Fatalf("fish should not need rc changes, got path=%q block=%q", target.RCPath, target.RCBlock)
	}
}

func TestAppendBlockIfMissingIsIdempotent(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".zshrc")
	block := "# gogetx shell completion\nsource example\n"
	if err := appendBlockIfMissing(path, "# gogetx shell completion", block); err != nil {
		t.Fatalf("appendBlockIfMissing returned error: %v", err)
	}
	if err := appendBlockIfMissing(path, "# gogetx shell completion", block); err != nil {
		t.Fatalf("appendBlockIfMissing second call returned error: %v", err)
	}

	content := readTestFile(t, path)
	if got := strings.Count(content, "# gogetx shell completion"); got != 1 {
		t.Fatalf("marker count = %d, want 1 in %q", got, content)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	return string(content)
}
