package tui

import (
	"github.com/fakeapate/pry/internal/store"
	"github.com/fakeapate/pry/internal/store/db"
)

// scan lifecycle events are in model.ScanProgressEvent / model.ScanDoneEvent / model.ScanFailedEvent
// (defined in model/events.go to avoid import cycles with orchestrator)

// DB load results — returned from tea.Cmd goroutines

type scansLoadedMsg struct {
	scans []db.ScanWithCount
	err   error
}

type findingsLoadedMsg struct {
	result store.FindingsResult
	err    error
}

// UI events

type refreshTickMsg struct{}

type scanDispatchedMsg struct {
	scanID string
	url    string
	err    error
}

type openInputMsg struct{}

type openFindingsMsg struct {
	scanID string
	url    string
}

type elapsedTickMsg struct {
	scanID string
}

type closeTabMsg struct{}

type exportDoneMsg struct {
	path string
	err  error
}

type treeDataMsg struct {
	findings []store.Finding
	err      error
}
