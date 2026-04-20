package tui

import "charm.land/bubbles/v2/key"

type globalKeyMap struct {
	Quit     key.Binding
	Help     key.Binding
	TabNext  key.Binding
	TabPrev  key.Binding
	TabClose key.Binding
}

type scanListKeyMap struct {
	New          key.Binding
	Open         key.Binding
	Delete       key.Binding
	Refresh      key.Binding
	SortStatus   key.Binding
	SortURL      key.Binding
	SortFindings key.Binding
	SortDate     key.Binding
}

type findingsKeyMap struct {
	Back         key.Binding
	Filter       key.Binding
	CopyURL      key.Binding
	Open         key.Binding
	SortInterest key.Binding
	ToggleTree   key.Binding
	Export       key.Binding
}

type activeKeyMap struct {
	ScrollUp   key.Binding
	ScrollDown key.Binding
	ViewFindings key.Binding
}

var globalKeys = globalKeyMap{
	Quit:     key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("q", "quit")),
	Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	TabNext:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
	TabPrev:  key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev tab")),
	TabClose: key.NewBinding(key.WithKeys("ctrl+w"), key.WithHelp("ctrl+w", "close tab")),
}

var scanListKeys = scanListKeyMap{
	New:          key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new scan")),
	Open:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open findings")),
	Delete:       key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Refresh:      key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	SortStatus:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort status")),
	SortURL:      key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "sort url")),
	SortFindings: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "sort findings")),
	SortDate:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "sort date")),
}

var findingsKeys = findingsKeyMap{
	Back:         key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Filter:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	CopyURL:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy url")),
	Open:         key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in browser")),
	SortInterest: key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "sort by interest")),
	ToggleTree:   key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "tree/table view")),
	Export:       key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "export")),
}

var activeKeys = activeKeyMap{
	ScrollUp:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "scroll up")),
	ScrollDown:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "scroll down")),
	ViewFindings: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "view findings")),
}
