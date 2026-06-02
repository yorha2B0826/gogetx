# gogetx

[中文](README.md)

`gogetx` is a Go module search and install CLI. It moves the workflow of opening pkg.go.dev, searching, copying a module path, and running `go get` into one terminal tool.

It is useful when:

- You only remember a package name or keyword, such as `zap`, `echo`, or `grpc`.
- You are not sure about the real module path, such as `go.uber.org/zap` or `github.com/labstack/echo/v4`.
- You want to search, confirm, and then add a dependency to the current Go module.

## Installation

Go 1.23 or newer is required.

Recommended fixed version:

```bash
go install github.com/yorha2B0826/gogetx@v0.1.1
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
gogetx versions <module>
gogetx latest <module>
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
gogetx add zap
gogetx add echo --dry-run --first --yes
gogetx add grpc --version latest --first --yes
gogetx versions go.uber.org/zap
gogetx latest go.uber.org/zap
gogetx doc go.uber.org/zap
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

The interactive selection list renders each candidate as a single truncated line. This avoids redraw drift when long package descriptions would otherwise wrap in the terminal.

## Search Sources

The default source is `pkgsite`, backed by pkg.go.dev's `/v1beta/search` JSON API.

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
