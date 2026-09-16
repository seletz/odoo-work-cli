package tui

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// Fixed clock for attendance tests: Wed 2026-03-04 18:00 UTC, in the week
// of Monday 2026-03-02 that newTestModel displays.
var attNow = time.Date(2026, 3, 4, 18, 0, 0, 0, time.UTC)

func attDay(d int) time.Time {
	return time.Date(2026, 3, d, 0, 0, 0, 0, time.UTC)
}

func attTime(d, h, min int) time.Time {
	return time.Date(2026, 3, d, h, min, 0, 0, time.UTC)
}

func ptrTime(t time.Time) *time.Time { return &t }

// closedRecord is a full day on Monday 2026-03-02 with odd seconds, as Odoo
// stores them.
func closedRecord() odoo.AttendanceRecord {
	in := time.Date(2026, 3, 2, 8, 58, 42, 0, time.UTC)
	out := time.Date(2026, 3, 2, 17, 3, 7, 0, time.UTC)
	return odoo.AttendanceRecord{ID: 11, CheckIn: in, CheckOut: &out, WorkedHours: 8.07}
}

func openRecord() odoo.AttendanceRecord {
	return odoo.AttendanceRecord{ID: 12, CheckIn: attTime(2, 18, 0)}
}

func newAttendanceTestModel(client *mockClient) Model {
	mon := MondayTime{Time: attDay(2)}
	m := NewModel(client, mon, config.DefaultHoursLimits(), "Deutschland", nil, nil)
	m.state = stateGrid
	m.now = func() time.Time { return attNow }
	return m
}

func TestParseAttendanceTime(t *testing.T) {
	day := attDay(2)
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr string
	}{
		{"hh:mm on day", "09:15", attTime(2, 9, 15), ""},
		{"explicit date", "2026-03-01 23:30", attTime(1, 23, 30), ""},
		{"whitespace trimmed", "  09:15 ", attTime(2, 9, 15), ""},
		{"invalid", "9am", time.Time{}, "invalid time"},
		{"future", "2026-03-04 18:05", time.Time{}, "in the future"},
		{"within grace", "2026-03-04 18:01", attTime(4, 18, 1), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAttendanceTime(tt.input, day, attNow)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildAttendanceChange(t *testing.T) {
	closed := closedRecord()
	open := openRecord()
	day := attDay(2)

	tests := []struct {
		name    string
		record  *odoo.AttendanceRecord
		in, out string
		want    attendanceChange
		wantErr string
	}{
		{
			name: "create with both times",
			in:   "09:00", out: "17:00",
			want: attendanceChange{create: true, checkIn: ptrTime(attTime(2, 9, 0)), checkOut: ptrTime(attTime(2, 17, 0))},
		},
		{
			name:    "create needs check-out",
			in:      "09:00",
			wantErr: "both",
		},
		{
			name:    "create needs check-in",
			out:     "17:00",
			wantErr: "both",
		},
		{
			name: "create out before in",
			in:   "17:00", out: "09:00",
			wantErr: "not after",
		},
		{
			name:   "edit unchanged is nothing to change",
			record: &closed,
			in:     "08:58", out: "17:03",
			wantErr: "nothing to change",
		},
		{
			name:   "edit check-in only",
			record: &closed,
			in:     "09:00", out: "17:03",
			want: attendanceChange{id: 11, checkIn: ptrTime(attTime(2, 9, 0))},
		},
		{
			name:   "edit check-out only",
			record: &closed,
			in:     "08:58", out: "17:30",
			want: attendanceChange{id: 11, checkOut: ptrTime(attTime(2, 17, 30))},
		},
		{
			name:   "edit both",
			record: &closed,
			in:     "09:00", out: "17:30",
			want: attendanceChange{id: 11, checkIn: ptrTime(attTime(2, 9, 0)), checkOut: ptrTime(attTime(2, 17, 30))},
		},
		{
			name:   "edit check-in required",
			record: &closed,
			in:     "", out: "17:03",
			wantErr: "check-in is required",
		},
		{
			name:   "edit cannot clear check-out of closed record",
			record: &closed,
			in:     "08:58", out: "",
			wantErr: "check-out is required",
		},
		{
			name:   "edit new check-in after existing check-out",
			record: &closed,
			in:     "17:30", out: "17:03",
			wantErr: "not after",
		},
		{
			name:   "edit new check-out before existing check-in",
			record: &closed,
			in:     "08:58", out: "08:30",
			wantErr: "not after",
		},
		{
			name:   "edit open record keeps it open",
			record: &open,
			in:     "17:45", out: "",
			want: attendanceChange{id: 12, checkIn: ptrTime(attTime(2, 17, 45))},
		},
		{
			name:   "edit open record closes it",
			record: &open,
			in:     "18:00", out: "2026-03-03 02:00",
			want: attendanceChange{id: 12, checkOut: ptrTime(attTime(3, 2, 0))},
		},
		{
			name:   "edit rejects future",
			record: &open,
			in:     "18:00", out: "2026-03-04 19:00",
			wantErr: "in the future",
		},
		{
			name:   "edit rejects garbage",
			record: &open,
			in:     "eight", out: "",
			wantErr: "invalid time",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildAttendanceChange(tt.record, tt.in, tt.out, day, attNow)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.create != tt.want.create || got.id != tt.want.id {
				t.Errorf("create/id = %v/%d, want %v/%d", got.create, got.id, tt.want.create, tt.want.id)
			}
			assertTimePtr(t, "checkIn", got.checkIn, tt.want.checkIn)
			assertTimePtr(t, "checkOut", got.checkOut, tt.want.checkOut)
		})
	}
}

func assertTimePtr(t *testing.T, label string, got, want *time.Time) {
	t.Helper()
	switch {
	case got == nil && want == nil:
	case got == nil || want == nil:
		t.Errorf("%s = %v, want %v", label, got, want)
	case !got.Equal(*want):
		t.Errorf("%s = %v, want %v", label, *got, *want)
	}
}

func TestModel_AttendanceKeyOpensOverlayForCursorDay(t *testing.T) {
	client := &mockClient{}
	m := newAttendanceTestModel(client)
	m.cursor[1] = 2 // Wednesday

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 't'})
	um := updated.(Model)

	if um.state != stateAttendance {
		t.Fatalf("state = %v, want stateAttendance", um.state)
	}
	if um.att.sub != attLoading {
		t.Errorf("sub = %v, want attLoading", um.att.sub)
	}
	if um.att.prevState != stateGrid {
		t.Errorf("prevState = %v, want stateGrid", um.att.prevState)
	}
	if !um.att.day.Equal(attDay(4)) {
		t.Errorf("day = %v, want Wed 2026-03-04", um.att.day)
	}
	if !um.loading {
		t.Error("expected loading=true while the day's records load")
	}
	if cmd == nil {
		t.Fatal("expected a load command")
	}
	execCmd(cmd)
	if !client.attendWeekFrom.Equal(attDay(4)) || !client.attendWeekTo.Equal(attDay(5)) {
		t.Errorf("ListAttendance range = [%v, %v), want [Wed, Thu)", client.attendWeekFrom, client.attendWeekTo)
	}
}

func TestModel_AttendanceKeyIgnoredOutsideGrid(t *testing.T) {
	for _, st := range []uiState{stateDetail, stateEdit, stateSearch} {
		m := newAttendanceTestModel(&mockClient{})
		m.state = st
		updated, _ := m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
		if um := updated.(Model); um.state != st {
			t.Errorf("state %v: 't' changed state to %v", st, um.state)
		}
	}
}

func TestModel_AttendanceDayLoaded(t *testing.T) {
	closed := closedRecord()
	open := openRecord()
	tests := []struct {
		name       string
		records    []odoo.AttendanceRecord
		wantSub    attendanceSubState
		wantIsNew  bool
		wantIn     string
		wantOut    string
		wantRecord int64
	}{
		{"no records creates", nil, attForm, true, "", "", 0},
		{"single record edits", []odoo.AttendanceRecord{closed}, attForm, false, "08:58", "17:03", 11},
		{"single open record", []odoo.AttendanceRecord{open}, attForm, false, "18:00", "", 12},
		{"several records pick", []odoo.AttendanceRecord{closed, open}, attPick, false, "", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newAttendanceTestModel(&mockClient{})
			m.state = stateAttendance
			m.att = attendanceState{sub: attLoading, day: attDay(2), prevState: stateGrid}
			m.loading = true

			updated, _ := m.Update(attendanceDayLoadedMsg{records: tt.records})
			um := updated.(Model)

			if um.loading {
				t.Error("loading should be cleared")
			}
			if um.att.sub != tt.wantSub {
				t.Fatalf("sub = %v, want %v", um.att.sub, tt.wantSub)
			}
			if tt.wantSub != attForm {
				return
			}
			if um.att.isNew() != tt.wantIsNew {
				t.Errorf("isNew = %v, want %v", um.att.isNew(), tt.wantIsNew)
			}
			if got := um.att.inInput.Value(); got != tt.wantIn {
				t.Errorf("check-in input = %q, want %q", got, tt.wantIn)
			}
			if got := um.att.outInput.Value(); got != tt.wantOut {
				t.Errorf("check-out input = %q, want %q", got, tt.wantOut)
			}
			if tt.wantRecord != 0 && (um.att.record == nil || um.att.record.ID != tt.wantRecord) {
				t.Errorf("record = %v, want ID %d", um.att.record, tt.wantRecord)
			}
		})
	}
}

func TestModel_AttendanceDayLoadedError_FormStaysUsable(t *testing.T) {
	m := newAttendanceTestModel(&mockClient{})
	m.state = stateAttendance
	m.att = attendanceState{sub: attLoading, day: attDay(2), prevState: stateGrid}
	m.loading = true

	updated, _ := m.Update(attendanceDayLoadedMsg{err: errors.New("boom")})
	um := updated.(Model)

	if um.state != stateAttendance || um.att.sub != attForm {
		t.Fatalf("state/sub = %v/%v, want attendance form", um.state, um.att.sub)
	}
	if um.att.err == nil || !strings.Contains(um.att.err.Error(), "boom") {
		t.Errorf("err = %v, want the load error", um.att.err)
	}
	// Esc still returns to the grid.
	updated, _ = um.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if um := updated.(Model); um.state != stateGrid {
		t.Errorf("state after Esc = %v, want stateGrid", um.state)
	}
}

func TestModel_AttendanceDayLoadedIgnoredAfterLeaving(t *testing.T) {
	m := newAttendanceTestModel(&mockClient{})
	m.state = stateGrid
	m.loading = true

	updated, _ := m.Update(attendanceDayLoadedMsg{records: []odoo.AttendanceRecord{closedRecord()}})
	um := updated.(Model)
	if um.state != stateGrid {
		t.Errorf("state = %v, want stateGrid", um.state)
	}
	if um.loading {
		t.Error("loading should be cleared")
	}
}

func loadedAttendanceModel(t *testing.T, client *mockClient, records ...odoo.AttendanceRecord) Model {
	t.Helper()
	m := newAttendanceTestModel(client)
	m.state = stateAttendance
	m.att = attendanceState{sub: attLoading, day: attDay(2), prevState: stateGrid}
	updated, _ := m.Update(attendanceDayLoadedMsg{records: records})
	return updated.(Model)
}

func TestModel_AttendancePickNavigation(t *testing.T) {
	m := loadedAttendanceModel(t, &mockClient{}, closedRecord(), openRecord())
	if m.att.sub != attPick {
		t.Fatalf("sub = %v, want attPick", m.att.sub)
	}

	steps := []struct {
		key        tea.KeyPressMsg
		wantCursor int
	}{
		{tea.KeyPressMsg{Code: tea.KeyUp}, 0},
		{tea.KeyPressMsg{Code: tea.KeyDown}, 1},
		{tea.KeyPressMsg{Code: tea.KeyDown}, 1},
		{tea.KeyPressMsg{Code: 'k'}, 0},
		{tea.KeyPressMsg{Code: 'j'}, 1},
	}
	for i, s := range steps {
		updated, _ := m.Update(s.key)
		m = updated.(Model)
		if m.att.pickCursor != s.wantCursor {
			t.Errorf("step %d: pickCursor = %d, want %d", i, m.att.pickCursor, s.wantCursor)
		}
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)
	if m.att.sub != attForm {
		t.Fatalf("sub after Enter = %v, want attForm", m.att.sub)
	}
	if m.att.record == nil || m.att.record.ID != 12 {
		t.Errorf("record = %v, want the open record #12", m.att.record)
	}
	if got := m.att.inInput.Value(); got != "18:00" {
		t.Errorf("check-in input = %q, want 18:00", got)
	}

	// Esc from the form returns to the pick list, Esc again to the grid.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(Model)
	if m.state != stateAttendance || m.att.sub != attPick {
		t.Fatalf("after Esc: state/sub = %v/%v, want attendance pick", m.state, m.att.sub)
	}
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(Model)
	if m.state != stateGrid {
		t.Errorf("after second Esc: state = %v, want stateGrid", m.state)
	}
}

func TestModel_AttendanceFormEscReturnsToGrid(t *testing.T) {
	m := loadedAttendanceModel(t, &mockClient{}, closedRecord())
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if um := updated.(Model); um.state != stateGrid {
		t.Errorf("state = %v, want stateGrid", um.state)
	}
}

func TestModel_AttendanceFormTabTogglesFocus(t *testing.T) {
	m := loadedAttendanceModel(t, &mockClient{}, closedRecord())
	if m.att.focus != 0 {
		t.Fatalf("initial focus = %d, want 0", m.att.focus)
	}
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)
	if m.att.focus != 1 || !m.att.outInput.Focused() || m.att.inInput.Focused() {
		t.Errorf("after Tab: focus=%d in.Focused=%v out.Focused=%v", m.att.focus, m.att.inInput.Focused(), m.att.outInput.Focused())
	}
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)
	if m.att.focus != 0 || !m.att.inInput.Focused() {
		t.Errorf("after second Tab: focus=%d in.Focused=%v", m.att.focus, m.att.inInput.Focused())
	}
}

func TestModel_AttendanceFormTypingGoesToFocusedInput(t *testing.T) {
	m := loadedAttendanceModel(t, &mockClient{})
	updated, _ := m.Update(tea.KeyPressMsg{Code: '9', Text: "9"})
	m = updated.(Model)
	if got := m.att.inInput.Value(); got != "9" {
		t.Errorf("check-in input = %q, want 9", got)
	}
	// 't' must be typed, not treated as the attendance key.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	m = updated.(Model)
	if got := m.att.inInput.Value(); got != "9t" {
		t.Errorf("check-in input = %q, want 9t", got)
	}
	if m.state != stateAttendance || m.att.sub != attForm {
		t.Errorf("state/sub = %v/%v, want attendance form", m.state, m.att.sub)
	}
}

func setAttendanceInputs(m Model, in, out string) Model {
	m.att.inInput.SetValue(in)
	m.att.outInput.SetValue(out)
	return m
}

func TestModel_AttendanceSubmitEdit(t *testing.T) {
	client := &mockClient{}
	m := loadedAttendanceModel(t, client, closedRecord())
	m = setAttendanceInputs(m, "09:00", "17:03")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	um := updated.(Model)
	if um.att.err != nil {
		t.Fatalf("unexpected validation error: %v", um.att.err)
	}
	if cmd == nil {
		t.Fatal("expected a save command")
	}
	msg := cmd()
	saved, ok := msg.(attendanceSavedMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want attendanceSavedMsg", msg)
	}
	if saved.err != nil {
		t.Fatalf("unexpected save error: %v", saved.err)
	}
	if !client.editAttCalled || client.editAttID != 11 {
		t.Fatalf("EditAttendance called=%v id=%d, want id 11", client.editAttCalled, client.editAttID)
	}
	assertTimePtr(t, "checkIn", client.editAttIn, ptrTime(attTime(2, 9, 0)))
	assertTimePtr(t, "checkOut", client.editAttOut, nil)
	if client.createAttCalled {
		t.Error("CreateAttendance must not be called when editing")
	}
}

func TestModel_AttendanceSubmitCreateMissedDay(t *testing.T) {
	client := &mockClient{}
	m := loadedAttendanceModel(t, client) // no records on the day
	m = setAttendanceInputs(m, "09:00", "17:00")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	um := updated.(Model)
	if um.att.err != nil {
		t.Fatalf("unexpected validation error: %v", um.att.err)
	}
	if cmd == nil {
		t.Fatal("expected a save command")
	}
	if _, ok := cmd().(attendanceSavedMsg); !ok {
		t.Fatal("expected attendanceSavedMsg")
	}
	if !client.createAttCalled {
		t.Fatal("expected CreateAttendance to be called")
	}
	if !client.createAttIn.Equal(attTime(2, 9, 0)) || !client.createAttOut.Equal(attTime(2, 17, 0)) {
		t.Errorf("CreateAttendance(%v, %v), want 09:00-17:00", client.createAttIn, client.createAttOut)
	}
	if client.editAttCalled {
		t.Error("EditAttendance must not be called when creating")
	}
}

func TestModel_AttendanceSubmitValidationInline(t *testing.T) {
	tests := []struct {
		name    string
		records []odoo.AttendanceRecord
		in, out string
		wantErr string
	}{
		{"create missing check-out", nil, "09:00", "", "both"},
		{"invalid time", []odoo.AttendanceRecord{closedRecord()}, "nine", "17:03", "invalid time"},
		{"future", []odoo.AttendanceRecord{closedRecord()}, "08:58", "2026-03-05 09:00", "in the future"},
		{"out before in", []odoo.AttendanceRecord{closedRecord()}, "18:00", "17:03", "not after"},
		{"unchanged", []odoo.AttendanceRecord{closedRecord()}, "08:58", "17:03", "nothing to change"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockClient{}
			m := loadedAttendanceModel(t, client, tt.records...)
			m = setAttendanceInputs(m, tt.in, tt.out)

			updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			um := updated.(Model)
			if cmd != nil {
				t.Error("no command expected on validation failure")
			}
			if um.state != stateAttendance || um.att.sub != attForm {
				t.Errorf("state/sub = %v/%v, want attendance form", um.state, um.att.sub)
			}
			if um.att.err == nil || !strings.Contains(um.att.err.Error(), tt.wantErr) {
				t.Errorf("err = %v, want containing %q", um.att.err, tt.wantErr)
			}
			if client.editAttCalled || client.createAttCalled {
				t.Error("no client call expected on validation failure")
			}
		})
	}
}

func TestModel_AttendanceSavedReloadsAndReturns(t *testing.T) {
	client := &mockClient{attendStatus: &odoo.AttendanceStatus{}}
	m := loadedAttendanceModel(t, client, closedRecord())
	client.attendCalls = 0

	updated, cmd := m.Update(attendanceSavedMsg{})
	um := updated.(Model)
	if um.state != stateGrid {
		t.Fatalf("state = %v, want stateGrid (previous state)", um.state)
	}
	if cmd == nil {
		t.Fatal("expected an attendance reload command")
	}
	execCmd(cmd)
	if client.attendCalls == 0 {
		t.Error("expected ListAttendance to be called to refresh header totals")
	}
	if !client.attendWeekFrom.Equal(attDay(2)) {
		t.Errorf("reload range from = %v, want displayed week Monday", client.attendWeekFrom)
	}
}

func TestModel_AttendanceSavedErrorKeepsFormUsable(t *testing.T) {
	m := loadedAttendanceModel(t, &mockClient{}, closedRecord())
	m = setAttendanceInputs(m, "09:00", "17:03")

	updated, _ := m.Update(attendanceSavedMsg{err: errors.New("Fault(4): denied")})
	um := updated.(Model)
	if um.state != stateAttendance || um.att.sub != attForm {
		t.Fatalf("state/sub = %v/%v, want attendance form", um.state, um.att.sub)
	}
	if um.att.err == nil {
		t.Fatal("expected the save error to be shown")
	}
	if got := um.att.inInput.Value(); got != "09:00" {
		t.Errorf("input value lost after error: %q", got)
	}
	// The form still accepts edits and a retry.
	updated, _ = um.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	um = updated.(Model)
	if got := um.att.inInput.Value(); got != "09:0" {
		t.Errorf("input after backspace = %q, want 09:0", got)
	}
	um = setAttendanceInputs(um, "09:30", "17:03")
	_, cmd := um.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected a retry save command after an error")
	}
}

func TestRenderAttendanceOverlay_Form(t *testing.T) {
	m := loadedAttendanceModel(t, &mockClient{}, closedRecord())
	out := renderAttendanceOverlay(m.att, m.spinner, m.keys)

	for _, want := range []string{"Edit attendance", "Mon 02 Mar 2026", "#11", "08:58", "17:03", "Check-in", "Check-out", "Enter: save"} {
		if !strings.Contains(out, want) {
			t.Errorf("form output should contain %q\n%s", want, out)
		}
	}
}

func TestRenderAttendanceOverlay_CreateForm(t *testing.T) {
	m := loadedAttendanceModel(t, &mockClient{})
	out := renderAttendanceOverlay(m.att, m.spinner, m.keys)
	if !strings.Contains(out, "Add attendance") {
		t.Errorf("create form should say 'Add attendance'\n%s", out)
	}
	if !strings.Contains(out, "no attendance") {
		t.Errorf("create form should explain the day has no record\n%s", out)
	}
}

func TestRenderAttendanceOverlay_Pick(t *testing.T) {
	m := loadedAttendanceModel(t, &mockClient{}, closedRecord(), openRecord())
	m.att.pickCursor = 1
	out := renderAttendanceOverlay(m.att, m.spinner, m.keys)

	if !strings.Contains(out, "#11") || !strings.Contains(out, "#12") {
		t.Errorf("pick list should show both records\n%s", out)
	}
	if !strings.Contains(out, "running") {
		t.Errorf("open record should be marked running\n%s", out)
	}
	selected := regexp.MustCompile(regexp.QuoteMeta(stylePrefix(t, cursorStyle)) + `[^\n]*#12`)
	if !selected.MatchString(out) {
		t.Errorf("record #12 should be rendered with the cursor style\n%s", out)
	}
	unselected := regexp.MustCompile(regexp.QuoteMeta(stylePrefix(t, cursorStyle)) + `[^\n]*#11`)
	if unselected.MatchString(out) {
		t.Errorf("record #11 should not be rendered with the cursor style\n%s", out)
	}
}

func TestRenderAttendanceOverlay_Loading(t *testing.T) {
	m := newAttendanceTestModel(&mockClient{})
	m.att = attendanceState{sub: attLoading, day: attDay(2)}
	out := renderAttendanceOverlay(m.att, m.spinner, m.keys)
	if !strings.Contains(out, "Loading") || !strings.Contains(out, "Mon 02 Mar 2026") {
		t.Errorf("loading overlay should name the day\n%s", out)
	}
}

func TestRenderAttendanceOverlay_Errors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantHint bool
	}{
		{"validation error", errors.New("invalid time \"x\""), false},
		{"raw access error", errors.New("Fault(4): You are not allowed to modify 'Attendance' (hr.attendance) records."), true},
		{"client-wrapped access error", errors.New("updating attendance record 11: Fault(4): denied\nhint: " + odoo.AttendanceAccessHint), true},
		{"other odoo fault", errors.New("Fault(1): internal"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := loadedAttendanceModel(t, &mockClient{}, closedRecord())
			m.att.err = tt.err
			out := renderAttendanceOverlay(m.att, m.spinner, m.keys)

			firstLine := strings.SplitN(tt.err.Error(), "\n", 2)[0]
			if len(firstLine) > 20 {
				firstLine = firstLine[:20]
			}
			errRe := regexp.MustCompile(regexp.QuoteMeta(stylePrefix(t, editErrorStyle)) + `[^\n]*` + regexp.QuoteMeta(firstLine))
			if !errRe.MatchString(out) {
				t.Errorf("error text should be rendered in the error style\n%s", out)
			}
			hintCount := strings.Count(out, "Attendance Officer")
			if tt.wantHint && hintCount != 1 {
				t.Errorf("officer-rights hint count = %d, want exactly 1\n%s", hintCount, out)
			}
			if !tt.wantHint && hintCount != 0 {
				t.Errorf("no officer-rights hint expected\n%s", out)
			}
			if tt.wantHint {
				hintRe := regexp.MustCompile(regexp.QuoteMeta(stylePrefix(t, editHintStyle)) + `[^\n]*Attendance Officer`)
				if !hintRe.MatchString(out) {
					t.Errorf("hint should be rendered in the hint style\n%s", out)
				}
			}
		})
	}
}

func TestView_AttendanceStateRendersOverlayAndStatus(t *testing.T) {
	m := loadedAttendanceModel(t, &mockClient{}, closedRecord())
	m.width = 120
	m.height = 40
	out := m.View().Content
	if !strings.Contains(out, "Edit attendance") {
		t.Errorf("view should contain the attendance overlay\n%s", out)
	}
	if !strings.Contains(out, "ATTENDANCE") {
		t.Errorf("status bar should show the ATTENDANCE state label\n%s", out)
	}
}

func TestHelpOverlay_ListsAttendanceEdit(t *testing.T) {
	out := renderHelpOverlay(DefaultKeyMap(), 100, 40)
	if !strings.Contains(out, "edit attendance") {
		t.Errorf("help overlay should list the attendance edit key\n%s", out)
	}
}

func TestModel_AttendanceDayLoadedSortsByCheckIn(t *testing.T) {
	// Odoo returns hr.attendance newest-first by default; the pick list
	// must read top-down in chronological order.
	closed := closedRecord()
	open := openRecord()
	m := loadedAttendanceModel(t, &mockClient{}, open, closed)
	if m.att.sub != attPick {
		t.Fatalf("sub = %v, want attPick", m.att.sub)
	}
	if got := m.att.records[0].ID; got != closed.ID {
		t.Errorf("first record = #%d, want the earlier record #%d", got, closed.ID)
	}
	out := renderAttendanceOverlay(m.att, m.spinner, m.keys)
	if strings.Index(out, "#11") > strings.Index(out, "#12") {
		t.Errorf("pick list should show #11 (08:58) before #12 (18:00)\n%s", out)
	}
}
