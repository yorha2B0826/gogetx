# gogetx

[中文](README.md)

> **Status: archived** — This project is no longer maintained and the repository is read-only. You can still install the final release with `go install github.com/yorha2B0826/gogetx@v0.2.0`, or clone it and run `make build`.

`gogetx` is a Go module search and install CLI. It moves the workflow of opening pkg.go.dev, searching, copying a module path, and running `go get` into one terminal tool.

It is useful when:

- You only remember a package name or keyword, such as `zap`, `echo`, or `grpc`.
- You are not sure about the real module path, such as `go.uber.org/zap` or `github.com/labstack/echo/v4`.
- You want to search, confirm, and then add a dependency to the current Go module.

## Installation

Go 1.23 or newer is required.

Recommended fixed version:

```bash
go install github.com/yorha2B0826/gogetx@v0.2.0
```

Latest version:

```bash
go install github.com/yorha2B0826/gogetx@latest
```

Local development:

```bash
go run . search zap
go run . add zap --dry-run --first --yes
```

Install shell completion:

```bash
gogetx completion install
```

`completion install` detects the current `$SHELL` automatically. You can also specify one explicitly:

```bash
gogetx completion install zsh
gogetx completion install bash
gogetx completion install fish
gogetx completion install powershell
```

## Commands

```bash
gogetx search <keyword>
gogetx add <keyword>
gogetx versions <module-or-package>
gogetx latest <module-or-package>
gogetx version
gogetx doc <target>
gogetx fav
gogetx addfav <alias> <module>
gogetx rmfav <alias>
```

Examples:

```bash
gogetx search zap
gogetx search logger --json
gogetx search echo --source all --limit 5
gogetx search air --limit 10 --page 3
gogetx search air --all --limit 30
gogetx add zap
gogetx add echo --dry-run --first --yes
gogetx add air --all
gogetx add grpc --version latest --first --yes
gogetx versions go.uber.org/zap
gogetx versions google.golang.org/grpc/status
gogetx latest go.uber.org/zap
gogetx latest google.golang.org/grpc/status
gogetx version
gogetx doc go.uber.org/zap
gogetx doc go.uber.org/zap --print
```

## Safe `add` Behavior

`gogetx add` only runs inside a Go module. If `go env GOMOD` returns `/dev/null`, initialize your project first:

```bash
go mod init example.com/myapp
```

`--yes` and `--first` are intentionally separate:

- `--yes`: skip the final confirmation; it does not choose a search result.
- `--first`: choose the first search result without interactive selection.
- Use `--first --yes` for scripts or non-interactive environments.
- The default page size is 30 results; when the interactive selector shows `Load more results`, choose it to load the next page of candidates.
- `--all`: fetch all available search pages (up to 25) before opening interactive selection. This is useful when you want the full result set first and then filter locally. Broad keywords may require multiple network requests; if the page cap is reached, gogetx tells you to use a more specific keyword.
- By default, `gogetx add` does not run `go mod tidy`, so newly added packages remain in `go.mod` / `go.sum` before you import them.
- `--tidy`: additionally run `go mod tidy` after `go get`. If your source code does not import the package yet, Go may remove the newly added dependency.
- `--dry-run` only prints commands and never runs `go get` or `go mod tidy`.
- Without `--yes`, the final confirmation defaults to `Y/n`; pressing Enter continues.

Example:

```bash
gogetx add zap --dry-run --first --yes
```

Example output:

```text
Selected: go.uber.org/zap
Resolved module: go.uber.org/zap
Command: go get go.uber.org/zap@latest
```

The interactive selection list renders each candidate as a single truncated line. This avoids redraw drift when long package descriptions would otherwise wrap in the terminal. The selector supports fzf-like filtering inside the current result set: type to filter by package path, module path, or synopsis; use the arrow keys to move and Enter to select.

## Search Sources

The default source is `pkgsite`, backed by pkg.go.dev's `/v1beta/search` JSON API. `--limit` controls the number of results per page and defaults to 30.

Paginated search:

```bash
gogetx search air --limit 10 --page 1
gogetx search air --limit 10 --page 2
gogetx search air --all --limit 30
```

- `--page N`: show page N. pkg.go.dev uses token pagination, so `gogetx` fetches prior page tokens before printing the requested page.
- `--all`: keep fetching available pages (up to 25) and print them together. Broad keywords may require multiple network requests; if the page cap is reached, gogetx tells you to use a more specific keyword.
- `search --json` prints `{ "results": [...], "total": N, "nextPageToken": "..." }`, so scripts can read the total and paginate.
- When `--all` is not used and another page exists, the output includes a next-page command hint.

Available sources:

```text
pkgsite
github
all
```

GitHub fallback searches Go repositories, reads each repository's `go.mod`, and uses the `module` directive as the candidate module path. Set `GITHUB_TOKEN` to raise GitHub API rate limits:

```bash
GITHUB_TOKEN=your_token_here gogetx search cobra --source github
```

## Versions And Module Resolution

`versions`, `latest`, and the module resolution used by `add` query the Go module proxy directly (default `https://proxy.golang.org`) instead of shelling out to `go list`, so nothing is written to your local module cache. The proxy URL honors the `GOPROXY` environment variable (the first `http(s)://` entry), for example:

```bash
GOPROXY=https://goproxy.cn,direct gogetx versions go.uber.org/zap
```

## Cache And Config

Search results are cached for 24 hours by default.

Default paths:

```text
cache:  os.UserCacheDir()/gogetx/search.json
config: os.UserConfigDir()/gogetx/config.yaml
```

Bypass or refresh the cache:

```bash
gogetx search zap --no-cache
gogetx search zap --refresh
```

Favorite modules:

```bash
gogetx addfav logger go.uber.org/zap
gogetx fav
gogetx add logger --dry-run --first --yes
gogetx rmfav logger
```

## Development

```bash
make test
make vet
make build
```

Tests do not run a real `go get`; Go command execution is wrapped behind a runner interface and replaced with fake runners in tests.
