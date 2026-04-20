package tui

import (
	"net/url"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type inputModel struct {
	ti  textinput.Model
	err string
}

func newInputModel() inputModel {
	ti := textinput.New()
	ti.Placeholder = "https://example.com/files/"
	ti.CharLimit = 2048
	return inputModel{ti: ti}
}

func (m *inputModel) reset() {
	m.ti.Reset()
	m.err = ""
}

// update handles key events when the input modal is open.
// Returns the submitted URL (non-empty on enter with a valid URL).
func (m inputModel) update(msg tea.Msg) (inputModel, tea.Cmd, string) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
			v := strings.TrimSpace(m.ti.Value())
			if v == "" {
				return m, nil, ""
			}
			if _, err := url.ParseRequestURI(v); err != nil {
				m.err = "Invalid URL"
				return m, nil, ""
			}
			m.err = ""
			m.ti.Reset()
			return m, nil, v
		}
	}
	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd, ""
}

func (m inputModel) view(width, height int, dark bool) string {
	title := lipgloss.NewStyle().Bold(true).Render("New Scan")

	errLine := ""
	if m.err != "" {
		errLine = "\n" + lipgloss.NewStyle().Foreground(errorC(dark)).Render(m.err)
	}

	hint := lipgloss.NewStyle().Foreground(muted(dark)).Render("enter to start  esc to cancel")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		"URL:",
		m.ti.View(),
		errLine,
		"",
		hint,
	)

	boxW := min(width-4, 60)
	box := modalStyle(dark).Width(boxW).Render(inner)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
