package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

const developmentVersion = "(devel)"

func currentVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return developmentVersion
	}
	return info.Main.Version
}

func newVersionCommand(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show gogetx version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "gogetx %s\n", version)
			return nil
		},
	}
}
