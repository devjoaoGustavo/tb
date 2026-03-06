# Contributing

## Development setup

```sh
git clone https://github.com/joaogustavo/tb
cd tb
go build ./...   # must compile cleanly before you start
go test ./...    # must pass
```

Go 1.22+ required. No other tooling needed — `modernc.org/sqlite` is pure Go (no CGO).

Use environment variables to keep dev data fully separate from your real `tb` data:

```sh
export TB_CONFIG_DIR=/tmp/tb-dev/config
export TB_DATA_DIR=/tmp/tb-dev/data
go run ./cmd/tb init
```

## Adding a new command

Every command lives in `cmd/tb/main.go` as a `newXxxCmd() *cobra.Command` constructor. The pattern is:

```go
func newFooCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "foo <arg>",
        Short: "One-line description",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            st, cfg, paths, err := openStore()
            if err != nil {
                return err
            }
            defer st.Close()
            // ...
            return nil
        },
    }
    cmd.Flags().String("some-flag", "", "description")
    return cmd
}
```

Register it in `newRootCmd()`, or as a subcommand of an existing group (`cmd.AddCommand(...)`).

If the argument refers to an existing entity (project ID, client ID, invoice number), wire up shell completion:

```go
cmd.ValidArgsFunction = completeProjectIDs   // positional arg
cmd.RegisterFlagCompletionFunc("client", completeClientIDs)  // flag value
```

The three completion helpers (`completeProjectIDs`, `completeClientIDs`, `completeInvoiceNumbers`) are at the bottom of `main.go`. Add a new one if you introduce a new entity type.

## Adding a store method

Store methods live in `internal/store/store.go`, grouped by entity with a `// --- Entity ---` comment header. The conventions are:

- Use `nullableTime(t)` to write `time.Time` values that can be zero (stored as NULL).
- Use `time.Parse(time.RFC3339, s)` to read them back; ignore the error (zero value on failure is fine for optional fields).
- Cascade deletes must be done in a transaction in Go — the schema has `FOREIGN KEY` constraints with no `ON DELETE CASCADE`, so deleting a parent without first removing children will fail.
- Per-client sequences use `INSERT OR IGNORE` + `SELECT` + `UPDATE` in a single transaction (`NextClientSequence`). Follow the same pattern for any other atomic counter.

## Adding a schema table

Add the `CREATE TABLE IF NOT EXISTS` statement to the `migrate()` function in `internal/store/store.go`. Migrations are run on every `store.Open()` call — `IF NOT EXISTS` makes them idempotent. There is no migration versioning; all statements are safe to re-run.

## Testing

Tests use only the standard library. The existing patterns to follow:

```go
// Isolate the filesystem — never rely on $HOME
t.Setenv("HOME", t.TempDir())
t.Setenv("TB_CONFIG_DIR", t.TempDir())
t.Setenv("TB_DATA_DIR", t.TempDir())

// Name: TestFunctionName_Scenario
func TestNextNumber_PerClientPrefix(t *testing.T) { ... }

// Prefer t.Errorf for non-fatal assertions, t.Fatalf when the test cannot continue
```

The `internal/store` package currently has no test file. Tests there should open a real SQLite database in `t.TempDir()`:

```go
st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
if err != nil {
    t.Fatal(err)
}
defer st.Close()
```

Run a single package during development:

```sh
go test ./internal/store
go test ./internal/numbering -run TestPreview
```

## Releasing

Releases are automated. When you are ready to ship:

1. **Update `CHANGELOG.md`** — fill in the `[Unreleased]` section with a summary of changes grouped by `Added`, `Changed`, `Fixed`, or `Removed`.
2. **Bump `VERSION`** — edit the `VERSION` file (e.g. `0.2.0`). Follow [Semantic Versioning](https://semver.org): patch for bug-fixes, minor for new features, major for breaking changes.
3. **Push to `main`** — the release workflow does the rest automatically.

The workflow skips silently if the tag already exists or `[Unreleased]` is empty, so normal development pushes to `main` are safe. After the release, the workflow commits the updated `CHANGELOG.md` back to `main` (moving the `[Unreleased]` entries to the new version section and leaving `[Unreleased]` empty).

## Pull requests

- Run `go vet ./...` and `go test ./...` before opening a PR — both must pass.
- Keep commits focused. One logical change per commit.
- PR descriptions should be concise (a few sentences on what and why). The reviewer will ask if they need more detail.
