package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"github.com/seletz/odoo-work-cli/internal/config"
)

// KeyMap defines key bindings for the TUI.
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	NextCol      key.Binding
	PrevCol      key.Binding
	Refresh      key.Binding
	Help         key.Binding
	Quit         key.Binding
	Enter        key.Binding
	Back         key.Binding
	Edit         key.Binding
	Add          key.Binding
	Delete       key.Binding
	Search       key.Binding
	SearchToggle key.Binding
	ClockToggle  key.Binding
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
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		SearchToggle: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("C-a", "toggle filter"),
		),
		ClockToggle: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clock in/out"),
		),
	}
}

// actionHelpDesc maps config action names to their help description text.
// Action names are prefixed with the context they apply to:
//   - cursor_  : shared cursor movement (grid, detail, search)
//   - grid_    : grid view actions
//   - detail_  : detail view actions
//   - search_  : search view actions
//   - global_  : actions available in all non-modal views
var actionHelpDesc = map[string]string{
	"cursor_up":           "up",
	"cursor_down":         "down",
	"grid_next_col":       "next day",
	"grid_prev_col":       "prev day",
	"grid_enter":          "detail",
	"grid_search":         "search",
	"detail_edit":         "edit",
	"detail_add":          "add",
	"detail_delete":       "delete",
	"search_toggle":       "toggle filter",
	"global_quit":         "quit",
	"global_help":         "help",
	"global_refresh":      "refresh",
	"global_back":         "back",
	"global_prev_week":    "prev week",
	"global_next_week":    "next week",
	"global_clock_toggle": "clock in/out",
}

// ApplyKeysConfig overrides key bindings in km from the given config.
// Unknown action names are silently ignored. Returns the modified KeyMap.
func ApplyKeysConfig(km KeyMap, cfg config.KeysConfig) KeyMap {
	if cfg == nil {
		return km
	}
	for action, keys := range cfg {
		desc, ok := actionHelpDesc[action]
		if !ok {
			continue
		}
		binding := key.NewBinding(
			key.WithKeys(keys...),
			key.WithHelp(strings.Join(keys, "/"), desc),
		)
		switch action {
		case "cursor_up":
			km.Up = binding
		case "cursor_down":
			km.Down = binding
		case "grid_next_col":
			km.NextCol = binding
		case "grid_prev_col":
			km.PrevCol = binding
		case "grid_enter":
			km.Enter = binding
		case "grid_search":
			km.Search = binding
		case "detail_edit":
			km.Edit = binding
		case "detail_add":
			km.Add = binding
		case "detail_delete":
			km.Delete = binding
		case "search_toggle":
			km.SearchToggle = binding
		case "global_quit":
			km.Quit = binding
		case "global_help":
			km.Help = binding
		case "global_refresh":
			km.Refresh = binding
		case "global_back":
			km.Back = binding
		case "global_prev_week":
			km.Left = binding
		case "global_next_week":
			km.Right = binding
		case "global_clock_toggle":
			km.ClockToggle = binding
		}
	}
	return km
}

// ShortHelp returns key bindings for the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.NextCol, k.Left, k.Right, k.Enter, k.Edit, k.Add, k.Delete, k.Search, k.ClockToggle, k.Refresh, k.Help, k.Quit}
}

// FullHelp returns key bindings for the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.NextCol, k.PrevCol},
		{k.Left, k.Right},
		{k.Enter, k.Back, k.Edit, k.Add, k.Delete, k.Search},
		{k.ClockToggle, k.Refresh, k.Help, k.Quit},
	}
}
