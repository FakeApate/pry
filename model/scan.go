package model

import "time"

type ScanStatus string

const (
	ScanStatusPending ScanStatus = "PENDING"
	ScanStatusRunning ScanStatus = "RUNNING"
	ScanStatusDone    ScanStatus = "DONE"
	ScanStatusFailed  ScanStatus = "FAILED"
)

type ScanEntry struct {
	ScanID      string        `json:"scan_id"`
	Status      ScanStatus    `json:"status"`
	URL         string        `json:"url"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Result      []ScanFinding `json:"result,omitempty"`
}

type ScanFinding struct {
	Url           string    `json:"url"`
	ScanTime      time.Time `json:"scan_time"`
	ContentType   string    `json:"content_type"`
	ContentLength int64     `json:"content_length"`
	LastModified  time.Time `json:"last_modified"`
}

type ScanStats struct {
	DurationMs   int64 `json:"duration_ms"`
	FindingCount int   `json:"finding_count"`
	FolderCount  int   `json:"folder_count"`
	SkippedCount int   `json:"skipped_count"`
	ErrorCount   int   `json:"error_count"`
	WarningCount int   `json:"warning_count"`
	TotalBytes   int64 `json:"total_bytes"`
}

type ScanResult struct {
	Stats    ScanStats     `json:"stats"`
	Findings []ScanFinding `json:"findings"`
}
