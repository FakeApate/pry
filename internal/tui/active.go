package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fakeapate/pry/model"
)

// activeModel shows a live view of a running scan.
type activeModel struct {
	scanID    string
	url       string
	folders   int64
	findings  int64
	skipped   int64
	errors    int64
	warnings  int64
	done      bool
	failed    bool
	reason    string
	startedAt time.Time
	elapsed   time.Duration
}

func newActiveModel(scanID, url string) *activeModel {
	return &activeModel{
		scanID:    scanID,
		url:       url,
		startedAt: time.Now(),
	}
}

func (a *activeModel) Title() string {
	short := a.url
	if len(short) > 22 {
		short = "…" + short[len(short)-21:]
	}
	return "scan: " + short
}

func (a *activeModel) Closeable() bool { return a.done || a.failed }

func (a *activeModel) Init() tea.Cmd {
	return a.tickCmd()
}

func (a *activeModel) tickCmd() tea.Cmd {
	if a.done || a.failed {
		return nil
	}
	scanID := a.scanID
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return elapsedTickMsg{scanID: scanID}
	})
}

func (a *activeModel) Update(msg tea.Msg) (tabView, tea.Cmd) {
	switch msg := msg.(type) {
	case elapsedTickMsg:
		if msg.scanID != a.scanID {
			return a, nil
		}
		a.elapsed = time.Since(a.startedAt)
		return a, a.tickCmd()

	case model.ScanProgressEvent:
		if msg.ScanID != a.scanID {
			return a, nil
		}
		a.folders = msg.Folders
		a.findings = msg.Findings
		a.skipped = msg.Skipped
		a.errors = msg.Errors
		a.warnings = msg.Warnings
		return a, nil

	case model.ScanDoneEvent:
		if msg.ScanID != a.scanID {
			return a, nil
		}
		a.done = true
		a.elapsed = time.Since(a.startedAt)
		a.folders = int64(msg.Result.Stats.FolderCount)
		a.findings = int64(msg.Result.Stats.FindingCount)
		a.skipped = int64(msg.Result.Stats.SkippedCount)
		a.errors = int64(msg.Result.Stats.ErrorCount)
		a.warnings = int64(msg.Result.Stats.WarningCount)
		return a, nil

	case model.ScanFailedEvent:
		if msg.ScanID != a.scanID {
			return a, nil
		}
		a.failed = true
		a.reason = msg.Reason
		a.elapsed = time.Since(a.startedAt)
		return a, nil

	case tea.KeyPressMsg:
		if a.done && key.Matches(msg, activeKeys.ViewFindings) {
			return a, func() tea.Msg {
				return openFindingsMsg{scanID: a.scanID, url: a.url}
			}
		}
	}
	return a, nil
}

func (a *activeModel) View(width, height int, dark bool) string {
	var sb strings.Builder

	// Status line
	status := "RUNNING"
	statusColor := accent(dark)
	if a.done {
		status = "DONE"
		statusColor = success(dark)
	} else if a.failed {
		status = "FAILED"
		statusColor = errorC(dark)
	}
	statusStr := lipgloss.NewStyle().Bold(true).Foreground(statusColor).Render(status)
	sb.WriteString(statusStr + "  " + a.url + "\n\n")

	// Progress counters
	sb.WriteString(fmt.Sprintf(
		"Folders: %d   Findings: %d   Skipped: %d   Errors: %d\n",
		a.folders, a.findings, a.skipped, a.errors,
	))

	// Warning line -- only shown when the server pushed back (429/5xx seen)
	if a.warnings > 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(warning(dark)).Render(
			fmt.Sprintf("Warnings: %d  (server returned 429 or 5xx)", a.warnings),
		) + "\n")
	}

	// Elapsed
	sb.WriteString(fmt.Sprintf("Elapsed: %s\n", fmtDuration(a.elapsed)))

	// Final state details
	if a.failed && a.reason != "" {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(errorC(dark)).Render("Error: "+a.reason) + "\n")
	}
	if a.done {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(muted(dark)).Render(
			"enter to view findings   ctrl+w to close",
		) + "\n")
	}
	if !a.done && !a.failed {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(muted(dark)).Render(
			"scan is running…",
		) + "\n")
	}

	return styleContent.Render(sb.String())
}

func fmtDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
