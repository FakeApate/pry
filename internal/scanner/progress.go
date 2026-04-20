package scanner

// ProgressEvent holds a snapshot of scan progress counters.
type ProgressEvent struct {
	ScanID       string
	FindingCount int
	FolderCount  int
	SkippedCount int
	ErrorCount   int
	WarningCount int
	TotalBytes   int64
}

// ProgressFunc is called periodically during a scan with a progress snapshot.
// Implementations must be safe for concurrent use.
type ProgressFunc func(ProgressEvent)
