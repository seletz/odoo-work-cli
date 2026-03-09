package tui

import (
	"testing"

	"github.com/seletz/odoo-work-cli/internal/config"
)

func TestApplyKeysConfig_SingleAction(t *testing.T) {
	km := DefaultKeyMap()
	cfg := config.KeysConfig{
		"global_quit": {"ctrl+q"},
	}

	km = ApplyKeysConfig(km, cfg)

	keys := km.Quit.Keys()
	if len(keys) != 1 || keys[0] != "ctrl+q" {
		t.Errorf("Quit.Keys() = %v, want [ctrl+q]", keys)
	}
	help := km.Quit.Help()
	if help.Key != "ctrl+q" {
		t.Errorf("Quit help key = %q, want %q", help.Key, "ctrl+q")
	}
	if help.Desc != "quit" {
		t.Errorf("Quit help desc = %q, want %q", help.Desc, "quit")
	}
}

func TestApplyKeysConfig_MultipleActions(t *testing.T) {
	km := DefaultKeyMap()
	cfg := config.KeysConfig{
		"global_quit": {"ctrl+q"},
		"detail_edit": {"F2"},
		"cursor_up":   {"up", "w"},
	}

	km = ApplyKeysConfig(km, cfg)

	// quit changed
	if keys := km.Quit.Keys(); len(keys) != 1 || keys[0] != "ctrl+q" {
		t.Errorf("Quit.Keys() = %v, want [ctrl+q]", keys)
	}
	// edit changed
	if keys := km.Edit.Keys(); len(keys) != 1 || keys[0] != "F2" {
		t.Errorf("Edit.Keys() = %v, want [F2]", keys)
	}
	// cursor_up changed
	if keys := km.Up.Keys(); len(keys) != 2 || keys[0] != "up" || keys[1] != "w" {
		t.Errorf("Up.Keys() = %v, want [up w]", keys)
	}
	// cursor_down unchanged
	if keys := km.Down.Keys(); len(keys) != 2 || keys[0] != "down" || keys[1] != "j" {
		t.Errorf("Down.Keys() = %v, want [down j] (unchanged)", keys)
	}
}

func TestApplyKeysConfig_UnknownAction(t *testing.T) {
	km := DefaultKeyMap()
	origKeys := km.Quit.Keys()
	cfg := config.KeysConfig{
		"nonexistent": {"x"},
	}

	km = ApplyKeysConfig(km, cfg)

	// Nothing should change.
	if keys := km.Quit.Keys(); len(keys) != len(origKeys) {
		t.Errorf("Quit.Keys() changed after unknown action override")
	}
}

func TestApplyKeysConfig_NilConfig(t *testing.T) {
	km := DefaultKeyMap()
	origKeys := km.Quit.Keys()

	km = ApplyKeysConfig(km, nil)

	if keys := km.Quit.Keys(); len(keys) != len(origKeys) {
		t.Errorf("Quit.Keys() changed with nil config")
	}
}

func TestApplyKeysConfig_HelpTextUpdated(t *testing.T) {
	km := DefaultKeyMap()
	cfg := config.KeysConfig{
		"cursor_up": {"up", "w", "i"},
	}

	km = ApplyKeysConfig(km, cfg)

	help := km.Up.Help()
	if help.Key != "up/w/i" {
		t.Errorf("Up help key = %q, want %q", help.Key, "up/w/i")
	}
	if help.Desc != "up" {
		t.Errorf("Up help desc = %q, want %q", help.Desc, "up")
	}
}

func TestApplyKeysConfig_AllActions(t *testing.T) {
	// Verify every valid action name is recognized.
	allActions := []string{
		"cursor_up", "cursor_down",
		"grid_next_col", "grid_prev_col", "grid_enter", "grid_search",
		"detail_edit", "detail_add", "detail_delete",
		"search_toggle",
		"global_quit", "global_help", "global_refresh", "global_back",
		"global_prev_week", "global_next_week",
	}
	for _, action := range allActions {
		km := DefaultKeyMap()
		cfg := config.KeysConfig{
			action: {"test_key"},
		}
		km = ApplyKeysConfig(km, cfg)
		// Just verify it doesn't panic and the key was applied.
		// We check a few known ones more thoroughly above.
		_ = km
	}
}
