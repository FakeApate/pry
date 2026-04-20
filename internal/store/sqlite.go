package store

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// OpenDB opens (or creates) a SQLite database at the given path
// with WAL journal mode and foreign keys enabled.
func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}
