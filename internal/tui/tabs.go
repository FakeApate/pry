package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// tabView is implemented by each tab's model.
type tabView interface {
	Title() string
	Init() tea.Cmd
	Update(msg tea.Msg) (tabView, tea.Cmd)
	View(width, height int, dark bool) string
	// Closeable returns false for permanent tabs (e.g. the scan list).
	Closeable() bool
}

// renderTabBar renders the tab strip with a bottom border.
func renderTabBar(tabs []tabView, active int, width int, dark bool) string {
	var parts []string
	for i, t := range tabs {
		if i == active {
			parts = append(parts, activeTabStyle(dark).Render(t.Title()))
		} else {
			parts = append(parts, tabStyle(dark).Render(t.Title()))
		}
	}

	bar := strings.Join(parts, "")
	// Pad to full width so the bottom border spans the terminal.
	bar = tabBarStyle(dark).Width(width).Render(bar)
	return bar
}

// renderFooter renders a one-line key-hint bar.
func renderFooter(hints []string, width int, dark bool) string {
	line := strings.Join(hints, "  ")
	return statusBarStyle(dark).Width(width).Render(line)
}

// hint formats a single "key desc" pair for the footer.
func hint(k, desc string, dark bool) string {
	return keyNameStyle(dark).Render(k) + styleKeyHint.Render(" "+desc)
}
