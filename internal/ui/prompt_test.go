package ui

import (
	"strings"
	"testing"

	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

func TestConfirmPromptDefaultsToYes(t *testing.T) {
	t.Parallel()

	prompt := confirmPrompt("Run go get example.com/pkg@latest?")
	if prompt.Default != "y" {
		t.Fatalf("Default = %q, want y", prompt.Default)
	}
	if !prompt.IsConfirm {
		t.Fatal("IsConfirm = false, want true")
	}
}

func TestCandidateDisplayLineTruncatesLongSynopsis(t *testing.T) {
	t.Parallel()

	candidate := packageinfo.PackageCandidate{
		PackagePath: "github.com/joho/godotenv",
		Synopsis:    "Package godotenv is a go port of the ruby dotenv library (https://github.com/bkeepers/dotenv) with a very long explanation that would wrap in terminals",
	}

	line := candidateDisplayLine(candidate, 72)
	if strings.ContainsAny(line, "\r\n") {
		t.Fatalf("line contains newline: %q", line)
	}
	if got := len([]rune(line)); got > 72 {
		t.Fatalf("line length = %d, want <= 72: %q", got, line)
	}
	if !strings.Contains(line, "github.com/joho/godotenv") {
		t.Fatalf("line = %q, want package path", line)
	}
	if !strings.HasSuffix(line, "...") {
		t.Fatalf("line = %q, want ellipsis suffix", line)
	}
}

func TestCandidateDisplayLineNormalizesWhitespace(t *testing.T) {
	t.Parallel()

	candidate := packageinfo.PackageCandidate{
		PackagePath: "example.com/pkg",
		Synopsis:    "first line\nsecond\tline",
	}

	line := candidateDisplayLine(candidate, 80)
	if line != "example.com/pkg  first line second line" {
		t.Fatalf("line = %q, want normalized single line", line)
	}
}

func TestCandidateMatchesSearchUsesPathModuleAndSynopsis(t *testing.T) {
	t.Parallel()

	candidate := packageinfo.PackageCandidate{
		PackagePath: "github.com/air-verse/air",
		ModulePath:  "github.com/air-verse/air",
		Synopsis:    "Live reload for Go apps",
	}

	for _, input := range []string{"airverse", "ghair", "live reload"} {
		if !candidateMatchesSearch(input, candidate) {
			t.Fatalf("candidateMatchesSearch(%q) = false, want true", input)
		}
	}
	if candidateMatchesSearch("zaplogger", candidate) {
		t.Fatal("candidateMatchesSearch(zaplogger) = true, want false")
	}
}
