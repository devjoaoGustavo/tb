# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
