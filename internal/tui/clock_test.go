package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// The TUI must take every "now" from its injected clock, so screenshots
// rendered for the docs are the same on every run (issue #99).

func TestWithNow_TodayColumnFollowsInjectedClock(t *testing.T) {
	mon := monday(2020, 3, 2)
	fixed := time.Date(2020, 3, 4, 10, 0, 0, 0, time.UTC) // Wednesday
	m := newTestModel(nil, nil)
	m.monday = MondayTime{Time: mon}
	m = m.WithNow(func() time.Time { return fixed })

	updated, _ := m.Update(timesheetsLoadedMsg{})
	got := updated.(Model)
	if got.cursor[1] != 2 {
		t.Fatalf("cursor column = %d, want 2 (Wednesday of the injected clock)", got.cursor[1])
	}
}

func TestRenderClockStatus_UsesNow(t *testing.T) {
	now := time.Date(2020, 3, 4, 10, 0, 0, 0, time.Local)
	checkIn := now.Add(-90 * time.Minute)
	status := &odoo.AttendanceStatus{ClockedIn: true, CheckIn: &checkIn}
	if got := renderClockStatus(status, now); !strings.Contains(got, "(1:30)") {
		t.Fatalf("renderClockStatus = %q, want elapsed (1:30) relative to now", got)
	}
}

func TestRenderHeaderBar_UsesNow(t *testing.T) {
	mon := monday(2020, 3, 2)
	now := time.Date(2020, 3, 4, 10, 0, 0, 0, time.Local)
	checkIn := now.Add(-45 * time.Minute)
	attendance := &odoo.AttendanceStatus{
		ClockedIn: true, CheckIn: &checkIn,
		Periods: []odoo.AttendanceRecord{{CheckIn: checkIn}},
	}
	out := RenderHeaderBar(mon, attendance, nil, false, spinner.New(), 160, now)
	if !strings.Contains(out, "(0:45)") || !strings.Contains(out, "Today 0:45") {
		t.Fatalf("header %q should count elapsed time up to the given now", out)
	}
}

func TestWithTick_ReplacesAttendanceTicker(t *testing.T) {
	var gotDelay time.Duration
	stub := func() tea.Msg { return nil }
	m := newTestModel(nil, nil).WithTick(func(d time.Duration, _ func(time.Time) tea.Msg) tea.Cmd {
		gotDelay = d
		return stub
	})
	checkIn := time.Now()
	_, cmd := m.Update(attendanceLoadedMsg{status: &odoo.AttendanceStatus{ClockedIn: true, CheckIn: &checkIn}})
	if cmd == nil {
		t.Fatal("expected the attendance ticker while clocked in")
	}
	if gotDelay != time.Minute {
		t.Fatalf("tick delay = %v, want 1m from the injected ticker", gotDelay)
	}
}
