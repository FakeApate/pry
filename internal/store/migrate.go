package store

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// MigrateUp applies all pending up-migrations to the database.
func MigrateUp(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	var current int
	_ = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&current)

	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	// Collect up-migration files sorted by version number.
	type migration struct {
		version  int
		filename string
	}
	var ups []migration
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		parts := strings.SplitN(e.Name(), "_", 2)
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		ups = append(ups, migration{version: v, filename: e.Name()})
	}
	sort.Slice(ups, func(i, j int) bool { return ups[i].version < ups[j].version })

	for _, m := range ups {
		if m.version <= current {
			continue
		}
		data, err := migrationFiles.ReadFile("migrations/" + m.filename)
		if err != nil {
			return fmt.Errorf("migration %d: read file: %w", m.version, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("migration %d: begin tx: %w", m.version, err)
		}
		if _, err := tx.Exec(string(data)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", m.version, err)
		}
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.version); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: record version: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migration %d: commit: %w", m.version, err)
		}
	}

	return nil
}
