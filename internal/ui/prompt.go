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
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "> {{ .Display | cyan }}",
		Inactive: "  {{ .Display }}",
		Selected: "{{ .Candidate.PackagePath | green }}",
	}
	prompt := promptui.Select{
		Label:     "Select package",
		Items:     items,
		Templates: templates,
		Size:      10,
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
