package tui

import "charm.land/bubbles/v2/key"

// KeyMap defines key bindings for the TUI.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	NextCol  key.Binding
	PrevCol  key.Binding
	Refresh  key.Binding
	Help     key.Binding
	Quit     key.Binding
	PrevWeek key.Binding
	NextWeek key.Binding
	Enter    key.Binding
	Back     key.Binding
	Edit     key.Binding
	Add      key.Binding
	Delete   key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "prev week"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next week"),
		),
		NextCol: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next day"),
		),
		PrevCol: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("s-tab", "prev day"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "detail"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
	}
}

// ShortHelp returns key bindings for the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.NextCol, k.Left, k.Right, k.Enter, k.Edit, k.Add, k.Delete, k.Refresh, k.Help, k.Quit}
}

// FullHelp returns key bindings for the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.NextCol, k.PrevCol},
		{k.Left, k.Right},
		{k.Enter, k.Back, k.Edit, k.Add, k.Delete},
		{k.Refresh, k.Help, k.Quit},
	}
}
