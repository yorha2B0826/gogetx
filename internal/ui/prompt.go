package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/manifoldco/promptui"

	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

type PromptUI struct{}

func New() PromptUI {
	return PromptUI{}
}

func (PromptUI) Select(results []packageinfo.PackageCandidate) (packageinfo.PackageCandidate, error) {
	if len(results) == 0 {
		return packageinfo.PackageCandidate{}, fmt.Errorf("no results to select")
	}
	items := make([]selectItem, 0, len(results))
	width := selectDisplayWidth()
	for _, result := range results {
		items = append(items, selectItem{
			Candidate: result,
			Display:   candidateDisplayLine(result, width),
		})
	}
	searcher := func(input string, index int) bool {
		if index < 0 || index >= len(items) {
			return false
		}
		return candidateMatchesSearch(input, items[index].Candidate)
	}
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "> {{ .Display | cyan }}",
		Inactive: "  {{ .Display }}",
		Selected: "{{ .Candidate.PackagePath | green }}",
	}
	prompt := promptui.Select{
		Label:             "Select package",
		Items:             items,
		Templates:         templates,
		Size:              selectDisplaySize(len(items)),
		Searcher:          searcher,
		StartInSearchMode: true,
	}
	index, _, err := prompt.Run()
	if err != nil {
		return packageinfo.PackageCandidate{}, err
	}
	return items[index].Candidate, nil
}

func (PromptUI) Confirm(message string) (bool, error) {
	prompt := confirmPrompt(message)
	_, err := prompt.Run()
	if err == nil {
		return true, nil
	}
	if err == promptui.ErrAbort {
		return false, nil
	}
	return false, err
}

type selectItem struct {
	Candidate packageinfo.PackageCandidate
	Display   string
}

func confirmPrompt(message string) promptui.Prompt {
	return promptui.Prompt{
		Label:     message,
		Default:   "y",
		IsConfirm: true,
	}
}

func selectDisplayWidth() int {
	columns, err := strconv.Atoi(os.Getenv("COLUMNS"))
	if err != nil || columns <= 0 {
		return 96
	}
	width := columns - 8
	if width < 40 {
		return 40
	}
	if width > 120 {
		return 120
	}
	return width
}

func selectDisplaySize(itemCount int) int {
	if itemCount < 1 {
		return 1
	}
	size := 15
	if lines, err := strconv.Atoi(os.Getenv("LINES")); err == nil && lines > 0 {
		size = lines - 8
	}
	if size < 10 {
		size = 10
	}
	if size > 25 {
		size = 25
	}
	if itemCount < size {
		return itemCount
	}
	return size
}

func candidateDisplayLine(candidate packageinfo.PackageCandidate, maxRunes int) string {
	path := strings.TrimSpace(candidate.PackagePath)
	synopsis := strings.Join(strings.Fields(candidate.Synopsis), " ")
	line := path
	if synopsis != "" {
		line += "  " + synopsis
	}
	return truncateRunes(line, maxRunes)
}

func truncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 || utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	if maxRunes <= 3 {
		return strings.Repeat(".", maxRunes)
	}
	runes := []rune(s)
	return string(runes[:maxRunes-3]) + "..."
}

func candidateMatchesSearch(input string, candidate packageinfo.PackageCandidate) bool {
	needle := normalizeSearchText(input)
	if needle == "" {
		return true
	}
	haystack := normalizeSearchText(strings.Join([]string{
		candidate.PackagePath,
		candidate.ModulePath,
		candidate.Synopsis,
		candidate.Source,
	}, " "))
	return strings.Contains(haystack, needle) || fuzzySubsequence(needle, haystack)
}

func normalizeSearchText(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// fuzzySubsequence reports whether needle appears in haystack in order. It
// indexes runes (not bytes) so multi-byte inputs remain correct.
func fuzzySubsequence(needle string, haystack string) bool {
	if needle == "" {
		return true
	}
	needleRunes := []rune(needle)
	i := 0
	for _, r := range haystack {
		if needleRunes[i] == r {
			i++
			if i == len(needleRunes) {
				return true
			}
		}
	}
	return false
}
