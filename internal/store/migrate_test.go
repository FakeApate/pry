package store_test

import (
	"database/sql"
	"testing"

	"github.com/fakeapate/pry/internal/store"

	_ "modernc.org/sqlite"
)

func openMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrateUp(t *testing.T) {
	db := openMemDB(t)

	if err := store.MigrateUp(db); err != nil {
		t.Fatalf("first migration: %v", err)
	}

	// Verify tables exist
	for _, table := range []string{"scans", "scan_findings", "schema_migrations"} {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}

	// Verify migration version recorded
	var version int
	if err := db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version < 1 {
		t.Errorf("expected version >= 1, got %d", version)
	}
}

func TestMigrateUpIdempotent(t *testing.T) {
	db := openMemDB(t)

	if err := store.MigrateUp(db); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := store.MigrateUp(db); err != nil {
		t.Fatalf("second migration (idempotency): %v", err)
	}

	// Insert a row to verify schema is intact after double migration
	_, err := db.Exec("INSERT INTO scans (scan_id, url, status) VALUES ('test-1', 'https://example.com/', 'PENDING')")
	if err != nil {
		t.Fatalf("insert after double migration: %v", err)
	}
}

func TestMigrateUpForeignKeys(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}

	// Inserting a finding with a non-existent scan_id should fail
	_, err := db.Exec("INSERT INTO scan_findings (scan_id, url) VALUES ('nonexistent', 'https://example.com/file.txt')")
	if err == nil {
		t.Error("expected foreign key violation, got nil")
	}
}

func TestMigrateUpCascadeDelete(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}

	// Insert scan + finding
	_, err := db.Exec("INSERT INTO scans (scan_id, url, status) VALUES ('s1', 'https://example.com/', 'DONE')")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO scan_findings (scan_id, url, content_type, content_length) VALUES ('s1', 'https://example.com/f.txt', 'text/plain', 100)")
	if err != nil {
		t.Fatal(err)
	}

	// Delete the scan — findings should cascade
	if _, err := db.Exec("DELETE FROM scans WHERE scan_id = 's1'"); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM scan_findings WHERE scan_id = 's1'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0 findings after cascade delete, got %d", count)
	}
}
