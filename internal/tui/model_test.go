package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// mockClient implements odoo.Client for testing.
type mockClient struct {
	entries      []odoo.TimesheetEntry
	err          error
	updateErr    error
	updatedID    int64
	updated      map[string]interface{} // capture last update call
	createErr    error
	createdID    int64
	createParams odoo.TimesheetWriteParams // capture last create call
}

func (c *mockClient) WhoAmI() (*odoo.UserInfo, error)              { return nil, nil }
func (c *mockClient) ListProjects() ([]odoo.ProjectInfo, error)    { return nil, nil }
func (c *mockClient) ListTasks(int64) ([]odoo.TaskInfo, error)     { return nil, nil }
func (c *mockClient) GetFields(string) ([]odoo.FieldInfo, error)   { return nil, nil }
func (c *mockClient) ListTimesheets(string, string) ([]odoo.TimesheetEntry, error) {
	return c.entries, c.err
}
func (c *mockClient) CreateTimesheet(params odoo.TimesheetWriteParams) (int64, error) {
	c.createParams = params
	return c.createdID, c.createErr
}
func (c *mockClient) UpdateTimesheet(id int64, fields map[string]interface{}) error {
	c.updatedID = id
	c.updated = fields
	return c.updateErr
}
func (c *mockClient) DeleteTimesheet(int64) error                              { return nil }
func (c *mockClient) ClockIn() (int64, error)                                  { return 0, nil }
func (c *mockClient) ClockOut() (*odoo.AttendanceRecord, error)                { return nil, nil }
func (c *mockClient) AttendanceStatus() (*odoo.AttendanceStatus, error)        { return nil, nil }

func newTestModel(entries []odoo.TimesheetEntry, err error) Model {
	client := &mockClient{entries: entries, err: err}
	mon := MondayTime{Time: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)}
	return NewModel(client, mon, config.DefaultHoursLimits(), "Deutschland")
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

func TestModel_AttendanceLoadedClockedIn(t *testing.T) {
	m := newTestModel(nil, nil)
	m.state = stateGrid

	checkIn := time.Now().Add(-2 * time.Hour)
	msg := attendanceLoadedMsg{
		status: &odoo.AttendanceStatus{
			ClockedIn: true,
			CheckIn:   &checkIn,
		},
	}
	updated, cmd := m.Update(msg)
	um := updated.(Model)

	if um.attendance == nil {
		t.Fatal("expected attendance to be set")
	}
	if !um.attendance.ClockedIn {
		t.Fatal("expected ClockedIn to be true")
	}
	if cmd == nil {
		t.Fatal("expected tick command when clocked in")
	}
}

func TestModel_AttendanceLoadedNotClockedIn(t *testing.T) {
	m := newTestModel(nil, nil)
	m.state = stateGrid

	msg := attendanceLoadedMsg{
		status: &odoo.AttendanceStatus{
			ClockedIn: false,
		},
	}
	updated, cmd := m.Update(msg)
	um := updated.(Model)

	if um.attendance == nil {
		t.Fatal("expected attendance to be set")
	}
	if um.attendance.ClockedIn {
		t.Fatal("expected ClockedIn to be false")
	}
	if cmd != nil {
		t.Fatal("expected no tick command when not clocked in")
	}
}

func TestModel_AttendanceLoadedError(t *testing.T) {
	m := newTestModel(nil, nil)
	m.state = stateGrid

	msg := attendanceLoadedMsg{err: errors.New("network error")}
	updated, cmd := m.Update(msg)
	um := updated.(Model)

	if um.attendance != nil {
		t.Fatal("expected attendance to be nil on error")
	}
	if cmd != nil {
		t.Fatal("expected no tick command on error")
	}
	// State should not change to error for attendance failure.
	if um.state != stateGrid {
		t.Fatalf("expected stateGrid, got %d", um.state)
	}
}

func TestModel_AttendanceTickContinuesWhenClockedIn(t *testing.T) {
	m := newTestModel(nil, nil)
	checkIn := time.Now().Add(-time.Hour)
	m.attendance = &odoo.AttendanceStatus{ClockedIn: true, CheckIn: &checkIn}

	_, cmd := m.Update(attendanceTickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("expected tick to continue when clocked in")
	}
}

func TestModel_AttendanceTickStopsWhenNotClockedIn(t *testing.T) {
	m := newTestModel(nil, nil)
	m.attendance = &odoo.AttendanceStatus{ClockedIn: false}

	_, cmd := m.Update(attendanceTickMsg(time.Now()))
	if cmd != nil {
		t.Fatal("expected tick to stop when not clocked in")
	}
}

func TestRenderClockStatus_Nil(t *testing.T) {
	result := renderClockStatus(nil)
	if result != "" {
		t.Fatalf("expected empty string for nil attendance, got %q", result)
	}
}

func TestRenderClockStatus_NotClockedIn(t *testing.T) {
	status := &odoo.AttendanceStatus{ClockedIn: false}
	result := renderClockStatus(status)
	if !strings.Contains(result, "Not clocked in") {
		t.Fatalf("expected 'Not clocked in' in %q", result)
	}
}

func TestRenderClockStatus_ClockedIn(t *testing.T) {
	checkIn := time.Now().Add(-90 * time.Minute)
	status := &odoo.AttendanceStatus{ClockedIn: true, CheckIn: &checkIn}
	result := renderClockStatus(status)
	if !strings.Contains(result, "Clocked in since") {
		t.Fatalf("expected 'Clocked in since' in %q", result)
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

// helper to build a model in detail state with entries.
func newDetailModel(entries []odoo.TimesheetEntry) Model {
	client := &mockClient{entries: entries}
	mon := MondayTime{Time: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)}
	m := NewModel(client, mon, config.DefaultHoursLimits(), "Deutschland")
	m.state = stateGrid
	m.grid = BuildWeekGrid(entries, mon.Time)
	m.cursor = [2]int{0, 0}
	m.width = 120
	m.height = 40

	// Transition to detail view.
	updated, _ := m.Update(tea.KeyPressMsg{Code: '\r'})
	return updated.(Model)
}

func TestModel_DetailCursorMovement(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 1, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.0, Name: "Task A"},
		{ID: 2, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 3.0, Name: "Task B"},
		{ID: 3, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 1.5, Name: "Task C"},
	}
	m := newDetailModel(entries)

	if m.state != stateDetail {
		t.Fatalf("expected stateDetail, got %d", m.state)
	}
	if m.detailCursor != 0 {
		t.Fatalf("expected detailCursor 0, got %d", m.detailCursor)
	}

	// Move down.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	um := updated.(Model)
	if um.detailCursor != 1 {
		t.Fatalf("expected detailCursor 1, got %d", um.detailCursor)
	}

	// Move down again.
	updated, _ = um.Update(tea.KeyPressMsg{Code: 'j'})
	um = updated.(Model)
	if um.detailCursor != 2 {
		t.Fatalf("expected detailCursor 2, got %d", um.detailCursor)
	}

	// Move down at bottom stays clamped.
	updated, _ = um.Update(tea.KeyPressMsg{Code: 'j'})
	um = updated.(Model)
	if um.detailCursor != 2 {
		t.Fatalf("expected detailCursor 2 (clamped), got %d", um.detailCursor)
	}

	// Move up.
	updated, _ = um.Update(tea.KeyPressMsg{Code: 'k'})
	um = updated.(Model)
	if um.detailCursor != 1 {
		t.Fatalf("expected detailCursor 1, got %d", um.detailCursor)
	}
}

func TestModel_EditTransition(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 42, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.5, Name: "Fix login"},
	}
	m := newDetailModel(entries)

	// Press 'e' to enter edit mode.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'e'})
	um := updated.(Model)

	if um.state != stateEdit {
		t.Fatalf("expected stateEdit, got %d", um.state)
	}
	if um.editHours.Value() != "2.5" {
		t.Fatalf("expected hours '2.5', got %q", um.editHours.Value())
	}
	if um.editDesc.Value() != "Fix login" {
		t.Fatalf("expected desc 'Fix login', got %q", um.editDesc.Value())
	}
	if um.editFocus != 0 {
		t.Fatalf("expected editFocus 0 (hours), got %d", um.editFocus)
	}
}

func TestModel_EditEscCancels(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 42, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.5, Name: "Fix login"},
	}
	m := newDetailModel(entries)

	// Enter edit.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'e'})
	um := updated.(Model)
	if um.state != stateEdit {
		t.Fatalf("expected stateEdit, got %d", um.state)
	}

	// Esc cancels back to detail.
	updated, _ = um.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	um = updated.(Model)
	if um.state != stateDetail {
		t.Fatalf("expected stateDetail after esc, got %d", um.state)
	}
}

func TestModel_EditTabTogglesFocus(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 42, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.5, Name: "Fix login"},
	}
	m := newDetailModel(entries)

	// Enter edit.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'e'})
	um := updated.(Model)
	if um.editFocus != 0 {
		t.Fatalf("expected editFocus 0, got %d", um.editFocus)
	}

	// Tab toggles to description.
	updated, _ = um.Update(tea.KeyPressMsg{Code: '\t'})
	um = updated.(Model)
	if um.editFocus != 1 {
		t.Fatalf("expected editFocus 1, got %d", um.editFocus)
	}

	// Tab toggles back to hours.
	updated, _ = um.Update(tea.KeyPressMsg{Code: '\t'})
	um = updated.(Model)
	if um.editFocus != 0 {
		t.Fatalf("expected editFocus 0, got %d", um.editFocus)
	}
}

func TestModel_EditSubmitSuccess(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 42, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.5, Name: "Fix login"},
	}
	client := &mockClient{entries: entries}
	mon := MondayTime{Time: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)}
	m := NewModel(client, mon, config.DefaultHoursLimits(), "Deutschland")
	m.state = stateGrid
	m.grid = BuildWeekGrid(entries, mon.Time)
	m.cursor = [2]int{0, 0}
	m.width = 120
	m.height = 40

	// Enter detail.
	updated, _ := m.Update(tea.KeyPressMsg{Code: '\r'})
	um := updated.(Model)
	// Enter edit.
	updated, _ = um.Update(tea.KeyPressMsg{Code: 'e'})
	um = updated.(Model)

	// Modify hours value.
	um.editHours.SetValue("3.0")
	um.editDesc.SetValue("Updated description")

	// Submit with Enter.
	updated, cmd := um.Update(tea.KeyPressMsg{Code: '\r'})
	um = updated.(Model)

	// Should produce a command (the async save).
	if cmd == nil {
		t.Fatal("expected save command")
	}

	// Execute the command to get the message.
	msg := cmd()
	savedMsg, ok := msg.(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", msg)
	}
	if savedMsg.err != nil {
		t.Fatalf("expected no error, got %v", savedMsg.err)
	}

	// Verify the mock captured the update.
	if client.updatedID != 42 {
		t.Fatalf("expected update ID 42, got %d", client.updatedID)
	}
	if client.updated["unit_amount"] != 3.0 {
		t.Fatalf("expected hours 3.0, got %v", client.updated["unit_amount"])
	}
	if client.updated["name"] != "Updated description" {
		t.Fatalf("expected desc 'Updated description', got %v", client.updated["name"])
	}

	// Handle the editSavedMsg — should go back to detail and trigger reload.
	updated, reloadCmd := um.Update(savedMsg)
	um = updated.(Model)
	if um.state != stateDetail {
		t.Fatalf("expected stateDetail after save, got %d", um.state)
	}
	if !um.loading {
		t.Fatal("expected loading=true after save")
	}
	if reloadCmd == nil {
		t.Fatal("expected reload command after save")
	}
}

func TestModel_EditSubmitError(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 42, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.5, Name: "Fix login"},
	}
	m := newDetailModel(entries)

	// Enter edit.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'e'})
	um := updated.(Model)

	// Handle editSavedMsg with error — should stay in edit with error.
	updated, _ = um.Update(editSavedMsg{err: errors.New("server error")})
	um = updated.(Model)
	if um.editErr == nil {
		t.Fatal("expected editErr to be set")
	}
	if um.state != stateEdit {
		t.Fatalf("expected stateEdit on error, got %d", um.state)
	}
}

func TestModel_EditValidation(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 42, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.5, Name: "Fix login"},
	}
	m := newDetailModel(entries)

	// Enter edit.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'e'})
	um := updated.(Model)

	// Set invalid hours.
	um.editHours.SetValue("0")
	um.editDesc.SetValue("Something")
	updated, cmd := um.Update(tea.KeyPressMsg{Code: '\r'})
	um = updated.(Model)
	if um.editErr == nil {
		t.Fatal("expected validation error for hours <= 0")
	}
	if cmd != nil {
		t.Fatal("expected no command on validation error")
	}

	// Set valid hours but empty description.
	um.editHours.SetValue("1.5")
	um.editDesc.SetValue("")
	updated, cmd = um.Update(tea.KeyPressMsg{Code: '\r'})
	um = updated.(Model)
	if um.editErr == nil {
		t.Fatal("expected validation error for empty description")
	}
	if cmd != nil {
		t.Fatal("expected no command on validation error")
	}

	// Set negative hours.
	um.editHours.SetValue("-1")
	um.editDesc.SetValue("Something")
	updated, cmd = um.Update(tea.KeyPressMsg{Code: '\r'})
	um = updated.(Model)
	if um.editErr == nil {
		t.Fatal("expected validation error for negative hours")
	}
	if cmd != nil {
		t.Fatal("expected no command on validation error")
	}

	// Set non-numeric hours.
	um.editHours.SetValue("abc")
	um.editDesc.SetValue("Something")
	updated, cmd = um.Update(tea.KeyPressMsg{Code: '\r'})
	um = updated.(Model)
	if um.editErr == nil {
		t.Fatal("expected validation error for non-numeric hours")
	}
	if cmd != nil {
		t.Fatal("expected no command on validation error")
	}
}

func TestModel_EditSaveStaysInDetail(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 42, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.5, Name: "Fix login"},
	}
	m := newDetailModel(entries)

	// Enter edit, submit, handle save msg, then handle reload.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'e'})
	um := updated.(Model)
	um.editHours.SetValue("3.0")
	um.editDesc.SetValue("Updated")
	updated, cmd := um.Update(tea.KeyPressMsg{Code: '\r'})
	um = updated.(Model)

	// Simulate editSavedMsg success.
	updated, _ = um.Update(editSavedMsg{err: nil})
	um = updated.(Model)
	if um.state != stateDetail {
		t.Fatalf("expected stateDetail after save, got %d", um.state)
	}

	// Simulate timesheetsLoadedMsg from reload — should stay in detail.
	updated, _ = um.Update(timesheetsLoadedMsg{entries: entries})
	um = updated.(Model)
	if um.state != stateDetail {
		t.Fatalf("expected stateDetail after reload, got %d", um.state)
	}
	_ = cmd
}

func TestModel_EditNoEntriesIgnoresEdit(t *testing.T) {
	// Day with no entries — 'e' should not transition to edit.
	entries := []odoo.TimesheetEntry{
		{ID: 1, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.0, Name: "Task A"},
	}
	m := newDetailModel(entries)
	// Move to Tuesday (col 1) which has no entries.
	m.cursor[1] = 1
	m.state = stateDetail

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'e'})
	um := updated.(Model)
	if um.state != stateDetail {
		t.Fatalf("expected stateDetail when no entries, got %d", um.state)
	}
}

func TestGridRow_ProjectTaskIDs(t *testing.T) {
	row := GridRow{
		Label: "Acme / Dev",
		Entries: [7][]odoo.TimesheetEntry{
			{}, // Mon: empty
			{{ID: 1, ProjectID: 10, TaskID: 20}}, // Tue: has entry
			{}, {}, {}, {}, {},
		},
	}
	pid, tid := row.ProjectTaskIDs()
	if pid != 10 {
		t.Fatalf("expected projectID 10, got %d", pid)
	}
	if tid != 20 {
		t.Fatalf("expected taskID 20, got %d", tid)
	}
}

func TestGridRow_ProjectTaskIDs_Empty(t *testing.T) {
	row := GridRow{Label: "Acme / Dev"}
	pid, tid := row.ProjectTaskIDs()
	if pid != 0 || tid != 0 {
		t.Fatalf("expected 0/0 for empty row, got %d/%d", pid, tid)
	}
}

func TestModel_AddTransition(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 1, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.0, Name: "Task A", ProjectID: 10, TaskID: 20},
	}
	m := newDetailModel(entries)

	// Press 'a' to add a new entry.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'a'})
	um := updated.(Model)

	if um.state != stateEdit {
		t.Fatalf("expected stateEdit, got %d", um.state)
	}
	if !um.editIsNew {
		t.Fatal("expected editIsNew=true")
	}
	if um.editHours.Value() != "" {
		t.Fatalf("expected empty hours, got %q", um.editHours.Value())
	}
	if um.editDesc.Value() != "" {
		t.Fatalf("expected empty desc, got %q", um.editDesc.Value())
	}
	if um.editFocus != 0 {
		t.Fatalf("expected editFocus 0, got %d", um.editFocus)
	}
}

func TestModel_AddEmptyCell(t *testing.T) {
	// Entry on Monday, but we open detail on Tuesday (empty cell).
	entries := []odoo.TimesheetEntry{
		{ID: 1, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.0, Name: "Task A", ProjectID: 10, TaskID: 20},
	}
	m := newDetailModel(entries)
	// Move to Tuesday (col 1).
	m.cursor[1] = 1
	m.state = stateDetail

	// Press 'a' — should still work (gets IDs from Monday's entry).
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'a'})
	um := updated.(Model)

	if um.state != stateEdit {
		t.Fatalf("expected stateEdit, got %d", um.state)
	}
	if !um.editIsNew {
		t.Fatal("expected editIsNew=true")
	}
}

func TestModel_AddSubmitCallsCreate(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 1, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.0, Name: "Task A", ProjectID: 10, TaskID: 20},
	}
	client := &mockClient{entries: entries, createdID: 99}
	mon := MondayTime{Time: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)}
	m := NewModel(client, mon, config.DefaultHoursLimits(), "Deutschland")
	m.state = stateGrid
	m.grid = BuildWeekGrid(entries, mon.Time)
	m.cursor = [2]int{0, 0}
	m.width = 120
	m.height = 40

	// Enter detail.
	updated, _ := m.Update(tea.KeyPressMsg{Code: '\r'})
	um := updated.(Model)
	// Press 'a' to add.
	updated, _ = um.Update(tea.KeyPressMsg{Code: 'a'})
	um = updated.(Model)

	// Fill in values.
	um.editHours.SetValue("1.5")
	um.editDesc.SetValue("New task")

	// Submit.
	updated, cmd := um.Update(tea.KeyPressMsg{Code: '\r'})
	um = updated.(Model)

	if cmd == nil {
		t.Fatal("expected save command")
	}

	// Execute command.
	msg := cmd()
	savedMsg, ok := msg.(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", msg)
	}
	if savedMsg.err != nil {
		t.Fatalf("expected no error, got %v", savedMsg.err)
	}

	// Verify create was called with correct params.
	if client.createParams.ProjectID != 10 {
		t.Fatalf("expected projectID 10, got %d", client.createParams.ProjectID)
	}
	if client.createParams.TaskID != 20 {
		t.Fatalf("expected taskID 20, got %d", client.createParams.TaskID)
	}
	if client.createParams.Date != "2026-03-02" {
		t.Fatalf("expected date 2026-03-02, got %q", client.createParams.Date)
	}
	if client.createParams.Hours != 1.5 {
		t.Fatalf("expected hours 1.5, got %v", client.createParams.Hours)
	}
	if client.createParams.Name != "New task" {
		t.Fatalf("expected name 'New task', got %q", client.createParams.Name)
	}

	// Handle saved msg — should return to detail and reload.
	updated, reloadCmd := um.Update(savedMsg)
	um = updated.(Model)
	if um.state != stateDetail {
		t.Fatalf("expected stateDetail after save, got %d", um.state)
	}
	if !um.loading {
		t.Fatal("expected loading=true after save")
	}
	if reloadCmd == nil {
		t.Fatal("expected reload command after save")
	}
}

func TestModel_AddSubmitValidation(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 1, Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.0, Name: "Task A", ProjectID: 10, TaskID: 20},
	}
	m := newDetailModel(entries)

	// Enter add mode.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'a'})
	um := updated.(Model)

	// Submit with empty hours.
	um.editHours.SetValue("")
	um.editDesc.SetValue("Something")
	updated, cmd := um.Update(tea.KeyPressMsg{Code: '\r'})
	um = updated.(Model)
	if um.editErr == nil {
		t.Fatal("expected validation error for empty hours")
	}
	if cmd != nil {
		t.Fatal("expected no command on validation error")
	}

	// Submit with valid hours but empty description.
	um.editHours.SetValue("1.0")
	um.editDesc.SetValue("")
	updated, cmd = um.Update(tea.KeyPressMsg{Code: '\r'})
	um = updated.(Model)
	if um.editErr == nil {
		t.Fatal("expected validation error for empty description")
	}
	if cmd != nil {
		t.Fatal("expected no command on validation error")
	}
}
