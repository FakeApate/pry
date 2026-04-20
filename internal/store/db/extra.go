package db

import (
	"context"
	"database/sql"
)

// ScanWithCount extends Scan with a pre-joined findings count.
// This is a hand-written query, not sqlc-generated.
type ScanWithCount struct {
	ScanID        string
	Url           string
	Status        string
	CreatedAt     string
	UpdatedAt     string
	CompletedAt   sql.NullString
	Result        sql.NullString
	FailureReason sql.NullString
	FindingsCount int64
}

const listScansWithCount = `
SELECT s.scan_id, s.url, s.status, s.created_at, s.updated_at,
       s.completed_at, s.result, s.failure_reason,
       COUNT(f.id) AS findings_count
FROM scans s
LEFT JOIN scan_findings f ON f.scan_id = s.scan_id
GROUP BY s.scan_id
ORDER BY s.created_at DESC`

func (q *Queries) ListScansWithCount(ctx context.Context) ([]ScanWithCount, error) {
	rows, err := q.db.QueryContext(ctx, listScansWithCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ScanWithCount
	for rows.Next() {
		var i ScanWithCount
		if err := rows.Scan(
			&i.ScanID,
			&i.Url,
			&i.Status,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.CompletedAt,
			&i.Result,
			&i.FailureReason,
			&i.FindingsCount,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
