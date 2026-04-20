# pry

A single-binary CLI and TUI for scanning open HTTP directories, classifying
the files they contain, and exploring the results.

- Crawls directory listings with [Colly](https://github.com/gocolly/colly).
- Classifies every finding into a category (document, archive, database,
  source_code, config, ...) with an interest score 0–100.
- Stores scans in SQLite so past scans stay browsable.
- Ships a Bubble Tea TUI with table + tree views, filtering, sorting, and
  category-aware styling.
- Exports to self-contained interactive HTML, JSON, or CSV.
- Optional Mullvad SOCKS5 proxy rotation for anonymised scanning.

## Install

```sh
go install github.com/fakeapate/pry@latest
```

or grab a release binary from the goreleaser artefacts in the repo.

## Quick start

```sh
# interactive — browse past scans, dispatch new ones, inspect findings
pry

# headless scan of one or more URLs
pry scan https://example.com/files/

# export the most recent scan as a self-contained HTML report
pry export --last

# specific scan, JSON output
pry export <scan-id> --format json --output results.json
```

## TUI

Launch with no arguments. Three tab kinds:

- **Scans** — list of every scan in the database. `n` new, `enter` open, `d`
  delete, `r` refresh, `/` filter.
- **Active scan** — live counters while a scan runs. `enter` to jump to
  findings once done. Warnings (429 / 5xx) surface here when the server
  pushes back.
- **Findings** — table by default; press `v` for the tree view. Both support
  filtering, sorting, and click-to-sort column headers. `x` exports HTML.

Mouse is supported for scrolling, row selection, column-header sort, and tab
switching.

## Configuration

Print the defaults:

```sh
pry config generate > pry.toml
```

Key fields (edit `pry.toml`, then pass `--config pry.toml`):

| Field | Default | Purpose |
|-------|---------|---------|
| `workers` | 1 | concurrent scans dispatched in parallel |
| `disable_database` | false | skip SQLite persistence |
| `[Database].db_path` | `pry.db` | SQLite file location |
| `[Scanner].parallelism` | 32 | concurrent HTTP requests per scan |
| `[Scanner].request_timeout` | 15s | per-request timeout |
| `[Scanner].retry_count` | 2 | retries on 429 / 5xx |
| `[Scanner].retry_backoff` | 1s | base backoff (multiplied by attempt number) |
| `[Scanner].skip_mime_prefixes` | `image/ font/ text/css audio/ video/` | MIME prefixes to skip |
| `[Scanner].skip_subdir_keywords` | `.git node_modules venv ...` | substring filters for subdirectory paths |
| `[Mullvad]` | — | enables SOCKS5 proxy rotation when the host is on Mullvad |

The config is validated at startup; invalid values fail fast with an error
instead of running with broken settings.

## Classification

Each finding gets:

- **Category** — `document`, `archive`, `media`, `software`, `database`,
  `source_code`, `config`, or `other`.
- **Interest score (0–100)** — higher = more likely to warrant a look. Base
  score by category, with additive bonuses for rare extensions (`.sql`,
  `.env`, `.bak`), sensitive filenames (`password`, `secret`, `credential`),
  and large file sizes. Scores above 50 are flagged in the TUI with a
  warning colour; above 80, danger.
- **Tags** — optional secondary labels such as `sensitive`, `backup`, `log`.

Rules live in [`internal/classify/rules.go`](internal/classify/rules.go) — pure
lookup tables, easy to tweak.

## Export

The HTML exporter produces a single self-contained file with:

- Dark / light theme (follows `prefers-color-scheme`)
- Collapsible directory tree with expand-all / collapse-all
- Real-time filename search
- Click-to-sort column headers (name, category, size, modified, interest)
- Category pill toggles
- Interest badges on high-scoring files
- Double-click a file name to copy its full URL to the clipboard

Categories are rendered with muted, semantic styling — no scatter of decorative
accents. The theme follows the Tideline palette.

## Architecture

```
cmd/                 cobra commands: root (TUI), scan, export, config, backfill
config/              TOML-backed config with Validate()
model/               shared domain types (ScanStats, ScanFinding, events)
internal/
  orchestrator/      scan lifecycle, worker semaphore, persist path
  scanner/           two-phase Colly scanner (crawl + HEAD), retry, warnings
  classify/          pure rule-based categorisation + interest scoring
  tree/              in-memory tree from flat URLs with rollups
  store/             SQLite, migrations, SQLC queries, findings filter
  export/            HTML (embedded template), JSON, CSV
  tui/               Bubble Tea model, tabs, table, tree view, modal
```

## Development

```sh
make build            # binary into bin/pry
make test             # go test -race -count=1 ./...
make lint             # go vet ./...
make generate         # regenerate SQLC code + JSON schema types
```

Pre-commit hooks (gofmt, go vet, go mod tidy, standard file checks) are
configured in [`.pre-commit-config.yaml`](.pre-commit-config.yaml). Install
them once with:

```sh
pre-commit install
```

CI runs build, vet, `-race` tests, and a goreleaser snapshot build on every
push and PR.

## License

MIT — see [LICENSE](LICENSE).
