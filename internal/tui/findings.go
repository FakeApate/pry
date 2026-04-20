package tui

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/paginator"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fakeapate/pry/internal/export"
	"github.com/fakeapate/pry/internal/store"
	"github.com/fakeapate/pry/internal/tree"
)

const findingsPageSize = 100

// findingsSortCol maps column indices to their DB sort column names.
var findingsSortCol = []string{"url", "category", "content_length", "last_modified", "interest_score"}

type findingsTab struct {
	scanID      string
	url         string
	fs          *store.FindingsStore
	result      store.FindingsResult
	filter      store.FindingsFilter
	table       table.Model
	pager       paginator.Model
	loading     bool
	err         error
	filterOpen  bool
	filterInput textinput.Model
	colWidths   []int // cached column widths from last render
	showTree    bool
	treeView    treeViewModel
	treeBuilt   bool
	statusMsg   string
}

func newFindingsTab(scanID, url string, database *sql.DB) *findingsTab {
	fi := textinput.New()
	fi.Placeholder = "filter by URL…"
	fi.CharLimit = 256

	t := table.New(table.WithFocused(true), table.WithColumns(findingsColumns(80)))

	p := paginator.New(paginator.WithPerPage(findingsPageSize))
	p.Type = paginator.Arabic
	p.KeyMap.PrevPage = key.NewBinding(key.WithKeys("left", "["))
	p.KeyMap.NextPage = key.NewBinding(key.WithKeys("right", "]"))

	return &findingsTab{
		scanID:      scanID,
		url:         url,
		fs:          store.NewFindingsStore(database),
		filter:      store.FindingsFilter{ScanID: scanID, Page: 1, PageSize: findingsPageSize, SortBy: "url", SortOrder: "asc"},
		table:       t,
		pager:       p,
		loading:     true,
		filterInput: fi,
	}
}

func (f *findingsTab) Title() string {
	short := f.url
	if len(short) > 22 {
		short = "…" + short[len(short)-21:]
	}
	return "findings: " + short
}

func (f *findingsTab) Closeable() bool { return true }

func (f *findingsTab) Init() tea.Cmd {
	return f.loadFindings()
}

func (f *findingsTab) Update(msg tea.Msg) (tabView, tea.Cmd) {
	switch msg := msg.(type) {
	case findingsLoadedMsg:
		f.loading = false
		if msg.err != nil {
			f.err = msg.err
			return f, nil
		}
		f.result = msg.result
		f.pager.SetTotalPages(msg.result.Total)
		f.table.SetRows(f.findingsToRows(msg.result.Findings))
		return f, nil

	case exportDoneMsg:
		if msg.err != nil {
			f.statusMsg = "Export failed: " + msg.err.Error()
		} else {
			f.statusMsg = "Exported to " + msg.path
		}
		return f, nil

	case treeDataMsg:
		f.loading = false
		if msg.err != nil {
			f.err = msg.err
			return f, nil
		}
		treeFindings := make([]tree.Finding, len(msg.findings))
		for i, sf := range msg.findings {
			treeFindings[i] = tree.Finding{
				URL:           sf.URL,
				ContentType:   sf.ContentType,
				ContentLength: sf.ContentLength,
				Category:      sf.Category,
				Interest:      sf.InterestScore,
				LastModified:  sf.LastModified,
			}
		}
		root := tree.Build(f.url, treeFindings)
		f.treeView = newTreeViewModel(root)
		f.treeBuilt = true
		return f, nil

	case tea.KeyPressMsg:
		// Filter bar intercepts keys when open
		if f.filterOpen {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				f.filterOpen = false
				f.filterInput.Reset()
				q := ""
				f.filter.Query = &q
				f.filter.Page = 1
				f.pager.Page = 0
				return f, f.loadFindings()
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				q := strings.TrimSpace(f.filterInput.Value())
				if q == "" {
					f.filter.Query = nil
				} else {
					f.filter.Query = &q
				}
				f.filter.Page = 1
				f.pager.Page = 0
				return f, f.loadFindings()
			default:
				var cmd tea.Cmd
				f.filterInput, cmd = f.filterInput.Update(msg)
				return f, cmd
			}
		}

		// Tree view delegates all keys except back/toggle
		if f.showTree && f.treeBuilt {
			switch {
			case key.Matches(msg, findingsKeys.Back):
				return f, func() tea.Msg { return closeTabMsg{} }
			case key.Matches(msg, findingsKeys.ToggleTree):
				f.showTree = false
				return f, nil
			case key.Matches(msg, findingsKeys.Export):
				f.statusMsg = "Exporting..."
				return f, f.exportHTML()
			}
			var cmd tea.Cmd
			f.treeView, cmd = f.treeView.update(msg)
			return f, cmd
		}

		switch {
		case key.Matches(msg, findingsKeys.Back):
			return f, func() tea.Msg { return closeTabMsg{} }

		case key.Matches(msg, findingsKeys.ToggleTree):
			f.showTree = true
			if !f.treeBuilt {
				f.loading = true
				return f, f.loadAllFindings()
			}
			return f, nil

		case key.Matches(msg, findingsKeys.Filter):
			f.filterOpen = true
			return f, f.filterInput.Focus()

		case key.Matches(msg, findingsKeys.SortInterest):
			if f.filter.SortBy == "interest_score" && f.filter.SortOrder == "desc" {
				f.filter.SortBy = "url"
				f.filter.SortOrder = "asc"
			} else {
				f.filter.SortBy = "interest_score"
				f.filter.SortOrder = "desc"
			}
			f.filter.Page = 1
			f.pager.Page = 0
			return f, f.loadFindings()

		case key.Matches(msg, findingsKeys.Export):
			f.statusMsg = "Exporting..."
			return f, f.exportHTML()
		}

		// Let paginator handle page keys ([, ], pgup, pgdn)
		prevPage := f.pager.Page
		f.pager, _ = f.pager.Update(msg)
		if f.pager.Page != prevPage {
			f.filter.Page = f.pager.Page + 1 // paginator is 0-indexed, filter is 1-indexed
			return f, f.loadFindings()
		}

		var cmd tea.Cmd
		f.table, cmd = f.table.Update(msg)
		return f, cmd

	case tea.MouseWheelMsg:
		if f.showTree && f.treeBuilt {
			f.treeView, _ = f.treeView.update(msg)
			return f, nil
		}
		if msg.Button == tea.MouseWheelUp {
			f.table.MoveUp(3)
		} else if msg.Button == tea.MouseWheelDown {
			f.table.MoveDown(3)
		}
		return f, nil

	case tea.MouseClickMsg:
		if f.showTree && f.treeBuilt {
			f.treeView, _ = f.treeView.update(msg)
			return f, nil
		}
		if msg.Button == tea.MouseLeft {
			col := f.columnAtX(msg.X)
			if col >= 0 && msg.Y == 0 {
				return f, f.cycleSortByColumn(col)
			}
		}
	}

	var cmd tea.Cmd
	f.table, cmd = f.table.Update(msg)
	return f, cmd
}

// columnAtX returns the column index at the given X position, or -1 if out of range.
// Accounts for cell padding (1 char each side) added by the table styles.
func (f *findingsTab) columnAtX(x int) int {
	const cellPad = 2 // Padding(0, 1) = 1 left + 1 right
	offset := 0
	for i, w := range f.colWidths {
		colW := w + cellPad
		if x >= offset && x < offset+colW {
			return i
		}
		offset += colW
	}
	return -1
}

// cycleSortByColumn cycles the sort state for the given column: asc → desc → default.
func (f *findingsTab) cycleSortByColumn(col int) tea.Cmd {
	if col < 0 || col >= len(findingsSortCol) {
		return nil
	}
	sortField := findingsSortCol[col]

	if f.filter.SortBy == sortField {
		if f.filter.SortOrder == "asc" {
			f.filter.SortOrder = "desc"
		} else {
			// Reset to default
			f.filter.SortBy = "url"
			f.filter.SortOrder = "asc"
		}
	} else {
		f.filter.SortBy = sortField
		f.filter.SortOrder = "asc"
	}
	f.filter.Page = 1
	return f.loadFindings()
}

func (f *findingsTab) View(width, height int, dark bool) string {
	if f.err != nil {
		msg := lipgloss.NewStyle().Foreground(errorC(dark)).Render("Error: ") + f.err.Error()
		return styleContent.Render(msg)
	}
	if f.loading && len(f.result.Findings) == 0 {
		return styleContent.Render(styleMuted(dark) + " Loading…")
	}

	// Tree view mode
	if f.showTree && f.treeBuilt {
		statusH := 1
		treeH := max(height-statusH, 1)
		content := f.treeView.view(width, treeH, dark)

		sortInfo := fmt.Sprintf("tree  sort:%s  %d files  %s",
			f.treeView.sortLabel(),
			f.treeView.root.FileCount,
			humanizeBytes(f.treeView.root.Size),
		)
		if f.statusMsg != "" {
			sortInfo += "  " + f.statusMsg
		}
		status := lipgloss.NewStyle().Foreground(muted(dark)).Render(sortInfo)

		return content + "\n" + status
	}

	// Table view mode
	filterH := 0
	if f.filterOpen {
		filterH = 1
	}
	pageLineH := 1
	tableH := max(height-filterH-pageLineH, 1)

	cols := findingsColumns(width)
	f.colWidths = make([]int, len(cols))
	for i, c := range cols {
		f.colWidths[i] = c.Width
	}
	// Add sort indicator to the active column header
	for i, c := range cols {
		if i < len(findingsSortCol) && findingsSortCol[i] == f.filter.SortBy {
			if f.filter.SortOrder == "asc" {
				cols[i].Title = c.Title + " ^"
			} else {
				cols[i].Title = c.Title + " v"
			}
		}
	}
	f.table.SetColumns(cols)
	f.table.SetHeight(tableH)
	f.table.SetWidth(width)

	ts := table.DefaultStyles()
	ts.Selected = lipgloss.NewStyle().Bold(true).Foreground(accent(dark))
	ts.Header = lipgloss.NewStyle().Bold(true).Foreground(muted(dark)).Padding(0, 1)
	f.table.SetStyles(ts)

	parts := []string{f.table.View()}

	// Filter bar
	if f.filterOpen {
		bar := lipgloss.NewStyle().Foreground(accent(dark)).Render("/") + " " + f.filterInput.View()
		parts = append(parts, bar)
	}

	// Pagination + status
	pagerView := f.pager.View()
	totalInfo := fmt.Sprintf("  %d items", f.result.Total)
	if f.statusMsg != "" {
		totalInfo += "  " + f.statusMsg
	}
	parts = append(parts, pagerView+lipgloss.NewStyle().Foreground(muted(dark)).Render(totalInfo))

	return strings.Join(parts, "\n")
}

func (f *findingsTab) loadFindings() tea.Cmd {
	fs := f.fs
	filter := f.filter
	return func() tea.Msg {
		result, err := fs.QueryFindings(context.Background(), filter)
		return findingsLoadedMsg{result: result, err: err}
	}
}

func (f *findingsTab) loadAllFindings() tea.Cmd {
	fs := f.fs
	scanID := f.scanID
	return func() tea.Msg {
		// Load all findings (no pagination) for tree building
		result, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
			ScanID:   scanID,
			Page:     1,
			PageSize: 100000,
			SortBy:   "url",
		})
		if err != nil {
			return treeDataMsg{err: err}
		}
		return treeDataMsg{findings: result.Findings}
	}
}

func (f *findingsTab) exportHTML() tea.Cmd {
	fs := f.fs
	scanID := f.scanID
	scanURL := f.url
	return func() tea.Msg {
		result, err := fs.QueryFindings(context.Background(), store.FindingsFilter{
			ScanID: scanID, Page: 1, PageSize: 1000000, SortBy: "url",
		})
		if err != nil {
			return exportDoneMsg{err: err}
		}

		treeFindings := make([]tree.Finding, len(result.Findings))
		exportFindings := make([]export.Finding, len(result.Findings))
		for i, sf := range result.Findings {
			treeFindings[i] = tree.Finding{
				URL: sf.URL, ContentType: sf.ContentType, ContentLength: sf.ContentLength,
				Category: sf.Category, Interest: sf.InterestScore, LastModified: sf.LastModified,
			}
			exportFindings[i] = export.Finding{
				URL: sf.URL, ContentType: sf.ContentType, ContentLength: sf.ContentLength,
				Category: sf.Category, InterestScore: sf.InterestScore, Tags: sf.Tags, LastModified: sf.LastModified,
			}
		}

		root := tree.Build(scanURL, treeFindings)
		data := export.ScanData{
			ScanID: scanID, URL: scanURL, ScanDate: time.Now(),
			Tree: root, Findings: exportFindings, Total: result.Total,
		}

		prefix := scanID
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
		outPath := fmt.Sprintf("pry-%s.html", prefix)

		file, err := os.Create(outPath)
		if err != nil {
			return exportDoneMsg{err: err}
		}
		defer file.Close()

		if err := (export.HTMLExporter{}).Export(file, data); err != nil {
			return exportDoneMsg{err: err}
		}
		return exportDoneMsg{path: outPath}
	}
}

func findingsColumns(width int) []table.Column {
	catW := 12
	sizeW := 9
	modW := 12
	scoreW := 7
	fixedW := catW + sizeW + modW + scoreW + 5
	urlW := max(width-fixedW, 20)
	return []table.Column{
		{Title: "URL", Width: urlW},
		{Title: "Cat", Width: catW},
		{Title: "Size", Width: sizeW},
		{Title: "Modified", Width: modW},
		{Title: "Int", Width: scoreW},
	}
}

func (ft *findingsTab) findingsToRows(findings []store.Finding) []table.Row {
	// Strip the common base URL prefix so only the relative path is shown.
	base := strings.TrimRight(ft.url, "/")

	rows := make([]table.Row, len(findings))
	for i, f := range findings {
		displayURL := f.URL
		if after, ok := strings.CutPrefix(displayURL, base); ok {
			displayURL = after
			if displayURL == "" {
				displayURL = "/"
			}
		}

		mod := "-"
		if f.LastModified != nil {
			mod = f.LastModified.Format("2006-01-02")
		}
		score := ""
		if f.InterestScore > 0 {
			score = fmt.Sprintf("%d", f.InterestScore)
		}
		rows[i] = table.Row{
			displayURL,
			f.Category,
			humanizeBytes(f.ContentLength),
			mod,
			score,
		}
	}
	return rows
}

func humanizeBytes(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%dB", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1fK", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.1fM", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%.1fG", float64(n)/(1024*1024*1024))
	}
}
