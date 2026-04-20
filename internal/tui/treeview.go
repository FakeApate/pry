package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fakeapate/pry/internal/tree"
)

// treeViewModel renders a collapsible directory tree using a viewport for scrolling.
type treeViewModel struct {
	root     *tree.Node
	expanded map[string]bool // keyed by node.Path
	rows     []treeRow       // flattened visible rows
	cursor   int
	width    int
	dark     bool
	sortMode tree.SortMode
	vp       viewport.Model
}

type treeRow struct {
	node  *tree.Node
	depth int
}

func newTreeViewModel(root *tree.Node) treeViewModel {
	vp := viewport.New()
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3
	m := treeViewModel{
		root:     root,
		expanded: map[string]bool{"/": true},
		sortMode: tree.SortByName,
		vp:       vp,
	}
	m.rebuild()
	return m
}

// rebuild flattens the tree into visible rows based on expanded state.
func (m *treeViewModel) rebuild() {
	m.rows = m.rows[:0]
	m.flatten(m.root, 0)
	if m.cursor >= len(m.rows) {
		m.cursor = max(len(m.rows)-1, 0)
	}
}

func (m *treeViewModel) flatten(n *tree.Node, depth int) {
	if n == m.root {
		if !m.expanded[n.Path] {
			return
		}
		for _, c := range n.Children {
			m.flatten(c, depth)
		}
		return
	}

	m.rows = append(m.rows, treeRow{node: n, depth: depth})
	if n.IsDir && m.expanded[n.Path] {
		for _, c := range n.Children {
			m.flatten(c, depth+1)
		}
	}
}

func (m *treeViewModel) toggle() {
	if m.cursor >= len(m.rows) {
		return
	}
	n := m.rows[m.cursor].node
	if !n.IsDir {
		return
	}
	m.expanded[n.Path] = !m.expanded[n.Path]
	m.rebuild()
}

func (m *treeViewModel) expandAll() {
	m.walkExpand(m.root, true)
	m.rebuild()
}

func (m *treeViewModel) collapseAll() {
	m.walkExpand(m.root, false)
	m.expanded["/"] = true
	m.rebuild()
}

func (m *treeViewModel) walkExpand(n *tree.Node, expand bool) {
	if n.IsDir {
		m.expanded[n.Path] = expand
		for _, c := range n.Children {
			m.walkExpand(c, expand)
		}
	}
}

func (m *treeViewModel) moveUp(n int) {
	m.cursor -= n
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *treeViewModel) moveDown(n int) {
	m.cursor += n
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *treeViewModel) goToParent() {
	if m.cursor >= len(m.rows) {
		return
	}
	row := m.rows[m.cursor]
	n := row.node

	if n.IsDir && m.expanded[n.Path] {
		m.expanded[n.Path] = false
		m.rebuild()
		return
	}

	if row.depth > 0 {
		for i := m.cursor - 1; i >= 0; i-- {
			if m.rows[i].depth < row.depth && m.rows[i].node.IsDir {
				m.cursor = i
				return
			}
		}
	}
}

func (m *treeViewModel) cycleSort() {
	m.sortMode = (m.sortMode + 1) % 3
	tree.Resort(m.root, m.sortMode)
	m.rebuild()
}

// renderContent builds the full tree as styled text and sets it on the viewport.
func (m *treeViewModel) renderContent() {
	var sb strings.Builder
	for i, row := range m.rows {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(m.renderRow(row, i == m.cursor))
	}
	m.vp.SetContent(sb.String())

	// Ensure cursor row is visible in the viewport
	vpHeight := m.vp.VisibleLineCount()
	if vpHeight <= 0 {
		return
	}
	yOff := m.vp.YOffset()
	if m.cursor < yOff {
		m.vp.SetYOffset(m.cursor)
	} else if m.cursor >= yOff+vpHeight {
		m.vp.SetYOffset(m.cursor - vpHeight + 1)
	}
}

func (m treeViewModel) update(msg tea.Msg) (treeViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			m.moveUp(1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			m.moveDown(1)
		case key.Matches(msg, key.NewBinding(key.WithKeys(" "))):
			m.toggle()
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			if m.cursor < len(m.rows) {
				n := m.rows[m.cursor].node
				if n.IsDir && !m.expanded[n.Path] {
					m.expanded[n.Path] = true
					m.rebuild()
				}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			m.goToParent()
		case key.Matches(msg, key.NewBinding(key.WithKeys("e"))):
			m.expandAll()
		case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
			m.collapseAll()
		case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
			m.cycleSort()
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
			m.moveUp(m.vp.VisibleLineCount())
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
			m.moveDown(m.vp.VisibleLineCount())
		case key.Matches(msg, key.NewBinding(key.WithKeys("home"))):
			m.cursor = 0
		case key.Matches(msg, key.NewBinding(key.WithKeys("end"))):
			m.cursor = len(m.rows) - 1
		default:
			// Let viewport handle any remaining keys
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			row := m.vp.YOffset() + msg.Y
			if row >= 0 && row < len(m.rows) {
				if row == m.cursor && m.rows[row].node.IsDir {
					m.toggle()
				} else {
					m.cursor = row
				}
			}
		}

	default:
		// Pass mouse wheel and other events to viewport
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		// Sync cursor to viewport scroll position if viewport scrolled
		return m, cmd
	}

	return m, nil
}

func (m treeViewModel) view(width, height int, dark bool) string {
	m.width = width
	m.dark = dark

	if len(m.rows) == 0 {
		return lipgloss.NewStyle().Foreground(muted(dark)).Render("  (empty)")
	}

	m.vp.SetWidth(width)
	m.vp.SetHeight(height)
	m.renderContent()

	return m.vp.View()
}

func (m treeViewModel) renderRow(row treeRow, selected bool) string {
	n := row.node
	indent := strings.Repeat("  ", row.depth)

	var icon string
	if n.IsDir {
		if m.expanded[n.Path] {
			icon = "v "
		} else {
			icon = "> "
		}
	} else {
		icon = "  "
	}

	var meta string
	if n.IsDir {
		meta = fmt.Sprintf("%d files  %s", n.FileCount, humanizeBytes(n.Size))
	} else {
		parts := []string{}
		if n.Category != "" && n.Category != "other" {
			parts = append(parts, n.Category)
		}
		parts = append(parts, humanizeBytes(n.Size))
		if n.LastModified != nil {
			parts = append(parts, n.LastModified.Format("2006-01-02"))
		}
		if n.Interest > 50 {
			parts = append(parts, fmt.Sprintf("[%d]", n.Interest))
		}
		meta = strings.Join(parts, "  ")
	}

	nameStr := n.Name
	if n.IsDir {
		nameStr += "/"
	}

	// Truncate name if needed to leave room for meta
	gap := 2
	maxName := m.width - len(indent) - len(icon) - gap - len(meta)
	if maxName < 10 {
		maxName = 10
	}
	if len(nameStr) > maxName {
		nameStr = nameStr[:maxName-1] + "~"
	}

	// Build the line with right-aligned meta
	left := indent + icon + nameStr
	padding := m.width - len(left) - len(meta)
	if padding < 1 {
		padding = 1
	}

	line := left + strings.Repeat(" ", padding) + meta

	if len(line) > m.width {
		line = line[:m.width]
	}

	// Style
	style := lipgloss.NewStyle()
	if selected {
		style = style.Bold(true).Foreground(accent(m.dark))
	} else if n.IsDir {
		style = style.Bold(true)
	} else if n.Interest > 80 {
		style = style.Foreground(errorC(m.dark))
	} else if n.Interest > 50 {
		style = style.Foreground(warning(m.dark))
	}

	return style.Render(line)
}

func (m treeViewModel) sortLabel() string {
	switch m.sortMode {
	case tree.SortBySize:
		return "size"
	case tree.SortByInterest:
		return "interest"
	default:
		return "name"
	}
}
