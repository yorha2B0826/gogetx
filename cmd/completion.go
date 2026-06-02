package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const completionRCMarker = "# gogetx shell completion"

type completionTarget struct {
	Shell      string
	ScriptPath string
	RCPath     string
	RCBlock    string
}

func newCompletionCommand(root *cobra.Command) *cobra.Command {
	command := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate or install shell completion scripts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeCompletionScript(root, normalizeShell(args[0]), cmd.OutOrStdout())
		},
	}

	var dryRun bool
	var home string
	install := &cobra.Command{
		Use:   "install [bash|zsh|fish|powershell]",
		Short: "Install shell completion for the current user",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := ""
			if len(args) > 0 {
				shell = args[0]
			}
			if home == "" {
				var err error
				home, err = os.UserHomeDir()
				if err != nil {
					return err
				}
			}
			return installCompletion(root, shell, home, dryRun, cmd.OutOrStdout())
		},
	}
	install.Flags().BoolVar(&dryRun, "dry-run", false, "print planned changes without writing files")
	install.Flags().StringVar(&home, "home", "", "home directory override")
	_ = install.Flags().MarkHidden("home")

	command.AddCommand(install)
	return command
}

func installCompletion(root *cobra.Command, shell string, home string, dryRun bool, out io.Writer) error {
	if shell == "" || shell == "auto" {
		shell = detectShell()
	}
	target, err := completionInstallTarget(shell, home)
	if err != nil {
		return err
	}

	var script bytes.Buffer
	if err := writeCompletionScript(root, target.Shell, &script); err != nil {
		return err
	}

	if dryRun {
		fmt.Fprintf(out, "Would install %s completion to %s\n", target.Shell, target.ScriptPath)
		if target.RCPath != "" {
			fmt.Fprintf(out, "Would update %s\n", target.RCPath)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(target.ScriptPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target.ScriptPath, script.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(out, "Installed %s completion to %s\n", target.Shell, target.ScriptPath)

	if target.RCPath != "" {
		if err := appendBlockIfMissing(target.RCPath, completionRCMarker, target.RCBlock); err != nil {
			return err
		}
		fmt.Fprintf(out, "Updated %s\n", target.RCPath)
	}
	fmt.Fprintln(out, "Restart your shell to use completions.")
	return nil
}

func writeCompletionScript(root *cobra.Command, shell string, out io.Writer) error {
	switch normalizeShell(shell) {
	case "bash":
		return root.GenBashCompletionV2(out, true)
	case "zsh":
		return root.GenZshCompletion(out)
	case "fish":
		return root.GenFishCompletion(out, true)
	case "powershell":
		return root.GenPowerShellCompletionWithDesc(out)
	default:
		return fmt.Errorf("unsupported shell %q; use bash, zsh, fish, or powershell", shell)
	}
}

func completionInstallTarget(shell string, home string) (completionTarget, error) {
	home = strings.TrimSpace(home)
	if home == "" {
		return completionTarget{}, fmt.Errorf("home directory is required")
	}

	switch normalizeShell(shell) {
	case "zsh":
		return completionTarget{
			Shell:      "zsh",
			ScriptPath: filepath.Join(home, ".zsh", "completions", "_gogetx"),
			RCPath:     filepath.Join(home, ".zshrc"),
			RCBlock: strings.Join([]string{
				completionRCMarker,
				`if [[ -d "$HOME/.zsh/completions" ]]; then`,
				`  fpath=("$HOME/.zsh/completions" $fpath)`,
				`  autoload -Uz compinit`,
				`  compinit`,
				`fi`,
				``,
			}, "\n"),
		}, nil
	case "bash":
		return completionTarget{
			Shell:      "bash",
			ScriptPath: filepath.Join(home, ".local", "share", "bash-completion", "completions", "gogetx"),
			RCPath:     filepath.Join(home, ".bashrc"),
			RCBlock: strings.Join([]string{
				completionRCMarker,
				`if [ -f "$HOME/.local/share/bash-completion/completions/gogetx" ]; then`,
				`  . "$HOME/.local/share/bash-completion/completions/gogetx"`,
				`fi`,
				``,
			}, "\n"),
		}, nil
	case "fish":
		return completionTarget{
			Shell:      "fish",
			ScriptPath: filepath.Join(home, ".config", "fish", "completions", "gogetx.fish"),
		}, nil
	case "powershell":
		return completionTarget{
			Shell:      "powershell",
			ScriptPath: filepath.Join(home, ".config", "powershell", "gogetx_completion.ps1"),
			RCPath:     filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"),
			RCBlock: strings.Join([]string{
				completionRCMarker,
				`. "$HOME/.config/powershell/gogetx_completion.ps1"`,
				``,
			}, "\n"),
		}, nil
	default:
		return completionTarget{}, fmt.Errorf("unsupported shell %q; use bash, zsh, fish, or powershell", shell)
	}
}

func appendBlockIfMissing(path string, marker string, block string) error {
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if strings.Contains(string(content), marker) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var next bytes.Buffer
	next.Write(content)
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		next.WriteByte('\n')
	}
	if len(content) > 0 {
		next.WriteByte('\n')
	}
	next.WriteString(block)
	if !strings.HasSuffix(block, "\n") {
		next.WriteByte('\n')
	}
	return os.WriteFile(path, next.Bytes(), 0o644)
}

func detectShell() string {
	shell := filepath.Base(os.Getenv("SHELL"))
	if shell == "" || shell == "." || shell == string(filepath.Separator) {
		return "zsh"
	}
	return normalizeShell(shell)
}

func normalizeShell(shell string) string {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "pwsh", "powershell.exe":
		return "powershell"
	default:
		return strings.ToLower(strings.TrimSpace(shell))
	}
}
