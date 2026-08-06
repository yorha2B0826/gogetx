# Changelog

## v0.2.0 (2026-08-06)

Final release. This project is archived and no longer maintained.

- Resolve `versions`, `latest`, and `add` through the Go module proxy
  instead of `go list`; honor the `GOPROXY` environment variable.
- Fetch `--source all` sources concurrently.
- Cache: atomic writes, expired-entry pruning, and self-healing on
  corrupt cache files.
- Cap `--all` pagination at 25 pages.
- `search --json` now includes `total` and `nextPageToken`.
- `doc --print` prints the URL without opening a browser.
- Make fuzzy search rune-safe for multi-byte input.
- Add tests for the GitHub and Go proxy clients.

## v0.1.x (2026-06)

- v0.1.8: all-page interactive filtering
- v0.1.7: paginated search
- v0.1.6: expand interactive search results
- v0.1.5: improve missing module argument errors
- v0.1.4: add version command
- v0.1.3: resolve package paths for version commands
- v0.1.2: preserve added modules by default
- v0.1.1: improve prompts and completion install
- v0.1.0: initial release — search, add, favorites, doc, completion
