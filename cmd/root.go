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

const (
	loadMoreSource = "__gogetx_load_more__"
)

func NewRootCommand(app *App) *Command {
	if app == nil {
		app = NewDefaultApp()
	}

	version := currentVersion()
	root := &cobra.Command{
		Use:           "gogetx",
		Short:         "Search and install Go modules from the terminal",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("gogetx {{.Version}}\n")
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newSearchCommand(app))
	root.AddCommand(newAddCommand(app))
	root.AddCommand(newVersionsCommand(app))
	root.AddCommand(newLatestCommand(app))
	root.AddCommand(newVersionCommand(version))
	root.AddCommand(newDocCommand(app))
	root.AddCommand(newFavCommand(app))
	root.AddCommand(newAddFavCommand(app))
	root.AddCommand(newRmFavCommand(app))
	root.AddCommand(newCompletionCommand(root))
	return root
}

func newSearchCommand(app *App) *cobra.Command {
	opts := packageinfo.SearchOptions{Limit: packageinfo.DefaultSearchLimit, Source: packageinfo.SourcePkgsite}
	var jsonOutput bool
	var pageNumber int
	var allPages bool
	command := &cobra.Command{
		Use:   "search <keyword>",
		Short: "Search Go packages and modules",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if pageNumber < 1 {
				return fmt.Errorf("page must be 1 or greater")
			}
			if allPages && pageNumber != 1 {
				return fmt.Errorf("--all cannot be combined with --page greater than 1")
			}

			page, err := searchResultsPage(cmd, app, args[0], opts, pageNumber, allPages)
			if err != nil {
				return err
			}
			if jsonOutput {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(page.Results)
			}
			printCandidates(cmd, page.Results)
			if !allPages && page.NextPageToken != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "\nMore results available. Next page: gogetx search %s --page %d --limit %d\n", shellQuote(args[0]), pageNumber+1, opts.Limit)
			}
			return nil
		},
	}
	addSearchFlags(command, &opts)
	command.Flags().BoolVar(&jsonOutput, "json", false, "output search results as JSON")
	command.Flags().IntVar(&pageNumber, "page", 1, "search result page to show")
	command.Flags().BoolVar(&allPages, "all", false, "fetch all available search result pages")
	return command
}

func newAddCommand(app *App) *cobra.Command {
	opts := packageinfo.SearchOptions{Limit: packageinfo.DefaultSearchLimit, Source: packageinfo.SourcePkgsite}
	var version string
	var tidy bool
	var yes bool
	var dryRun bool
	var first bool
	var allPages bool

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

			selected, err := selectCandidate(cmd, app, args[0], opts, first, allPages)
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
	command.Flags().BoolVar(&allPages, "all", false, "fetch all available search result pages before interactive selection")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "print commands without executing them")
	return command
}

func newVersionsCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "versions <module-or-package>",
		Short: "List available module versions",
		Args:  moduleOrPackageArg("versions"),
		RunE: func(cmd *cobra.Command, args []string) error {
			modulePath, err := resolveModulePath(cmd, app, args[0])
			if err != nil {
				return err
			}
			versions, err := app.Runner.ListVersions(cmd.Context(), modulePath)
			if err != nil {
				return err
			}
			if len(versions) == 0 {
				return fmt.Errorf("no versions found for module %q", modulePath)
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
		Use:   "latest <module-or-package>",
		Short: "Show the latest module version from the Go proxy",
		Args:  moduleOrPackageArg("latest"),
		RunE: func(cmd *cobra.Command, args []string) error {
			modulePath, err := resolveModulePath(cmd, app, args[0])
			if err != nil {
				return err
			}
			info, err := app.Latest.Latest(cmd.Context(), modulePath)
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

func moduleOrPackageArg(commandName string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 1 {
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf(`missing module or package path

Usage:
  gogetx %s <module-or-package>

Examples:
  gogetx %s go.uber.org/zap
  gogetx %s google.golang.org/grpc/status`, commandName, commandName, commandName)
		}
		return fmt.Errorf("expected one module or package path, received %d", len(args))
	}
}

func resolveModulePath(cmd *cobra.Command, app *App, input string) (string, error) {
	return app.Resolver.Resolve(cmd.Context(), packageinfo.PackageCandidate{
		PackagePath: input,
	})
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
	command.Flags().IntVar(&opts.Limit, "limit", packageinfo.DefaultSearchLimit, "search results per page")
	command.Flags().StringVar(&opts.Source, "source", packageinfo.SourcePkgsite, "search source: pkgsite, github, or all")
	command.Flags().BoolVar(&opts.NoCache, "no-cache", false, "disable search cache")
	command.Flags().BoolVar(&opts.Refresh, "refresh", false, "refresh cached search results")
}

func searchResultsPage(cmd *cobra.Command, app *App, keyword string, opts packageinfo.SearchOptions, pageNumber int, allPages bool) (packageinfo.SearchPage, error) {
	if !allPages {
		return searchPageNumber(cmd, app.Searcher, keyword, opts, pageNumber)
	}
	return collectSearchPages(cmd, app.Searcher, keyword, opts)
}

func searchPageNumber(cmd *cobra.Command, searcher Searcher, keyword string, opts packageinfo.SearchOptions, pageNumber int) (packageinfo.SearchPage, error) {
	if pageNumber < 1 {
		return packageinfo.SearchPage{}, fmt.Errorf("page must be 1 or greater")
	}

	opts.PageToken = ""
	var page packageinfo.SearchPage
	for current := 1; current <= pageNumber; current++ {
		var err error
		page, err = searchPage(cmd, searcher, keyword, opts)
		if err != nil {
			return packageinfo.SearchPage{}, err
		}
		if current == pageNumber {
			return page, nil
		}
		if page.NextPageToken == "" {
			return packageinfo.SearchPage{}, fmt.Errorf("page %d is not available; only %d page(s) found", pageNumber, current)
		}
		opts.PageToken = page.NextPageToken
	}
	return page, nil
}

func collectSearchPages(cmd *cobra.Command, searcher Searcher, keyword string, opts packageinfo.SearchOptions) (packageinfo.SearchPage, error) {
	opts.PageToken = ""
	var combined []packageinfo.PackageCandidate
	var total int
	seenResults := map[string]bool{}
	seenTokens := map[string]bool{}

	for {
		page, err := searchPage(cmd, searcher, keyword, opts)
		if err != nil {
			return packageinfo.SearchPage{}, err
		}
		total = page.Total
		for _, result := range page.Results {
			key := result.ModulePath + "|" + result.PackagePath
			if seenResults[key] {
				continue
			}
			seenResults[key] = true
			combined = append(combined, result)
		}
		if page.NextPageToken == "" {
			return packageinfo.SearchPage{Results: combined, Total: total}, nil
		}
		if seenTokens[page.NextPageToken] {
			return packageinfo.SearchPage{}, fmt.Errorf("search pagination returned a repeated page token")
		}
		seenTokens[page.NextPageToken] = true
		opts.PageToken = page.NextPageToken
	}
}

func searchPage(cmd *cobra.Command, searcher Searcher, keyword string, opts packageinfo.SearchOptions) (packageinfo.SearchPage, error) {
	if paged, ok := searcher.(PagedSearcher); ok {
		return paged.SearchPage(cmd.Context(), keyword, opts)
	}
	results, err := searcher.Search(cmd.Context(), keyword, opts)
	if err != nil {
		return packageinfo.SearchPage{}, err
	}
	return packageinfo.SearchPage{Results: results}, nil
}

func selectCandidate(cmd *cobra.Command, app *App, keyword string, opts packageinfo.SearchOptions, first bool, allPages bool) (packageinfo.PackageCandidate, error) {
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

	if allPages && !first {
		page, err := collectSearchPages(cmd, app.Searcher, keyword, opts)
		if err != nil {
			return packageinfo.PackageCandidate{}, err
		}
		if len(page.Results) == 0 {
			return packageinfo.PackageCandidate{}, fmt.Errorf("no results found for %q", keyword)
		}
		if len(page.Results) == 1 {
			fmt.Fprintf(cmd.OutOrStdout(), "Selected: %s\n", page.Results[0].PackagePath)
			return page.Results[0], nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Loaded %d results.\n", len(page.Results))
		return app.Selector.Select(page.Results)
	}

	opts.PageToken = ""
	var allResults []packageinfo.PackageCandidate
	seenResults := map[string]bool{}

	for {
		page, err := searchPage(cmd, app.Searcher, keyword, opts)
		if err != nil {
			return packageinfo.PackageCandidate{}, err
		}
		for _, result := range page.Results {
			key := result.ModulePath + "|" + result.PackagePath
			if seenResults[key] {
				continue
			}
			seenResults[key] = true
			allResults = append(allResults, result)
		}
		if len(allResults) == 0 {
			return packageinfo.PackageCandidate{}, fmt.Errorf("no results found for %q", keyword)
		}
		if first || len(allResults) == 1 {
			fmt.Fprintf(cmd.OutOrStdout(), "Selected: %s\n", allResults[0].PackagePath)
			return allResults[0], nil
		}

		selectionResults := appendLoadMoreCandidate(allResults, page.NextPageToken)
		selected, err := app.Selector.Select(selectionResults)
		if err != nil {
			return packageinfo.PackageCandidate{}, err
		}
		if !isLoadMoreCandidate(selected) {
			return selected, nil
		}
		if page.NextPageToken == "" {
			return packageinfo.PackageCandidate{}, fmt.Errorf("no more results can be loaded; try a more specific keyword")
		}
		opts.PageToken = page.NextPageToken
		fmt.Fprintln(cmd.OutOrStdout(), "Loading more results...")
	}
}

func appendLoadMoreCandidate(results []packageinfo.PackageCandidate, nextPageToken string) []packageinfo.PackageCandidate {
	if nextPageToken == "" {
		return results
	}
	out := make([]packageinfo.PackageCandidate, 0, len(results)+1)
	out = append(out, results...)
	out = append(out, packageinfo.PackageCandidate{
		PackagePath: "Load more results",
		Synopsis:    "Fetch the next page for this keyword.",
		Source:      loadMoreSource,
	})
	return out
}

func isLoadMoreCandidate(candidate packageinfo.PackageCandidate) bool {
	return candidate.Source == loadMoreSource
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

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if !strings.ContainsAny(value, " \t\n\"'\\$&;()<>|*?[]{}!") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
