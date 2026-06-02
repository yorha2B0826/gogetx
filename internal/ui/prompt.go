package ui

import (
	"fmt"

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
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "> {{ .PackagePath | cyan }} {{ .Synopsis }}",
		Inactive: "  {{ .PackagePath }} {{ .Synopsis }}",
		Selected: "{{ .PackagePath | green }}",
	}
	prompt := promptui.Select{
		Label:     "Select package",
		Items:     results,
		Templates: templates,
		Size:      10,
	}
	index, _, err := prompt.Run()
	if err != nil {
		return packageinfo.PackageCandidate{}, err
	}
	return results[index], nil
}

func (PromptUI) Confirm(message string) (bool, error) {
	prompt := promptui.Prompt{
		Label:     message,
		IsConfirm: true,
	}
	_, err := prompt.Run()
	if err == nil {
		return true, nil
	}
	if err == promptui.ErrAbort {
		return false, nil
	}
	return false, err
}
