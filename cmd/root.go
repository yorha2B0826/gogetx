package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yorha2B0826/gogetx/internal/doc"
	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

func NewRootCommand(app *App) *Command {
	if app == nil {
		app = NewDefaultApp()
	}

	root := &cobra.Command{
		Use:           "gogetx",
		Short:         "Search and install Go modules from the terminal",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newSearchCommand(app))
	root.AddCommand(newAddCommand(app))
	root.AddCommand(newVersionsCommand(app))
	root.AddCommand(newLatestCommand(app))
	root.AddCommand(newDocCommand(app))
	root.AddCommand(newFavCommand(app))
	root.AddCommand(newAddFavCommand(app))
	root.AddCommand(newRmFavCommand(app))
	root.AddCommand(newCompletionCommand(root))
	return root
}

func newSearchCommand(app *App) *cobra.Command {
	opts := packageinfo.SearchOptions{Limit: 10, Source: packageinfo.SourcePkgsite}
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "search <keyword>",
		Short: "Search Go packages and modules",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := app.Searcher.Search(cmd.Context(), args[0], opts)
			if err != nil {
				return err
			}
			if jsonOutput {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(results)
			}
			printCandidates(cmd, results)
			return nil
		},
	}
	addSearchFlags(command, &opts)
	command.Flags().BoolVar(&jsonOutput, "json", false, "output search results as JSON")
	return command
}

func newAddCommand(app *App) *cobra.Command {
	opts := packageinfo.SearchOptions{Limit: 10, Source: packageinfo.SourcePkgsite}
	var version string
	var tidy bool
	var yes bool
	var dryRun bool
	var first bool

	command := &cobra.Command{
		Use:   "add <keyword>",
		Short: "Search and add a Go module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inside, err := app.Runner.IsInsideModule(cmd.Context())
			if err != nil {
				return err
			}
			if !inside {
				return fmt.Errorf("current directory is not inside a Go module; run `go mod init <module>` first")
			}

			selected, err := selectCandidate(cmd, app, args[0], opts, first)
			if err != nil {
				return err
			}
			modulePath, err := app.Resolver.Resolve(cmd.Context(), selected)
			if err != nil {
				return err
			}
			if version == "" {
				version = "latest"
			}
			commandLine := fmt.Sprintf("go get %s@%s", modulePath, version)
			fmt.Fprintf(cmd.OutOrStdout(), "Resolved module: %s\n", modulePath)
			fmt.Fprintf(cmd.OutOrStdout(), "Command: %s\n", commandLine)
			if dryRun {
				return nil
			}
			if !yes {
				ok, err := app.Selector.Confirm("Run " + commandLine)
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}
			if err := app.Runner.Get(cmd.Context(), modulePath, version); err != nil {
				return err
			}
			if tidy {
				return app.Runner.ModTidy(cmd.Context())
			}
			return nil
		},
	}
	addSearchFlags(command, &opts)
	command.Flags().StringVar(&version, "version", "latest", "version to install")
	command.Flags().BoolVar(&tidy, "tidy", false, "run go mod tidy after go get")
	command.Flags().BoolVar(&yes, "yes", false, "skip confirmation before running go get")
	command.Flags().BoolVar(&first, "first", false, "select the first search result without prompting")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "print commands without executing them")
	return command
}

func newVersionsCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "versions <module>",
		Short: "List available module versions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			versions, err := app.Runner.ListVersions(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			for _, version := range versions {
				fmt.Fprintln(cmd.OutOrStdout(), version)
			}
			return nil
		},
	}
}

func newLatestCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "latest <module>",
		Short: "Show the latest module version from the Go proxy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := app.Latest.Latest(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if info.Time.IsZero() {
				fmt.Fprintln(cmd.OutOrStdout(), info.Version)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", info.Version, info.Time.Format("2006-01-02T15:04:05Z07:00"))
			return nil
		},
	}
}

func newDocCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "doc <target>",
		Short: "Open pkg.go.dev documentation or search",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := doc.URLFor(args[0])
			fmt.Fprintf(cmd.OutOrStdout(), "Opening %s\n", url)
			return app.Opener.Open(cmd.Context(), url)
		},
	}
}

func newFavCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "fav",
		Short: "List favorite module aliases",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			favorites, err := app.Favorites.Favorites()
			if err != nil {
				return err
			}
			if len(favorites) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No favorites configured.")
				return nil
			}
			aliases := make([]string, 0, len(favorites))
			for alias := range favorites {
				aliases = append(aliases, alias)
			}
			sort.Strings(aliases)
			for _, alias := range aliases {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", alias, favorites[alias])
			}
			return nil
		},
	}
}

func newAddFavCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "addfav <alias> <module>",
		Short: "Add a favorite module alias",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Favorites.AddFavorite(args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added favorite %s -> %s\n", args[0], args[1])
			return nil
		},
	}
}

func newRmFavCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "rmfav <alias>",
		Short: "Remove a favorite module alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Favorites.RemoveFavorite(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed favorite %s\n", args[0])
			return nil
		},
	}
}

func addSearchFlags(command *cobra.Command, opts *packageinfo.SearchOptions) {
	command.Flags().IntVar(&opts.Limit, "limit", 10, "maximum number of results")
	command.Flags().StringVar(&opts.Source, "source", packageinfo.SourcePkgsite, "search source: pkgsite, github, or all")
	command.Flags().BoolVar(&opts.NoCache, "no-cache", false, "disable search cache")
	command.Flags().BoolVar(&opts.Refresh, "refresh", false, "refresh cached search results")
}

func selectCandidate(cmd *cobra.Command, app *App, keyword string, opts packageinfo.SearchOptions, first bool) (packageinfo.PackageCandidate, error) {
	if modulePath, ok, err := app.Favorites.Favorite(keyword); err != nil {
		return packageinfo.PackageCandidate{}, err
	} else if ok {
		fmt.Fprintf(cmd.OutOrStdout(), "Favorite: %s -> %s\n", keyword, modulePath)
		return packageinfo.PackageCandidate{
			PackagePath: modulePath,
			ModulePath:  modulePath,
			Source:      "favorite",
		}, nil
	}

	results, err := app.Searcher.Search(cmd.Context(), keyword, opts)
	if err != nil {
		return packageinfo.PackageCandidate{}, err
	}
	if len(results) == 0 {
		return packageinfo.PackageCandidate{}, fmt.Errorf("no results found for %q", keyword)
	}
	if first || len(results) == 1 {
		fmt.Fprintf(cmd.OutOrStdout(), "Selected: %s\n", results[0].PackagePath)
		return results[0], nil
	}
	return app.Selector.Select(results)
}

func printCandidates(cmd *cobra.Command, results []packageinfo.PackageCandidate) {
	if len(results) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No results found.")
		return
	}
	for i, result := range results {
		path := result.PackagePath
		if result.ModulePath != "" && result.ModulePath != result.PackagePath {
			path = path + " (" + result.ModulePath + ")"
		}
		metadata := compactJoin([]string{result.Version, result.Source}, " ")
		if metadata != "" {
			metadata = " [" + metadata + "]"
		}
		if result.Synopsis == "" {
			fmt.Fprintf(cmd.OutOrStdout(), "%2d. %s%s\n", i+1, path, metadata)
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%2d. %s%s\n    %s\n", i+1, path, metadata, result.Synopsis)
	}
}

func compactJoin(parts []string, sep string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, sep)
}
