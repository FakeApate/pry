package model

// Events sent from the orchestrator to the TUI via tea.Program.Send.
// Defined here (not in internal/tui) to avoid import cycles.

type ScanProgressEvent struct {
	ScanID   string
	Folders  int64
	Findings int64
	Skipped  int64
	Errors   int64
	Warnings int64
}

type ScanDoneEvent struct {
	ScanID string
	Result ScanResult
}

type ScanFailedEvent struct {
	ScanID string
	Reason string
}
