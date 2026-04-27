package tui

import (
	"database/sql"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fakeapate/pry/internal/store/db"
	"github.com/fakeapate/pry/model"
)

// DispatchFunc creates a scan record and spawns a scanner goroutine. Non-blocking.
type DispatchFunc func(url string) (scanID string, err error)

type rootModel struct {
	tabs      []tabView
	activeTab int
	width     int
	height    int
	hasDarkBg bool
	database  *sql.DB
	queries   *db.Queries
	dispatch  DispatchFunc
	inputOpen bool
	inputView inputModel
	showHelp  bool
	flashMsg  string // transient error banner; cleared on next key press
}

func NewModel(database *sql.DB, dispatch DispatchFunc) rootModel {
	queries := db.New(database)
	return rootModel{
		tabs:      []tabView{newScansTab(queries)},
		queries:   queries,
		database:  database,
		dispatch:  dispatch,
		inputView: newInputModel(),
	}
}

func (m rootModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		func() tea.Msg { return tea.RequestBackgroundColor() },
	}
	for _, t := range m.tabs {
		if cmd := t.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.BackgroundColorMsg:
		m.hasDarkBg = msg.IsDark()
		return m, nil

	case closeTabMsg:
		if len(m.tabs) > 1 && m.tabs[m.activeTab].Closeable() {
			m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
			if m.activeTab >= len(m.tabs) {
				m.activeTab = len(m.tabs) - 1
			}
		}
		return m, nil

	// Orchestrator events — route to the matching active scan tab.

	case model.ScanProgressEvent:
		i := m.findActiveTab(msg.ScanID)
		if i < 0 {
			return m, nil
		}
		var cmd tea.Cmd
		m.tabs[i], cmd = m.tabs[i].Update(msg)
		return m, cmd

	case model.ScanDoneEvent:
		var cmds []tea.Cmd
		i := m.findActiveTab(msg.ScanID)
		if i >= 0 {
			var cmd tea.Cmd
			m.tabs[i], cmd = m.tabs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		// Refresh scans list
		cmds = append(cmds, func() tea.Msg { return refreshTickMsg{} })
		return m, tea.Batch(cmds...)

	case model.ScanFailedEvent:
		var cmds []tea.Cmd
		i := m.findActiveTab(msg.ScanID)
		if i >= 0 {
			var cmd tea.Cmd
			m.tabs[i], cmd = m.tabs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, func() tea.Msg { return refreshTickMsg{} })
		return m, tea.Batch(cmds...)

	// Always route to scans tab (index 0) regardless of which tab is active.
	case refreshTickMsg:
		var cmd tea.Cmd
		m.tabs[0], cmd = m.tabs[0].Update(msg)
		return m, cmd

	// Route elapsed ticks to the correct active scan tab.
	case elapsedTickMsg:
		i := m.findActiveTab(msg.scanID)
		if i < 0 {
			return m, nil
		}
		var cmd tea.Cmd
		m.tabs[i], cmd = m.tabs[i].Update(msg)
		return m, cmd

	// New scan dispatched: open an active scan tab.
	case scanDispatchedMsg:
		if msg.err != nil {
			m.flashMsg = "dispatch failed: " + msg.err.Error()
			return m, nil
		}
		at := newActiveModel(msg.scanID, msg.url)
		m.tabs = append(m.tabs, at)
		m.activeTab = len(m.tabs) - 1
		return m, at.Init()

	// Open the URL input modal.
	case openInputMsg:
		m.inputOpen = true
		// Focus directly on m so the pointer-receiver Focus() mutates our copy.
		cmd := m.inputView.ti.Focus()
		return m, cmd

	// Open findings tab for a completed scan.
	case openFindingsMsg:
		// Findings tab already open for this scan?
		existing := -1
		for i, t := range m.tabs {
			if ft, ok := t.(*findingsTab); ok && ft.scanID == msg.scanID {
				existing = i
				break
			}
		}

		// Is the caller the matching active scan tab? If so, replace it.
		replaceIdx := -1
		if at, ok := m.tabs[m.activeTab].(*activeModel); ok && at.scanID == msg.scanID {
			replaceIdx = m.activeTab
		}

		switch {
		case existing >= 0 && replaceIdx >= 0:
			// Close the scan tab and focus the existing findings tab.
			m.tabs = append(m.tabs[:replaceIdx], m.tabs[replaceIdx+1:]...)
			if existing > replaceIdx {
				existing--
			}
			m.activeTab = existing
			return m, nil
		case existing >= 0:
			m.activeTab = existing
			return m, nil
		case replaceIdx >= 0:
			ft := newFindingsTab(msg.scanID, msg.url, m.database)
			m.tabs[replaceIdx] = ft
			return m, ft.Init()
		default:
			ft := newFindingsTab(msg.scanID, msg.url, m.database)
			m.tabs = append(m.tabs, ft)
			m.activeTab = len(m.tabs) - 1
			return m, ft.Init()
		}

	case tea.KeyPressMsg:
		// Any key press dismisses the flash banner.
		m.flashMsg = ""
		// Input overlay consumes all key events when open.
		if m.inputOpen {
			if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
				m.inputOpen = false
				m.inputView.reset()
				return m, nil
			}
			var cmd tea.Cmd
			var submitted string
			m.inputView, cmd, submitted = m.inputView.update(msg)
			if submitted != "" {
				m.inputOpen = false
				return m, tea.Batch(cmd, m.dispatchURL(submitted))
			}
			return m, cmd
		}

		// Help overlay: any key dismisses it.
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// Global keys.
		switch {
		case key.Matches(msg, globalKeys.Quit):
			return m, tea.Quit
		case key.Matches(msg, globalKeys.Help):
			m.showHelp = true
			return m, nil
		case key.Matches(msg, globalKeys.TabNext):
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			return m, nil
		case key.Matches(msg, globalKeys.TabPrev):
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			return m, nil
		case key.Matches(msg, globalKeys.TabClose):
			if len(m.tabs) > 1 && m.tabs[m.activeTab].Closeable() {
				m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
				if m.activeTab >= len(m.tabs) {
					m.activeTab = len(m.tabs) - 1
				}
			}
			return m, nil
		}
	}

	// When the input modal is open, route remaining messages (e.g. cursor blink)
	// to the text input so the cursor animates.
	if m.inputOpen {
		var cmd tea.Cmd
		m.inputView.ti, cmd = m.inputView.ti.Update(msg)
		return m, cmd
	}

	// Handle mouse events: tab bar clicks and coordinate translation.
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		innerW, leftPad := m.contentLayout()
		tabBar := renderTabBar(m.tabs, m.activeTab, innerW, m.hasDarkBg)
		tabBarH := lipgloss.Height(tabBar)
		mouse := mouseMsg.Mouse()

		// Translate X to content-relative coordinates.
		mouse.X -= leftPad

		// Click in the tab bar area — left switches tabs, middle closes.
		if click, isClick := msg.(tea.MouseClickMsg); isClick && mouse.Y < tabBarH {
			if click.Button == tea.MouseLeft {
				if idx := m.tabAtX(mouse.X); idx >= 0 {
					m.activeTab = idx
				}
				return m, nil
			}
			if click.Button == tea.MouseMiddle {
				idx := m.tabAtX(mouse.X)
				if idx >= 0 && len(m.tabs) > 1 && m.tabs[idx].Closeable() {
					m.tabs = append(m.tabs[:idx], m.tabs[idx+1:]...)
					switch {
					case m.activeTab > idx:
						m.activeTab--
					case m.activeTab >= len(m.tabs):
						m.activeTab = len(m.tabs) - 1
					}
				}
				return m, nil
			}
		}

		// Translate Y to content-relative coordinates for the active tab.
		mouse.Y -= tabBarH
		var translated tea.Msg
		switch msg.(type) {
		case tea.MouseClickMsg:
			translated = tea.MouseClickMsg(mouse)
		case tea.MouseWheelMsg:
			translated = tea.MouseWheelMsg(mouse)
		case tea.MouseReleaseMsg:
			translated = tea.MouseReleaseMsg(mouse)
		case tea.MouseMotionMsg:
			translated = tea.MouseMotionMsg(mouse)
		default:
			translated = msg
		}
		var cmd tea.Cmd
		m.tabs[m.activeTab], cmd = m.tabs[m.activeTab].Update(translated)
		return m, cmd
	}

	// Delegate remaining messages to the active tab.
	var cmd tea.Cmd
	m.tabs[m.activeTab], cmd = m.tabs[m.activeTab].Update(msg)
	return m, cmd
}

// maxContentWidth caps the usable width on wide terminals so columns don't
// spread uncomfortably far apart.
const maxContentWidth = 140

// contentLayout returns the inner content width and the left padding needed
// to centre it within the terminal.
func (m rootModel) contentLayout() (innerW, leftPad int) {
	innerW = m.width
	if innerW > maxContentWidth {
		innerW = maxContentWidth
	}
	leftPad = (m.width - innerW) / 2
	return
}

func (m rootModel) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	innerW, leftPad := m.contentLayout()

	tabBar := renderTabBar(m.tabs, m.activeTab, innerW, m.hasDarkBg)
	tabBarH := lipgloss.Height(tabBar)

	footer := m.renderFooter(innerW)
	footerH := lipgloss.Height(footer)

	var flash string
	var flashH int
	if m.flashMsg != "" {
		flash = lipgloss.NewStyle().
			Foreground(errorC(m.hasDarkBg)).
			Width(innerW).
			Render(" " + m.flashMsg)
		flashH = lipgloss.Height(flash)
	}

	contentH := max(m.height-tabBarH-footerH-flashH, 1)
	content := m.tabs[m.activeTab].View(innerW, contentH, m.hasDarkBg)

	// Input overlay
	if m.inputOpen {
		content = m.inputView.view(innerW, contentH, m.hasDarkBg)
	}

	// Help overlay
	if m.showHelp {
		content = m.renderHelp(innerW, contentH)
	}

	var inner string
	if flash != "" {
		inner = lipgloss.JoinVertical(lipgloss.Left, tabBar, content, flash, footer)
	} else {
		inner = lipgloss.JoinVertical(lipgloss.Left, tabBar, content, footer)
	}
	if leftPad > 0 {
		inner = lipgloss.NewStyle().PaddingLeft(leftPad).Render(inner)
	}

	v := tea.NewView(inner)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m rootModel) renderFooter(width int) string {
	dark := m.hasDarkBg
	var hints []string

	switch m.tabs[m.activeTab].(type) {
	case *findingsTab:
		ft := m.tabs[m.activeTab].(*findingsTab)
		if ft.showTree {
			hints = []string{
				hint("esc", "back", dark),
				hint("v", "table view", dark),
				hint("space", "expand/collapse", dark),
				hint("e/c", "expand/collapse all", dark),
				hint("s", "sort", dark),
				hint("x", "export", dark),
			}
		} else {
			hints = []string{
				hint("esc", "back", dark),
				hint("v", "tree view", dark),
				hint("/", "filter", dark),
				hint("i", "sort interest", dark),
				hint("</>", "page", dark),
				hint("x", "export", dark),
			}
		}
	case *activeModel:
		hints = []string{
			hint("enter", "findings", dark),
			hint("ctrl+w", "close", dark),
			hint("q", "quit", dark),
		}
	default:
		hints = []string{
			hint("n", "new scan", dark),
			hint("enter", "findings", dark),
			hint("d", "delete", dark),
			hint("?", "help", dark),
			hint("q", "quit", dark),
		}
	}
	return renderFooter(hints, width, dark)
}

func (m rootModel) renderHelp(width, height int) string {
	dark := m.hasDarkBg
	lines := []string{
		lipgloss.NewStyle().Bold(true).Render("Key Bindings"),
		"",
		lipgloss.NewStyle().Foreground(accent(dark)).Bold(true).Render("Global"),
		hint("q/ctrl+c", "quit", dark),
		hint("tab/shift+tab", "switch tabs", dark),
		hint("ctrl+w", "close tab", dark),
		hint("middle-click", "close tab from bar", dark),
		hint("?", "toggle help", dark),
		"",
		lipgloss.NewStyle().Foreground(accent(dark)).Bold(true).Render("Scan List"),
		hint("n", "new scan", dark),
		hint("enter", "open findings", dark),
		hint("d", "delete scan", dark),
		hint("r", "refresh", dark),
		hint("s/u/f/t", "sort by status/url/findings/date", dark),
		"",
		lipgloss.NewStyle().Foreground(accent(dark)).Bold(true).Render("Findings"),
		hint("/", "filter", dark),
		hint("[/]  pgup/pgdn", "page", dark),
		hint("esc", "clear filter / back", dark),
		"",
		lipgloss.NewStyle().Foreground(muted(dark)).Render("press any key to close"),
	}
	content := strings.Join(lines, "\n")
	boxW := min(width-4, 56)
	box := modalStyle(dark).Width(boxW).Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (m rootModel) dispatchURL(u string) tea.Cmd {
	dispatch := m.dispatch
	return func() tea.Msg {
		scanID, err := dispatch(u)
		return scanDispatchedMsg{scanID: scanID, url: u, err: err}
	}
}

// tabAtX returns the tab index at the given X position in the tab bar, or -1.
// Tab titles use Padding(0, 2) so each tab occupies len(title) + 4 chars.
func (m rootModel) tabAtX(x int) int {
	const tabPad = 4 // Padding(0, 2) = 2 left + 2 right
	offset := 0
	for i, t := range m.tabs {
		w := len(t.Title()) + tabPad
		if x >= offset && x < offset+w {
			return i
		}
		offset += w
	}
	return -1
}

func (m rootModel) findActiveTab(scanID string) int {
	for i, t := range m.tabs {
		if at, ok := t.(*activeModel); ok && at.scanID == scanID {
			return i
		}
	}
	return -1
}
