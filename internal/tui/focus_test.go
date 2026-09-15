package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// Issue #41: default key bindings (j/k, h/l, q, ...) must never intercept
// typing in a focused text input. The search view uses a Tab focus model:
// the search field has focus first; Tab moves focus to the results list,
// where cursor_up/cursor_down (incl. j/k) navigate.

// typed builds a key press the way a terminal delivers a printable character:
// Code and Text both set.
func typed(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// searchLoadedModel returns a model in the search state with two projects loaded.
func searchLoadedModel(t *testing.T, keys config.KeysConfig) Model {
	t.Helper()
	client := &mockClient{
		entries: []odoo.TimesheetEntry{
			{ID: 1, Date: "2026-03-02", Project: "Acme", Task: "Dev", Company: "Acme Org", Hours: 2.0, ProjectID: 10, TaskID: 20},
		},
	}
	mon := MondayTime{Time: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)}
	m := NewModel(client, mon, config.DefaultHoursLimits(), "Deutschland", keys, nil)
	m.state = stateGrid
	m.grid = BuildWeekGrid(client.entries, mon.Time)
	m.width = 120
	m.height = 40

	updated, _ := m.Update(tea.KeyPressMsg{Code: '/'})
	um := updated.(Model)
	if um.state != stateSearch {
		t.Fatalf("expected stateSearch, got %d", um.state)
	}
	updated, _ = um.Update(searchDataLoadedMsg{
		projects: []odoo.ProjectInfo{
			{ID: 1, Name: "Alpha"},
			{ID: 2, Name: "Beta"},
		},
	})
	return updated.(Model)
}

func TestSearch_TypingIsNotInterceptedByDefaultBindings(t *testing.T) {
	// Every printable key that is bound to an action by default.
	for _, r := range "jkhlnpqrcaed?/" {
		t.Run(string(r), func(t *testing.T) {
			m := searchLoadedModel(t, nil)
			updated, _ := m.Update(typed(r))
			um := updated.(Model)

			if um.state != stateSearch {
				t.Fatalf("state = %d, want stateSearch", um.state)
			}
			if got := um.searchInput.Value(); got != string(r) {
				t.Fatalf("search input = %q, want %q", got, string(r))
			}
			if um.searchCursor != 0 {
				t.Fatalf("searchCursor = %d, want 0 (typing must not navigate)", um.searchCursor)
			}
			if !um.searchUseFilter {
				t.Fatal("typing must not toggle the filter")
			}
		})
	}
}

func TestSearch_TypingIsNotInterceptedByLetterBoundConfig(t *testing.T) {
	// A user may bind any action to a letter; while the field has focus the
	// letter is still typed.
	keys := config.KeysConfig{
		"global_back":   {"x"},
		"search_toggle": {"t"},
		"cursor_down":   {"down", "n"},
	}
	m := searchLoadedModel(t, keys)
	for _, r := range "xtn" {
		updated, _ := m.Update(typed(r))
		m = updated.(Model)
	}
	if m.state != stateSearch {
		t.Fatalf("state = %d, want stateSearch", m.state)
	}
	if got := m.searchInput.Value(); got != "xtn" {
		t.Fatalf("search input = %q, want %q", got, "xtn")
	}
	if !m.searchUseFilter || m.searchCursor != 0 {
		t.Fatal("letter-bound actions must not fire while the search field has focus")
	}
}

func TestSearch_TabTogglesFocusBetweenFieldAndList(t *testing.T) {
	m := searchLoadedModel(t, nil)
	if m.searchFocus != focusInput {
		t.Fatalf("initial focus = %d, want focusInput", m.searchFocus)
	}
	if !m.searchInput.Focused() {
		t.Fatal("search input should be focused initially")
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)
	if m.searchFocus != focusList {
		t.Fatalf("focus after tab = %d, want focusList", m.searchFocus)
	}
	if m.searchInput.Focused() {
		t.Fatal("search input should be blurred while the list has focus")
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	m = updated.(Model)
	if m.searchFocus != focusInput {
		t.Fatalf("focus after shift+tab = %d, want focusInput", m.searchFocus)
	}
	if !m.searchInput.Focused() {
		t.Fatal("search input should be focused again")
	}
}

func TestSearch_ListFocusNavigatesWithCursorBindings(t *testing.T) {
	m := searchLoadedModel(t, nil)
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)

	updated, _ = m.Update(typed('j'))
	m = updated.(Model)
	if m.searchCursor != 1 {
		t.Fatalf("cursor after j = %d, want 1", m.searchCursor)
	}
	updated, _ = m.Update(typed('j'))
	m = updated.(Model)
	if m.searchCursor != 1 {
		t.Fatalf("cursor after j at bottom = %d, want 1 (clamped)", m.searchCursor)
	}
	updated, _ = m.Update(typed('k'))
	m = updated.(Model)
	if m.searchCursor != 0 {
		t.Fatalf("cursor after k = %d, want 0", m.searchCursor)
	}
	if got := m.searchInput.Value(); got != "" {
		t.Fatalf("search input = %q, want empty (list focus must not type)", got)
	}
}

func TestSearch_ArrowKeysNavigateInBothFocusStates(t *testing.T) {
	tests := []struct {
		name  string
		toTab bool
	}{
		{"field focus", false},
		{"list focus", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := searchLoadedModel(t, nil)
			if tt.toTab {
				updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
				m = updated.(Model)
			}
			updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			m = updated.(Model)
			if m.searchCursor != 1 {
				t.Fatalf("cursor after down = %d, want 1", m.searchCursor)
			}
			updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
			m = updated.(Model)
			if m.searchCursor != 0 {
				t.Fatalf("cursor after up = %d, want 0", m.searchCursor)
			}
		})
	}
}

func TestSearch_EnterAndEscWorkInBothFocusStates(t *testing.T) {
	for _, toTab := range []bool{false, true} {
		m := searchLoadedModel(t, nil)
		if toTab {
			updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
			m = updated.(Model)
		}
		updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		if got := updated.(Model).state; got != stateGrid {
			t.Fatalf("toTab=%v: state after esc = %d, want stateGrid", toTab, got)
		}

		updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = updated.(Model)
		updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = updated.(Model)
		if m.state != stateGrid {
			t.Fatalf("toTab=%v: state after enter = %d, want stateGrid", toTab, m.state)
		}
		if row := m.grid.Rows[m.cursor[0]]; row.HintProjectID != 2 {
			t.Fatalf("toTab=%v: selected row project = %d, want 2 (Beta)", toTab, row.HintProjectID)
		}
	}
}

func TestSearch_LetterBoundBackWorksInListFocus(t *testing.T) {
	m := searchLoadedModel(t, config.KeysConfig{"global_back": {"x"}})
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)
	updated, _ = m.Update(typed('x'))
	if got := updated.(Model).state; got != stateGrid {
		t.Fatalf("state after x in list focus = %d, want stateGrid", got)
	}
}

func TestSearch_FocusToggleIsConfigurable(t *testing.T) {
	m := searchLoadedModel(t, config.KeysConfig{"focus_toggle": {"ctrl+n"}})
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)
	if m.searchFocus != focusInput {
		t.Fatal("tab must not toggle focus when focus_toggle is rebound")
	}
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m = updated.(Model)
	if m.searchFocus != focusList {
		t.Fatal("ctrl+n should toggle focus to the list")
	}
}

func TestSearch_EnterSearchResetsFocusToField(t *testing.T) {
	m := searchLoadedModel(t, nil)
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/'})
	m = updated.(Model)
	if m.searchFocus != focusInput {
		t.Fatalf("focus on re-entering search = %d, want focusInput", m.searchFocus)
	}
}

// --- Add/edit form ---

func TestEdit_TypingIsNotInterceptedByBindings(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 42, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.5, Name: ""},
	}
	tests := []struct {
		name string
		keys config.KeysConfig
		text string
	}{
		{"default bindings", nil, "qaedjkhlrc?"},
		{"letter-bound back", config.KeysConfig{"global_back": {"x"}}, "x"},
		{"letter-bound focus toggle", config.KeysConfig{"focus_toggle": {"f"}}, "f"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockClient{entries: entries}
			mon := MondayTime{Time: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)}
			m := NewModel(client, mon, config.DefaultHoursLimits(), "Deutschland", tt.keys, nil)
			m.state = stateDetail
			m.grid = BuildWeekGrid(entries, mon.Time)
			m.cursor = [2]int{0, 0}

			updated, _ := m.Update(tea.KeyPressMsg{Code: 'e'})
			m = updated.(Model)
			if m.state != stateEdit {
				t.Fatalf("state = %d, want stateEdit", m.state)
			}
			// Type into the description field; when the focus toggle itself is
			// rebound to a letter, stay on the hours field and type there.
			if _, rebound := tt.keys["focus_toggle"]; !rebound {
				updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
				m = updated.(Model)
			}
			for _, r := range tt.text {
				updated, _ = m.Update(typed(r))
				m = updated.(Model)
			}
			if m.state != stateEdit {
				t.Fatalf("state = %d, want stateEdit (typing must not leave the form)", m.state)
			}
			got, want := m.editDesc.Value(), tt.text
			if _, rebound := tt.keys["focus_toggle"]; rebound {
				got, want = m.editHours.Value(), "2.5"+tt.text // hours field is pre-filled
			}
			if got != want {
				t.Fatalf("input value = %q, want %q", got, want)
			}
		})
	}
}

func TestEdit_FocusToggleIsConfigurable(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 42, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.5, Name: "x"},
	}
	client := &mockClient{entries: entries}
	mon := MondayTime{Time: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)}
	m := NewModel(client, mon, config.DefaultHoursLimits(), "Deutschland", config.KeysConfig{"focus_toggle": {"ctrl+n"}}, nil)
	m.state = stateDetail
	m.grid = BuildWeekGrid(entries, mon.Time)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'e'})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)
	if m.editFocus != 0 {
		t.Fatal("tab must not toggle focus when focus_toggle is rebound")
	}
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m = updated.(Model)
	if m.editFocus != 1 {
		t.Fatal("ctrl+n should move focus to the description")
	}
}

// --- Rendering ---

func TestRenderSearchOverlay_ShowsFocus(t *testing.T) {
	input := textinput.New()
	items := []searchItem{
		{Kind: "project", Name: "Alpha"},
		{Kind: "project", Name: "Beta"},
	}
	km := DefaultKeyMap()

	fieldView := renderSearchOverlay(input, items, 0, searchReady, true, nil, spinner.New(), 80, 40, nil, focusInput, km)
	listView := renderSearchOverlay(input, items, 0, searchReady, true, nil, spinner.New(), 80, 40, nil, focusList, km)

	activePrompt := regexp.MustCompile(regexp.QuoteMeta(stylePrefix(t, editActiveLabelStyle)) + `\s*>`)
	if !activePrompt.MatchString(fieldView) {
		t.Error("field focus: prompt should be rendered with the active label style")
	}
	if activePrompt.MatchString(listView) {
		t.Error("list focus: prompt should not be rendered with the active label style")
	}

	listCursor := regexp.MustCompile(regexp.QuoteMeta(stylePrefix(t, cursorStyle)) + `[^\n]*Alpha`)
	if !listCursor.MatchString(listView) {
		t.Error("list focus: selected row should use the bright cursor style")
	}
	if listCursor.MatchString(fieldView) {
		t.Error("field focus: selected row should not use the bright cursor style")
	}

	for _, view := range []string{fieldView, listView} {
		if !strings.Contains(strings.ToLower(view), "tab") {
			t.Error("search hints should mention the Tab focus toggle")
		}
	}
	if !strings.Contains(listView, "↑/k") || !strings.Contains(listView, "↓/j") {
		t.Error("list focus hint should show the configured cursor keys")
	}
	if strings.Contains(fieldView, "↑/k") {
		t.Error("field focus hint must not suggest that k navigates while typing")
	}
}

func TestRenderEditForm_HintShowsFocusToggleKey(t *testing.T) {
	km := ApplyKeysConfig(DefaultKeyMap(), config.KeysConfig{"focus_toggle": {"ctrl+n"}})
	row := GridRow{Label: "Acme / Dev"}
	out := renderEditForm(row, time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC), textinput.New(), textinput.New(), 0, nil, 80, false, km)
	if !strings.Contains(out, "ctrl+n") {
		t.Errorf("edit form hint should show the configured focus toggle key, got %q", out)
	}
}

func TestRenderHelpOverlay_ListsFocusToggle(t *testing.T) {
	out := renderHelpOverlay(DefaultKeyMap(), 80, 40)
	if !strings.Contains(out, "switch focus") {
		t.Error("help overlay should list the focus toggle action")
	}
	if !strings.Contains(out, "tab") {
		t.Error("help overlay should show tab as the focus toggle key")
	}
}

var ansiSeq = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func TestSearchView_RendersSinglePrompt(t *testing.T) {
	m := searchLoadedModel(t, nil)
	updated, _ := m.Update(typed('a'))
	m = updated.(Model)
	view := ansiSeq.ReplaceAllString(m.View().Content, "")
	if strings.Contains(view, "> >") {
		t.Fatalf("search view renders a doubled prompt:\n%s", view)
	}
	if !strings.Contains(view, "> a") {
		t.Fatalf("search view should show the prompt followed by the typed text:\n%s", view)
	}
}

func TestRenderHelpOverlay_AlignsLongKeys(t *testing.T) {
	km := ApplyKeysConfig(DefaultKeyMap(), config.KeysConfig{"focus_toggle": {"tab", "shift+tab"}})
	out := ansiSeq.ReplaceAllString(renderHelpOverlay(km, 80, 40), "")
	col := func(desc string) int {
		for _, line := range strings.Split(out, "\n") {
			if i := strings.Index(line, desc); i >= 0 {
				return i
			}
		}
		t.Fatalf("help overlay lacks %q", desc)
		return -1
	}
	if a, b := col("toggle filter"), col("switch focus"); a != b {
		t.Errorf("description columns differ: %q at %d, %q at %d", "toggle filter", a, "switch focus", b)
	}
}
