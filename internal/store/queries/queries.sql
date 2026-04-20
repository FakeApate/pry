-- name: CreateScan :one
INSERT INTO scans (scan_id, url, status)
VALUES (?, ?, 'PENDING')
RETURNING *;

-- name: GetScan :one
SELECT * FROM scans WHERE scan_id = ?;

-- name: ListScans :many
SELECT * FROM scans ORDER BY created_at DESC;

-- name: UpdateScanStatus :exec
UPDATE scans
SET status = ?, updated_at = datetime('now')
WHERE scan_id = ?;

-- name: CompleteScan :exec
UPDATE scans
SET status = 'DONE', result = ?, completed_at = datetime('now'), updated_at = datetime('now')
WHERE scan_id = ?;

-- name: FailScan :exec
UPDATE scans
SET status = 'FAILED', failure_reason = ?, completed_at = datetime('now'), updated_at = datetime('now')
WHERE scan_id = ?;

-- name: DeleteScan :exec
DELETE FROM scans WHERE scan_id = ?;
