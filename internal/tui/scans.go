package tui

import (
	"context"
	"fmt"
	"io"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fakeapate/pry/internal/store/db"
)

// scanItem wraps a DB scan record for the list component.
type scanItem struct {
	scan db.ScanWithCount
}

func (s scanItem) FilterValue() string { return s.scan.Url }

// scanDelegate renders each scan item in the list.
type scanDelegate struct {
	dark bool
}

func (d scanDelegate) Height() int  { return 1 }
func (d scanDelegate) Spacing() int { return 0 }

func (d scanDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d scanDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(scanItem)
	if !ok {
		return
	}
	sc := si.scan

	selected := index == m.Index()
	width := m.Width()

	const (
		statusW   = 9
		findingsW = 14
		dateW     = 12
		gap       = 1
	)
	urlW := width - 2 - statusW - findingsW - dateW - gap*3
	if urlW < 10 {
		urlW = 10
	}

	// Status cell
	statusColour := muted(d.dark)
	switch sc.Status {
	case "RUNNING":
		statusColour = accent(d.dark)
	case "DONE":
		statusColour = success(d.dark)
	case "FAILED":
		statusColour = errorC(d.dark)
	}
	statusCell := lipgloss.NewStyle().Foreground(statusColour).Width(statusW).Render(sc.Status)

	// URL cell (fills remaining width)
	url := sc.Url
	if len(url) > urlW {
		url = url[:urlW-1] + "~"
	}
	urlStyle := lipgloss.NewStyle().Width(urlW)
	if selected {
		urlStyle = urlStyle.Bold(true).Foreground(accent(d.dark))
	}
	urlCell := urlStyle.Render(url)

	// Findings cell (right-aligned)
	findings := fmt.Sprintf("%d findings", sc.FindingsCount)
	if sc.Status == "RUNNING" || sc.Status == "PENDING" {
		findings = "scanning"
	}
	findingsCell := lipgloss.NewStyle().
		Foreground(muted(d.dark)).
		Width(findingsW).
		Align(lipgloss.Right).
		Render(findings)

	// Date cell (right-aligned)
	dateCell := lipgloss.NewStyle().
		Foreground(muted(d.dark)).
		Width(dateW).
		Align(lipgloss.Right).
		Render(formatTime(sc.CreatedAt))

	fmt.Fprintf(w, "  %s %s %s %s", statusCell, urlCell, findingsCell, dateCell)
}

type scansTab struct {
	list    list.Model
	queries *db.Queries
	scans   []db.ScanWithCount
	loading bool
	err     error
	dark    bool
}

func newScansTab(queries *db.Queries) *scansTab {
	delegate := scanDelegate{}
	l := list.New(nil, delegate, 80, 20)
	l.Title = ""
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return &scansTab{
		list:    l,
		queries: queries,
		loading: true,
	}
}

func (s *scansTab) Title() string   { return "Scans" }
func (s *scansTab) Closeable() bool { return false }

func (s *scansTab) Init() tea.Cmd {
	return s.loadScans()
}

func (s *scansTab) Update(msg tea.Msg) (tabView, tea.Cmd) {
	switch msg := msg.(type) {
	case scansLoadedMsg:
		s.loading = false
		if msg.err != nil {
			s.err = msg.err
			return s, nil
		}
		s.scans = msg.scans
		items := make([]list.Item, len(s.scans))
		for i, sc := range s.scans {
			items[i] = scanItem{scan: sc}
		}
		cmd := s.list.SetItems(items)
		interval := idleRefreshInterval
		if hasActive(s.scans) {
			interval = activeRefreshInterval
		}
		return s, tea.Batch(cmd, tickRefreshEvery(interval))

	case refreshTickMsg:
		return s, s.loadScans()

	case tea.KeyPressMsg:
		// Don't intercept keys when list is filtering
		if s.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			s.list, cmd = s.list.Update(msg)
			return s, cmd
		}

		switch {
		case key.Matches(msg, scanListKeys.Refresh):
			s.loading = true
			return s, s.loadScans()

		case key.Matches(msg, scanListKeys.New):
			return s, func() tea.Msg { return openInputMsg{} }

		case key.Matches(msg, scanListKeys.Open):
			if sc := s.selected(); sc != nil {
				return s, func() tea.Msg {
					return openFindingsMsg{scanID: sc.ScanID, url: sc.Url}
				}
			}
			return s, nil

		case key.Matches(msg, scanListKeys.Delete):
			if sc := s.selected(); sc != nil {
				return s, s.deleteScan(sc.ScanID)
			}
			return s, nil
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft && msg.Y >= 0 {
			// scanDelegate renders one line per item with zero spacing, so the
			// click's Y coordinate (already translated to content-space) maps
			// directly to a row index.
			clickedIdx := msg.Y
			if clickedIdx < len(s.list.VisibleItems()) {
				if sc, ok := s.list.VisibleItems()[clickedIdx].(scanItem); ok {
					return s, func() tea.Msg {
						return openFindingsMsg{scanID: sc.scan.ScanID, url: sc.scan.Url}
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *scansTab) View(width, height int, dark bool) string {
	if s.err != nil {
		msg := lipgloss.NewStyle().Foreground(errorC(dark)).Render("Error: ") + s.err.Error()
		return styleContent.Render(msg)
	}
	if s.loading && len(s.scans) == 0 {
		return styleContent.Render(styleMuted(dark) + " Loading…")
	}

	// Only re-set the delegate when dark-mode changes, to avoid allocating
	// one per render.
	if s.dark != dark {
		s.dark = dark
		s.list.SetDelegate(scanDelegate{dark: dark})
	}
	s.list.SetWidth(width)
	s.list.SetHeight(height)

	return s.list.View()
}

func (s *scansTab) selected() *db.ScanWithCount {
	item := s.list.SelectedItem()
	if item == nil {
		return nil
	}
	si, ok := item.(scanItem)
	if !ok {
		return nil
	}
	return &si.scan
}

func (s *scansTab) loadScans() tea.Cmd {
	q := s.queries
	return func() tea.Msg {
		scans, err := q.ListScansWithCount(context.Background())
		return scansLoadedMsg{scans: scans, err: err}
	}
}

func (s *scansTab) deleteScan(scanID string) tea.Cmd {
	q := s.queries
	return func() tea.Msg {
		_ = q.DeleteScan(context.Background(), scanID)
		scans, err := q.ListScansWithCount(context.Background())
		return scansLoadedMsg{scans: scans, err: err}
	}
}

const (
	activeRefreshInterval = 2 * time.Second
	idleRefreshInterval   = 5 * time.Second
)

func tickRefreshEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func hasActive(scans []db.ScanWithCount) bool {
	for _, s := range scans {
		if s.Status == "RUNNING" || s.Status == "PENDING" {
			return true
		}
	}
	return false
}

func formatTime(ts string) string {
	t, err := time.Parse("2006-01-02 15:04:05", ts)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05Z", ts)
		if err != nil {
			return ts
		}
	}
	return fmt.Sprintf("%02d-%02d %02d:%02d", t.Month(), t.Day(), t.Hour(), t.Minute())
}

func styleMuted(dark bool) string {
	return lipgloss.NewStyle().Foreground(muted(dark)).Render("●")
}
