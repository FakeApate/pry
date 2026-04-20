-- name: InsertScanFinding :exec
INSERT INTO scan_findings (scan_id, url, scan_time, content_type, content_length, last_modified, category, interest_score, tags)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CountScanFindings :one
SELECT COUNT(*) FROM scan_findings WHERE scan_id = ?;

-- name: ListScanFindings :many
SELECT * FROM scan_findings WHERE scan_id = ? ORDER BY url ASC;
