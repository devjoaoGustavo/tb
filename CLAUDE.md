# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build (embeds version from git tag via ldflags)
make build          # → ./tb binary
make install        # go install to $GOPATH/bin
make test           # go test ./...
make dist           # cross-compile to dist/ for all platforms
make release        # dist + gh release create (requires git tag + gh auth)

# Without Make
go build ./...
go test ./...
go test ./internal/config    # single package
go test ./internal/numbering

# Run the sample app (writes invoice-hourly.html + invoice-fixed.html to cwd)
go run ./cmd/sample

# Smoke-test the CLI locally
TB_CONFIG_DIR=/tmp/tb-test/config TB_DATA_DIR=/tmp/tb-test/data go run ./cmd/tb <command>
```

## Architecture

**tb** is a Go CLI for freelancers: track time → generate HTML invoices. One real dependency: `modernc.org/sqlite` (pure-Go, no CGO). Everything else is stdlib + Cobra.

### Package layout

- `internal/model` — All domain types. Read this first. `BillingType` (hourly/fixed), `InvoiceStatus` (draft/sent/paid), `Currency` (BRL/USD/EUR). `Session.Duration()` / `Session.Hours()` return live values when `End` is zero.
- `internal/config` — `Config` (issuer info, invoice defaults) lives at `~/.config/tb/config.json`; `State` at `~/.local/share/tb/state.json`. Both paths are overridable via `TB_CONFIG_DIR` / `TB_DATA_DIR`. `ResolvePaths(cfg)` is the single source of truth for all file locations including `DBFile` and `InvoiceDir`.
- `internal/numbering` — Formats invoice numbers from a sequence integer + config rules (prefix, year, padding, separator, per-client prefix).
- `internal/invoice` — `Render(w, inv, project, client, issuer)` executes an embedded `html/template` against an `InvoiceData` view model. `RenderToFile` is a convenience wrapper. The template lives entirely in `template.go`.
- `internal/store` — SQLite persistence via `database/sql`. `Open(path)` runs `migrate()` on every startup (idempotent `CREATE TABLE IF NOT EXISTS`). WAL mode, foreign keys enabled. `line_items` are stored as a JSON blob. The `sequences` table provides per-client atomic invoice counters via `NextClientSequence` / `PeekClientSequence`.
- `cmd/tb/main.go` — The entire CLI. Each command is a `newXxxCmd()` constructor returning `*cobra.Command`. The `openStore()` helper is called at the top of every RunE that needs persistence. Shell completions are wired via `ValidArgsFunction` and `RegisterFlagCompletionFunc` using three helpers at the bottom of the file: `completeProjectIDs`, `completeClientIDs`, `completeInvoiceNumbers`.
- `cmd/sample` — Standalone demo; constructs model objects directly and calls `invoice.Render`.

### Data flow

1. `openStore()` → `config.Load()` + `store.Open(paths.DBFile)`
2. Commands read/write via `*store.Store` methods
3. `invoice create` → `store.NextClientSequence` → `numbering.FormatNumber` → `invoice.RenderToFile` → `store.CreateInvoice` → `store.MarkSessionsBilled`

### Key conventions

- IDs for clients and projects are slugs derived via `toSlug()` (lowercase, hyphens). Users see and type these everywhere.
- `Session.End` being zero means the session is still running. `ActiveSession()` returns the single running session (enforced by a partial unique index on `sessions`).
- Deletes cascade in code, not via SQLite FK cascades: `DeleteClient` manually removes sessions → projects → invoices → sequences → client inside a transaction.
- `invoice create` / `invoice preview` share `runInvoiceCreate(cmd, preview bool)`. Preview uses `PeekClientSequence` (no increment) and writes to a temp file.
- `version` var in `cmd/tb/main.go` is set at build time: `-ldflags "-X main.version=<tag>"`.

### Release workflow

```bash
git tag v0.1.0 && git push origin v0.1.0
make release   # VERSION auto-detected from git tag
```
