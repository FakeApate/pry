package store_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/fakeapate/pry/internal/store"
)

func seedFindings(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec("INSERT INTO scans (scan_id, url, status) VALUES ('scan-1', 'https://example.com/', 'DONE')")
	if err != nil {
		t.Fatal(err)
	}

	findings := []struct {
		url         string
		contentType string
		size        int64
		lastMod     string
	}{
		{"https://example.com/report.pdf", "application/pdf", 2048000, "2026-03-15T10:00:00Z"},
		{"https://example.com/data.csv", "text/csv", 512000, "2026-03-10T08:00:00Z"},
		{"https://example.com/backup.sql", "application/sql", 8300000, "2026-03-10T09:00:00Z"},
		{"https://example.com/image.png", "image/png", 150000, "2026-02-20T12:00:00Z"},
		{"https://example.com/readme.txt", "text/plain", 4096, "2026-04-01T00:00:00Z"},
		{"https://example.com/app.exe", "application/octet-stream", 52428800, "2026-01-15T06:00:00Z"},
		{"https://example.com/archive.tar.gz", "application/gzip", 104857600, "2026-04-10T14:00:00Z"},
		{"https://example.com/config.yaml", "text/yaml", 1024, "2026-03-20T11:00:00Z"},
		{"https://example.com/main.go", "text/x-go", 4200, "2026-04-01T16:00:00Z"},
		{"https://example.com/dump.sql", "application/sql", 1073741824, "2026-04-05T03:00:00Z"},
	}

	for _, f := range findings {
		_, err := db.Exec(
			"INSERT INTO scan_findings (scan_id, url, scan_time, content_type, content_length, last_modified) VALUES (?, ?, ?, ?, ?, ?)",
			"scan-1", f.url, "2026-04-10T10:00:00Z", f.contentType, f.size, f.lastMod,
		)
		if err != nil {
			t.Fatalf("seed finding %s: %v", f.url, err)
		}
	}
}

func TestQueryFindingsBasic(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	seedFindings(t, db)

	fs := store.NewFindingsStore(db)
	result, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:   "scan-1",
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 10 {
		t.Errorf("expected 10 total, got %d", result.Total)
	}
	if len(result.Findings) != 10 {
		t.Errorf("expected 10 findings, got %d", len(result.Findings))
	}
}

func TestQueryFindingsPagination(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	seedFindings(t, db)

	fs := store.NewFindingsStore(db)

	// Page 1 of 3
	r1, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:   "scan-1",
		Page:     1,
		PageSize: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(r1.Findings) != 3 {
		t.Errorf("page 1: expected 3 findings, got %d", len(r1.Findings))
	}
	if r1.Total != 10 {
		t.Errorf("page 1: expected total 10, got %d", r1.Total)
	}

	// Page 4 (last page, 1 item)
	r4, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:   "scan-1",
		Page:     4,
		PageSize: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(r4.Findings) != 1 {
		t.Errorf("page 4: expected 1 finding, got %d", len(r4.Findings))
	}

	// Page beyond range
	r5, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:   "scan-1",
		Page:     100,
		PageSize: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(r5.Findings) != 0 {
		t.Errorf("page 100: expected 0 findings, got %d", len(r5.Findings))
	}
	if r5.Total != 10 {
		t.Errorf("page 100: expected total 10, got %d", r5.Total)
	}
}

func TestQueryFindingsFilterContentType(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	seedFindings(t, db)

	fs := store.NewFindingsStore(db)
	result, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:       "scan-1",
		ContentTypes: []string{"application/sql"},
		Page:         1,
		PageSize:     100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 SQL findings, got %d", result.Total)
	}
	for _, f := range result.Findings {
		if f.ContentType != "application/sql" {
			t.Errorf("unexpected content type: %s", f.ContentType)
		}
	}
}

func TestQueryFindingsFilterSize(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	seedFindings(t, db)

	fs := store.NewFindingsStore(db)

	// Files larger than 10MB
	minSize := int64(10_000_000)
	result, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:   "scan-1",
		MinSize:  &minSize,
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 3 {
		t.Errorf("expected 3 large files (app.exe, archive.tar.gz, dump.sql), got %d", result.Total)
	}

	// Files smaller than 5000 bytes
	maxSize := int64(5000)
	result, err = fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:   "scan-1",
		MaxSize:  &maxSize,
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range result.Findings {
		if f.ContentLength > 5000 {
			t.Errorf("expected size <= 5000, got %d for %s", f.ContentLength, f.URL)
		}
	}
}

func TestQueryFindingsFilterQuery(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	seedFindings(t, db)

	fs := store.NewFindingsStore(db)
	q := ".sql"
	result, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:   "scan-1",
		Query:    &q,
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 .sql files, got %d", result.Total)
	}
}

func TestQueryFindingsFilterQueryEscaping(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}

	// Insert a scan and two findings whose names differ by a literal underscore.
	_, err := db.Exec("INSERT INTO scans (scan_id, url, status) VALUES ('scan-esc', 'https://example.com/', 'DONE')")
	if err != nil {
		t.Fatal(err)
	}
	urls := []string{
		"https://example.com/report_v1.pdf",
		"https://example.com/report1v1.pdf",
	}
	for _, u := range urls {
		if _, err := db.Exec(
			"INSERT INTO scan_findings (scan_id, url, scan_time, content_type, content_length, last_modified) VALUES (?, ?, ?, ?, ?, ?)",
			"scan-esc", u, "2026-04-10T10:00:00Z", "application/pdf", 1, "",
		); err != nil {
			t.Fatal(err)
		}
	}

	fs := store.NewFindingsStore(db)

	// Underscore must match a literal '_' only, not any single character.
	q := "report_"
	result, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID: "scan-esc", Query: &q, Page: 1, PageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 match for literal underscore, got %d", result.Total)
	}

	// Percent sign must likewise be literal.
	q = "%"
	result, err = fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID: "scan-esc", Query: &q, Page: 1, PageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 matches for literal '%%', got %d", result.Total)
	}
}

func TestQueryFindingsSortBySize(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	seedFindings(t, db)

	fs := store.NewFindingsStore(db)
	result, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:    "scan-1",
		SortBy:    "content_length",
		SortOrder: "desc",
		Page:      1,
		PageSize:  3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) < 2 {
		t.Fatal("expected at least 2 findings")
	}
	if result.Findings[0].ContentLength < result.Findings[1].ContentLength {
		t.Errorf("expected descending size order: %d < %d",
			result.Findings[0].ContentLength, result.Findings[1].ContentLength)
	}
}

func TestQueryFindingsCombinedFilters(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	seedFindings(t, db)

	fs := store.NewFindingsStore(db)
	minSize := int64(1000)
	q := "example.com"
	result, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:       "scan-1",
		ContentTypes: []string{"application/sql", "application/pdf"},
		MinSize:      &minSize,
		Query:        &q,
		SortBy:       "content_length",
		SortOrder:    "desc",
		Page:         1,
		PageSize:     100,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Should match: report.pdf (2MB), backup.sql (8.3MB), dump.sql (1GB) — all > 1000 bytes
	if result.Total != 3 {
		t.Errorf("expected 3 combined filter results, got %d", result.Total)
	}
}

func TestGetContentTypes(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	seedFindings(t, db)

	fs := store.NewFindingsStore(db)
	types, err := fs.GetContentTypes(context.Background(), "scan-1")
	if err != nil {
		t.Fatal(err)
	}

	// Should be sorted and distinct
	if len(types) == 0 {
		t.Fatal("expected content types, got none")
	}
	for i := 1; i < len(types); i++ {
		if types[i] <= types[i-1] {
			t.Errorf("content types not sorted: %q <= %q", types[i], types[i-1])
		}
	}
}

func TestQueryFindingsLastModified(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	seedFindings(t, db)

	fs := store.NewFindingsStore(db)
	result, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:    "scan-1",
		SortBy:    "last_modified",
		SortOrder: "desc",
		Page:      1,
		PageSize:  100,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify that LastModified is parsed correctly for at least the first result
	if len(result.Findings) == 0 {
		t.Fatal("no findings")
	}

	hasLastMod := false
	for _, f := range result.Findings {
		if f.LastModified != nil && !f.LastModified.IsZero() {
			hasLastMod = true
			if f.LastModified.Year() < 2026 || f.LastModified.Year() > 2030 {
				t.Errorf("unexpected last_modified year: %v", f.LastModified)
			}
		}
	}
	if !hasLastMod {
		t.Error("no findings had a parsed LastModified")
	}
}

func TestQueryFindingsEmptyScan(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}

	// Create a scan with no findings
	_, err := db.Exec("INSERT INTO scans (scan_id, url, status) VALUES ('empty', 'https://example.com/', 'DONE')")
	if err != nil {
		t.Fatal(err)
	}

	fs := store.NewFindingsStore(db)
	result, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:   "empty",
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 total, got %d", result.Total)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(result.Findings))
	}
}

func TestQueryFindingsScanTime(t *testing.T) {
	db := openMemDB(t)
	if err := store.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	seedFindings(t, db)

	fs := store.NewFindingsStore(db)
	result, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
		ScanID:   "scan-1",
		Page:     1,
		PageSize: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("no findings")
	}
	expected, _ := time.Parse(time.RFC3339, "2026-04-10T10:00:00Z")
	if !result.Findings[0].ScanTime.Equal(expected) {
		t.Errorf("expected scan time %v, got %v", expected, result.Findings[0].ScanTime)
	}
}
