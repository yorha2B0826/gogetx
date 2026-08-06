# gogetx

语言 / Language: [中文](README.md) | [English](README.en.md)

`gogetx` 是一个 Go module 搜索与安装 CLI。它把“打开 pkg.go.dev 搜索、复制 module path、再运行 go get”的流程压缩到终端里完成。

它适合这些场景：

- 只记得包名或关键词，例如 `zap`、`echo`、`grpc`。
- 不确定真实 module path，例如 `go.uber.org/zap`、`github.com/labstack/echo/v4`。
- 希望先搜索、确认、再把依赖加入当前 Go module。

## 安装

需要 Go 1.23 或更高版本。

推荐安装固定版本：

```bash
go install github.com/yorha2B0826/gogetx@v0.1.8
```

也可以安装最新版本：

```bash
go install github.com/yorha2B0826/gogetx@latest
```

本地开发运行：

```bash
go run . search zap
go run . add zap --dry-run --first --yes
```

安装 shell 补全：

```bash
gogetx completion install
```

`completion install` 会根据当前 `$SHELL` 自动选择 shell，也可以显式指定：

```bash
gogetx completion install zsh
gogetx completion install bash
gogetx completion install fish
gogetx completion install powershell
```

## 常用命令

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

示例：

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

## `add` 的安全语义

`gogetx add` 只会在 Go module 目录里执行。如果 `go env GOMOD` 返回 `/dev/null`，请先初始化项目：

```bash
go mod init example.com/myapp
```

`--yes` 和 `--first` 的含义是分开的：

- `--yes`：跳过最终确认，不负责选择搜索结果。
- `--first`：明确选择第一条搜索结果，不进入交互选择。
- 脚本或非交互环境建议同时使用 `--first --yes`。
- 默认每页搜索 30 条结果；交互选择列表末尾出现 `Load more results` 时，选择它会继续加载下一页候选。
- `--all`：在进入交互选择前拉取所有可用搜索页（最多 25 页），适合想一次看完整结果集再筛选的情况。关键词很宽泛时会进行多次网络请求；超过页数上限会提示改用更具体的关键词。
- 默认不运行 `go mod tidy`，确保刚添加但尚未 import 的依赖会保留在 `go.mod` / `go.sum` 中。
- `--tidy`：在 `go get` 后额外运行 `go mod tidy`。如果项目源码还没有 import 该包，Go 可能会把刚添加的依赖移除。
- `--dry-run` 只打印命令，不执行 `go get` 或 `go mod tidy`。
- 不带 `--yes` 时，最终确认默认是 `Y/n`；直接回车会继续执行。

示例：

```bash
gogetx add zap --dry-run --first --yes
```

输出类似：

```text
Selected: go.uber.org/zap
Resolved module: go.uber.org/zap
Command: go get go.uber.org/zap@latest
```

交互选择列表会把每个候选项压缩为单行显示，避免过长简介在终端里换行后造成选项重绘偏移。选择器支持类似 fzf 的结果集内过滤：进入选择器后直接输入关键词即可筛选，匹配范围包括 package path、module path 和 synopsis；按方向键移动，按 Enter 选择。

## 搜索来源

默认搜索来源是 `pkgsite`，使用 pkg.go.dev 的 `/v1beta/search` JSON API。`--limit` 控制每页搜索结果数量，默认是 30。

分页搜索：

```bash
gogetx search air --limit 10 --page 1
gogetx search air --limit 10 --page 2
gogetx search air --all --limit 30
```

- `--page N`：显示第 N 页。pkg.go.dev 使用 token 分页，`gogetx` 会按顺序获取前面的 token 后再展示目标页。
- `--all`：持续获取所有可用页面并一次性输出（最多 25 页）。关键词很宽泛时可能会进行多次网络请求；超过页数上限会提示改用更具体的关键词。
- `search --json` 输出 `{ "results": [...], "total": N, "nextPageToken": "..." }`，便于脚本获取总数并翻页。
- 不带 `--all` 且还有后续页面时，输出末尾会给出下一页命令提示。

可用来源：

```text
pkgsite
github
all
```

GitHub fallback 会搜索 Go 仓库、读取仓库的 `go.mod`，再使用其中的 `module` directive 作为候选 module path。可以设置 `GITHUB_TOKEN` 提高 GitHub API 额度：

```bash
GITHUB_TOKEN=your_token_here gogetx search cobra --source github
```

## 版本查询与模块解析

`versions`、`latest` 以及 `add` 里的模块解析都直接查询 Go module proxy（默认 `https://proxy.golang.org`），不再依赖 `go list` 子进程，也不会往本地 module cache 写入内容。proxy 地址遵循 `GOPROXY` 环境变量（取第一个 `http(s)://` 条目），例如：

```bash
GOPROXY=https://goproxy.cn,direct gogetx versions go.uber.org/zap
```

## 缓存与配置

搜索结果默认缓存 24 小时。

默认路径：

```text
cache:  os.UserCacheDir()/gogetx/search.json
config: os.UserConfigDir()/gogetx/config.yaml
```

跳过或刷新缓存：

```bash
gogetx search zap --no-cache
gogetx search zap --refresh
```

收藏常用依赖：

```bash
gogetx addfav logger go.uber.org/zap
gogetx fav
gogetx add logger --dry-run --first --yes
gogetx rmfav logger
```

## 开发

```bash
make test
make vet
make build
```

测试不会真实执行 `go get`；Go 命令执行被封装在 runner 接口后面，测试里使用 fake runner。
