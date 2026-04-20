package testutil

import (
	"database/sql"
	"testing"

	"github.com/fakeapate/pry/internal/store"
)

// OpenTestDB returns an in-memory SQLite database with all migrations applied.
// The database is automatically closed when the test completes.
func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	db.SetMaxOpenConns(1)

	if err := store.MigrateUp(db); err != nil {
		db.Close()
		t.Fatalf("migrate test db: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}
