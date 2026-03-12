# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-03-11

### Added

- **Multi-language invoice support** — invoices can now be rendered in 5 locales: `en`, `es`, `it`, `pt-BR`, `pt`
- New `internal/i18n` package with locale-aware money formatting (e.g., `R$ 5.700,00` for pt-BR, `$ 5,700.00` for en) and date formatting
- All invoice HTML labels are now localized: headers, table columns, totals, footer messages, and status badges
- Locale is configured via the `locale` field in `config.json` (defaults to `pt-BR`)

### Changed

- **Beautiful TUI output** — all CLI list/show/dashboard commands now use styled lipgloss tables with borders and colored headers
- Colored status badges in invoice list: `● Draft` (gray), `● Sent` (amber), `● Paid` (green)
- Success/action indicators: `✓` for completions, `▶` for timer start, `■` for timer stop
- Dashboard sections use styled headers with visual separators
- Invoice show uses styled key-value pairs

### Dependencies

- Added `github.com/charmbracelet/lipgloss` for terminal styling and table rendering

## [0.2.0] - 2026-03-06

### Added

- Simple dashboard with active job and other useful information

## [0.1.1] - 2026-03-06

### Added
- `tb init` — interactive first-time setup for issuer info and invoice defaults
- `tb client add/list/delete` — client management with interactive prompts and `--force` delete
- `tb project add/list/delete` — project management with hourly and fixed billing types
- `tb start/stop/switch/now/cancel` — timer tracking with optional backdating (`--at`)
- `tb log` — manual session entry by duration
- `tb report` — hours and earnings grouped by project, filterable by week/month/date range
- `tb invoice create/preview/list/show/open` — invoice generation with HTML rendering
- `tb invoice mark-sent/mark-paid/delete` — invoice lifecycle management
- `tb invoice delete` now requires `--force` for sent/paid invoices and unmarks sessions on deletion so they can be re-invoiced
- `tb export` — session export to CSV or JSON
- `tb config show/edit` — config inspection and editing
- Shell completion for project IDs, client IDs, and invoice numbers (bash and zsh)
- Per-client invoice number sequences stored in SQLite
- Cross-platform HTML invoice rendering with embedded CSS (print-ready A4)
- Local SQLite storage via `modernc.org/sqlite` (no CGO required)
