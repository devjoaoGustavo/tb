# tb — Time Bill

A CLI for freelancers to track time and generate professional HTML invoices.
All data is stored locally — no accounts, no cloud, no subscriptions.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/devjoaoGustavo/tb/main/install.sh | sh
```

Or pin a specific version:

```sh
VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/devjoaoGustavo/tb/main/install.sh | sh
```

Requires `curl` or `wget`. Installs to `/usr/local/bin/tb` (uses `sudo` if needed).

### Shell completion

```sh
# zsh
echo 'source <(tb completion zsh)' >> ~/.zshrc

# bash
echo 'source <(tb completion bash)' >> ~/.bashrc
```

Once active, `tb start <TAB>` completes project IDs, `tb invoice open <TAB>` completes invoice numbers, and so on.

---

## Quick start

```sh
tb init                                              # interactive setup (issuer info + defaults)
tb client add                                        # interactive client registration
tb project add "API Work" --client acme-corp --rate 150 --currency USD

tb start api-work                                    # start timer
tb stop --note "auth module"                         # stop and annotate

tb invoice create --project api-work --period 2026-03   # create invoice for March
tb invoice open INV-2026-001                            # view in browser
```

---

## Usage

### Setup

```
tb init                   Interactive first-time setup (issuer info, invoice defaults)
tb config show            Print current config as JSON
tb config edit            Open config file in $EDITOR
```

### Clients

```
tb client add [name]      Add a client — interactive when called without flags
tb client list            List all clients
tb client delete <id>     Delete a client and all associated data [-f to skip prompt]
```

`add` flags: `--email`, `--company`, `--address`, `--phone`, `--tax-id`

Client IDs are auto-generated slugs from the name (`"ACME Corp"` → `acme-corp`).

### Projects

```
tb project add <name> --client <id>   Add a project
tb project list [--client <id>]       List projects, optionally filtered
tb project delete <id>                Delete a project and its sessions [-f to skip prompt]
```

`add` flags: `--client` (required), `--type` (hourly|fixed, default hourly), `--rate`, `--amount`, `--currency`

### Time tracking

```
tb start <project> [--at "2h ago"]   Start a timer (optionally backdated)
tb stop [--note "..."]               Stop the current timer
tb switch <project>                  Stop current timer and start another
tb now                               Show the running timer and today's total
tb log <project> <duration> [note]   Manually add a session (e.g. tb log api-work 2h30m)
tb cancel                            Discard the current running session
```

Only one timer can run at a time. `--at` accepts Go duration syntax followed by `ago` (e.g. `1h30m ago`).

### Reports

```
tb report [--week | --month | --from YYYY-MM-DD --to YYYY-MM-DD]
          [--client <id>] [--project <id>]
```

Shows hours and earnings grouped by project.

### Invoices

```
tb invoice create --project <id> [--period YYYY-MM] [-d "description"] [--due YYYY-MM-DD] [--fixed]
tb invoice preview --project <id> [--period YYYY-MM] [-d "description"]
tb invoice list [--status draft|sent|paid]
tb invoice show <number>
tb invoice mark-sent <number>
tb invoice mark-paid <number> [--date YYYY-MM-DD]
tb invoice open <number>
tb invoice delete <number> [-f]
```

- `--period` defaults to the current month.
- `-d` / `--description` sets the invoice notes. If omitted, `$EDITOR` opens so you can write it.
- `--due` overrides the configured default due date.
- `--fixed` forces fixed-price billing regardless of the project type.
- `preview` renders to a temp file and opens the browser — nothing is saved.
- `open` also regenerates the HTML file if it is missing.

Invoice numbers follow the format configured in `~/.config/tb/config.json` (default: `INV-2026-001`).

### Export

```
tb export [--format csv|json] [--month YYYY-MM]
```

Writes session data to stdout.

---

## Configuration

`tb init` creates `~/.config/tb/config.json`. Edit it directly or via `tb config edit`.

```jsonc
{
  "issuer": {
    "name": "Your Name",
    "email": "you@example.com",
    "title": "Freelance Developer",
    "phone": "+55 11 99999-9999",
    "address": "São Paulo, Brazil",
    "tax_id": "123.456.789-00"
  },
  "invoice": {
    "prefix": "INV",           // invoice number prefix
    "start_number": 1,
    "padding": 3,              // zero-pad width → 001
    "separator": "-",
    "include_year": true,      // INV-2026-001
    "per_client_prefix": false,// true → ACME-2026-001
    "default_due_days": 15,
    "default_tax_rate": 0.05,  // 5%
    "default_currency": "BRL",
    "default_notes": ""
  }
}
```

Override data and config paths with environment variables:

```sh
TB_CONFIG_DIR=/custom/config/path
TB_DATA_DIR=/custom/data/path
```

Data is stored at `~/.local/share/tb/`:

| Path | Contents |
|------|----------|
| `timebill.db` | SQLite database (clients, projects, sessions, invoices) |
| `invoices/` | Rendered HTML invoice files |

---

## Development

**Requirements:** Go 1.22+

```sh
git clone https://github.com/devjoaoGustavo/tb
cd tb
go build ./...        # verify everything compiles
go test ./...         # run tests
```

### Make targets

```
make build      Build ./tb binary with version from git tag
make install    go install to $GOPATH/bin
make uninstall  Remove from $GOPATH/bin
make test       Run all tests
make dist       Cross-compile to dist/ (darwin, linux, windows × amd64/arm64)
make release    dist + create GitHub release (requires gh auth login)
```

### Running locally

Use `TB_CONFIG_DIR` and `TB_DATA_DIR` to keep development data separate from your real data:

```sh
export TB_CONFIG_DIR=/tmp/tb-dev/config
export TB_DATA_DIR=/tmp/tb-dev/data
go run ./cmd/tb init
go run ./cmd/tb client add
```

### Releasing

Releases are fully automated via GitHub Actions. To cut a new release:

1. Add entries to the `[Unreleased]` section of `CHANGELOG.md`.
2. Bump `VERSION` (e.g. `echo "0.2.0" > VERSION`).
3. Commit and push to `main`.

```sh
# Example: releasing v0.2.0
echo "0.2.0" > VERSION
# edit CHANGELOG.md — add entries under [Unreleased]
git add VERSION CHANGELOG.md
git commit -m "prepare v0.2.0"
git push origin main
```

The release workflow will:
- Skip silently if the tag already exists or `[Unreleased]` is empty.
- Run the full test suite.
- Cross-compile binaries for all platforms.
- Create a git tag and GitHub release with the `[Unreleased]` content as release notes.
- Commit an updated `CHANGELOG.md` (moving `[Unreleased]` entries to `[0.2.0] - YYYY-MM-DD` and leaving `[Unreleased]` empty for the next cycle).

### Project layout

```
cmd/tb/         CLI entry point — all commands in main.go
cmd/sample/     Standalone demo that renders two example invoices
internal/
  model/        Domain types (Client, Project, Session, Invoice, …)
  config/       Config + path resolution (XDG-compliant)
  store/        SQLite persistence (modernc.org/sqlite, no CGO)
  numbering/    Invoice number formatting
  invoice/      HTML rendering (html/template + embedded CSS)
Makefile
install.sh      Curl-pipeable installer
```
