package main

import (
	"fmt"
	"os"

	"github.com/yorha2B0826/gogetx/cmd"
)

func main() {
	root := cmd.NewRootCommand(nil)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
