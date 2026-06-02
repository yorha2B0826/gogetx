# gogetx

`gogetx` is a Go module search and install CLI. It shortens the workflow from
"open pkg.go.dev, search, copy a module path, run go get" to one terminal
command.

## Install

```bash
go install github.com/yorha2B0826/gogetx@latest
```

For local development:

```bash
go run . search zap
go run . add zap --dry-run --yes
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
gogetx add echo --dry-run --yes
gogetx add grpc --version latest --yes
gogetx versions go.uber.org/zap
gogetx latest go.uber.org/zap
gogetx doc go.uber.org/zap
```

`add` only runs inside a Go module. If `go env GOMOD` returns `/dev/null`,
initialize the project first:

```bash
go mod init example.com/myapp
```

## Search Sources

The default source is `pkgsite`, backed by the official pkg.go.dev
`/v1beta/search` JSON API.

Available source values:

```text
pkgsite
github
all
```

GitHub fallback searches Go repositories, reads each repository's `go.mod`, and
uses the module directive as the candidate module path. Set `GITHUB_TOKEN` to
raise GitHub API rate limits:

```bash
GITHUB_TOKEN=your_token_here gogetx search cobra --source github
```

## Cache And Config

Search results are cached for 24 hours.

Default paths:

```text
cache:  os.UserCacheDir()/gogetx/search.json
config: os.UserConfigDir()/gogetx/config.yaml
```

Bypass or refresh cache:

```bash
gogetx search zap --no-cache
gogetx search zap --refresh
```

Favorites are stored in `config.yaml`:

```bash
gogetx addfav logger go.uber.org/zap
gogetx fav
gogetx add logger --dry-run --yes
gogetx rmfav logger
```

## Development

```bash
make test
make vet
make build
```

The tests avoid running real `go get`; command execution is injected behind a
runner interface.
