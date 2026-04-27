# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
make build              # go build -o bin/pry .
make test               # go test -race -count=1 ./...
make lint               # go vet ./...
make generate           # regenerate SQLC code + go-jsonschema types
make generate-sqlc      # only SQLC; run after editing queries/ or migrations/
make generate-jsonschema

go test -race -run TestFoo ./internal/classify   # single test
go test -race ./internal/scanner/...             # one package recursively
```

CI (`.github/workflows/ci.yml`) runs `go build ./...`, `go vet ./...`, and `go test -race -count=1 ./...` on push / PR to `main`. Releases are cut by pushing a tag; `.goreleaser.yaml` builds cross-platform binaries.

## Architecture

Single Go binary composed of:

- **`cmd/`** — Cobra entrypoints (`root` launches the TUI; `scan`, `export`, `backfill`, `config`, `version`). `root.go:initConfig` loads TOML via BurntSushi/toml into `config.AppConfig` and calls `Validate()` — bad config fails fast.
- **`config/`** — Process-global singleton (`GetConfig`/`SetConfig`). `AppConfig.Validate()` is the canonical check for usable settings.
- **`internal/orchestrator/`** — Singleton (`GetInstance`) that owns the DB, a `workerSem` semaphore sized to `cfg.Workers`, and a reference to the bubbletea program. `Dispatch(url)` creates a PENDING scan row, blocks on the semaphore, transitions to RUNNING, runs the scanner, then either `persistFindings` (classify + insert in a single tx + `CompleteScan`) or `failScan`. Progress/done/failed events are pushed to the TUI via `tea.Program.Send`.
- **`internal/scanner/`** — Two-phase Colly crawler:
  1. GET phase builds a flat list of discovered URLs by visiting the index page and recursing subdirectories. An index page is validated by `patterns.Detect`.
  2. HEAD phase (`tagC`) fills in `Content-Type`, `Content-Length`, `Last-Modified`, `Date`.
  Retries `429` and `5xx` up to `cfg.RetryCount` with linear backoff. Optional Mullvad SOCKS5 rotation is wired when the host is connected. `newBareScanner` exists so tests don't touch the host's Mullvad state.
- **`internal/scanner/patterns/`** — Index-page detectors (Apache, nginx, IIS, Caddy, Jetty, Tomcat, Lighttpd, Python http.server, Go fileserver, plus a loose generic fallback). Each registers via `init()` → `Register(...)`; order matters (specific before generic). `Detect` fans out Matches calls in goroutines and picks the first match in registration order. To add a new server: implement `IndexPattern`, register in `init()`, and add a testdata fixture.
- **`internal/classify/`** — Pure function `Classify(url, ct, size) → {Category, Tags, InterestScore}`. All thresholds and lookup tables live in `rules.go`. Used both live (orchestrator) and for `pry backfill` (re-classifies old findings with empty category).
- **`internal/tree/`** — Builds a nested directory tree from flat URLs with per-node rollups (size, count, max interest). Shared by TUI tree view and HTML export.
- **`internal/store/`** — SQLite via `modernc.org/sqlite` (pure Go, no cgo). `OpenDB` sets WAL + foreign keys and pins `SetMaxOpenConns(1)` — the whole app serialises through a single connection, so every query runs against the same writer. `MigrateUp` reads embedded `migrations/*.up.sql`, sorts by numeric prefix, and applies anything past `schema_migrations.version`. `findings.go` is the query layer for filtered/paged TUI listings that SQLC doesn't cover.
- **`internal/store/db/`** — **Generated** by SQLC from `internal/store/queries/*.sql` against `internal/store/migrations/*.up.sql` (see `resources/sqlc.yaml`). Never hand-edit. After changing queries or migrations, run `make generate-sqlc`.
- **`internal/export/`** — Pluggable `Exporter` interface, implementations for HTML (single self-contained file via embedded `html_template.html`), JSON, CSV.
- **`internal/tui/`** — Bubble Tea (`charm.land/bubbletea/v2`) root model dispatches to tab views (scans list, active scan, findings table/tree). Orchestrator pushes `model.ScanProgressEvent` / `ScanDoneEvent` / `ScanFailedEvent` that the root routes to the active tab by `ScanID`.
- **`model/`** — Shared domain types (`ScanStats`, `ScanFinding`, `ScanResult`, event messages). Kept outside `internal/` deliberately so `cmd/` and `internal/*` can share without import cycles.

## Workflow notes

- **Adding a schema change**: create `internal/store/migrations/NNN_name.up.sql` (monotonic integer prefix), edit any affected queries in `internal/store/queries/`, then `make generate-sqlc`. `MigrateUp` picks it up on next startup.
- **Changing classification rules**: edit `internal/classify/rules.go` and run `pry backfill` afterwards to re-tag findings from scans taken before the change.
- **Config flow**: `--config pry.toml` is decoded in `cmd/root.go:initConfig` into the singleton. Anything reading config at runtime calls `config.GetConfig()`. Invalid values fail at startup, not mid-scan.
- **Orchestrator is a singleton** — every command that touches the DB calls `orchestrator.GetInstance(nil)` (or opens its own `store.OpenDB` for read-only flows like `export`, `backfill`). `Close()` cancels in-flight scans, waits, then closes the DB.
- **Go version**: `go.mod` pins `go 1.25.9`. CI uses `go-version-file: go.mod`.
