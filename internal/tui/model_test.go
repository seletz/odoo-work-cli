package tui

import (
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// mockClient implements odoo.Client for testing.
type mockClient struct {
	entries []odoo.TimesheetEntry
	err     error
}

func (c *mockClient) WhoAmI() (*odoo.UserInfo, error)              { return nil, nil }
func (c *mockClient) ListProjects() ([]odoo.ProjectInfo, error)    { return nil, nil }
func (c *mockClient) ListTasks(int64) ([]odoo.TaskInfo, error)     { return nil, nil }
func (c *mockClient) GetFields(string) ([]odoo.FieldInfo, error)   { return nil, nil }
func (c *mockClient) ListTimesheets(string, string) ([]odoo.TimesheetEntry, error) {
	return c.entries, c.err
}

func newTestModel(entries []odoo.TimesheetEntry, err error) Model {
	client := &mockClient{entries: entries, err: err}
	mon := MondayTime{Time: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)}
	return NewModel(client, mon, config.DefaultHoursLimits())
}

func TestModel_LoadedTransitionsToGrid(t *testing.T) {
	m := newTestModel([]odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 8.0},
	}, nil)

	msg := timesheetsLoadedMsg{
		entries: []odoo.TimesheetEntry{
			{Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 8.0},
		},
	}
	updated, _ := m.Update(msg)
	um := updated.(Model)

	if um.state != stateGrid {
		t.Fatalf("expected stateGrid, got %d", um.state)
	}
	if len(um.grid.Rows) != 1 {
		t.Fatalf("expected 1 grid row, got %d", len(um.grid.Rows))
	}
}

func TestModel_ErrorTransitionsToError(t *testing.T) {
	m := newTestModel(nil, nil)

	msg := timesheetsLoadedMsg{err: errors.New("connection failed")}
	updated, _ := m.Update(msg)
	um := updated.(Model)

	if um.state != stateError {
		t.Fatalf("expected stateError, got %d", um.state)
	}
	if um.err == nil {
		t.Fatal("expected error to be set")
	}
}

func TestModel_CursorMovement(t *testing.T) {
	m := newTestModel(nil, nil)
	// Simulate loaded state with 3 rows.
	m.state = stateGrid
	m.grid = BuildWeekGrid([]odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "A", Task: "1", Hours: 1},
		{Date: "2026-03-02", Project: "B", Task: "2", Hours: 1},
		{Date: "2026-03-02", Project: "C", Task: "3", Hours: 1},
	}, m.monday.Time)
	m.cursor = [2]int{0, 0}

	// Move down.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	um := updated.(Model)
	if um.cursor[0] != 1 {
		t.Fatalf("expected cursor row 1, got %d", um.cursor[0])
	}

	// Move down again.
	updated, _ = um.Update(tea.KeyPressMsg{Code: 'j'})
	um = updated.(Model)
	if um.cursor[0] != 2 {
		t.Fatalf("expected cursor row 2, got %d", um.cursor[0])
	}

	// Move down at bottom stays at bottom.
	updated, _ = um.Update(tea.KeyPressMsg{Code: 'j'})
	um = updated.(Model)
	if um.cursor[0] != 2 {
		t.Fatalf("expected cursor row 2 (clamped), got %d", um.cursor[0])
	}

	// Move up.
	updated, _ = um.Update(tea.KeyPressMsg{Code: 'k'})
	um = updated.(Model)
	if um.cursor[0] != 1 {
		t.Fatalf("expected cursor row 1, got %d", um.cursor[0])
	}

	// Tab moves column right.
	updated, _ = um.Update(tea.KeyPressMsg{Code: '\t'})
	um = updated.(Model)
	if um.cursor[1] != 1 {
		t.Fatalf("expected cursor col 1, got %d", um.cursor[1])
	}
}

func TestModel_QuitCmd(t *testing.T) {
	m := newTestModel(nil, nil)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestModel_WeekNavigation(t *testing.T) {
	m := newTestModel(nil, nil)
	original := m.monday.Time

	// Right arrow = next week.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'l'})
	um := updated.(Model)
	expected := original.AddDate(0, 0, 7)
	if !um.monday.Equal(expected) {
		t.Fatalf("expected monday %v, got %v", expected, um.monday.Time)
	}
	if um.state != stateLoading {
		t.Fatalf("expected stateLoading after week nav, got %d", um.state)
	}
}

func TestModel_WindowSizeMsg(t *testing.T) {
	m := newTestModel(nil, nil)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	um := updated.(Model)
	if um.width != 120 || um.height != 40 {
		t.Fatalf("expected 120x40, got %dx%d", um.width, um.height)
	}
}
