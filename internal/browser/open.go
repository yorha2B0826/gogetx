package browser

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

type Opener struct{}

func New() Opener {
	return Opener{}
}

func (Opener) Open(ctx context.Context, url string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name = "open"
		args = []string{url}
	case "windows":
		name = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		name = "xdg-open"
		args = []string{url}
	}
	if err := exec.CommandContext(ctx, name, args...).Run(); err != nil {
		return fmt.Errorf("open documentation URL: %w", err)
	}
	return nil
}
