package tui

import (
	tea "charm.land/bubbletea/v2"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// berlin returns the Europe/Berlin location, failing the test if the
// timezone database is unavailable.
func berlin(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("loading Europe/Berlin: %v", err)
	}
	return loc
}

func TestSummarizeWeekAttendance(t *testing.T) {
	loc := berlin(t)
	mon := time.Date(2026, 3, 9, 0, 0, 0, 0, loc) // Mon 09 Mar 2026 (CET)
	at := func(day, h, m int) time.Time { return time.Date(2026, 3, day, h, m, 0, 0, loc) }
	closed := func(in, out time.Time, worked float64) odoo.AttendanceRecord {
		// Odoo returns UTC; the summary must bucket by local day.
		o := out.UTC()
		return odoo.AttendanceRecord{CheckIn: in.UTC(), CheckOut: &o, WorkedHours: worked}
	}
	open := func(in time.Time) odoo.AttendanceRecord {
		return odoo.AttendanceRecord{CheckIn: in.UTC()}
	}
	now := at(12, 15, 0) // Thursday afternoon

	type dayWant struct {
		day       int
		in, out   string // formatted local HH:MM, "" = nil
		hours     float64
		running   bool
		overnight bool
		records   int
		future    bool
	}

	tests := []struct {
		name      string
		records   []odoo.AttendanceRecord
		days      []dayWant
		wantTotal float64
	}{
		{
			name:    "no records",
			records: nil,
			days: []dayWant{
				{day: 0}, {day: 3}, {day: 4, future: true}, {day: 6, future: true},
			},
		},
		{
			name: "single closed record uses worked_hours",
			records: []odoo.AttendanceRecord{
				closed(at(9, 8, 0), at(9, 17, 0), 8.0), // lunch already subtracted by Odoo
			},
			days:      []dayWant{{day: 0, in: "08:00", out: "17:00", hours: 8.0, records: 1}},
			wantTotal: 8.0,
		},
		{
			name: "several records on one day: earliest in, latest out, summed hours",
			records: []odoo.AttendanceRecord{
				closed(at(10, 13, 0), at(10, 17, 30), 4.5),
				closed(at(10, 7, 45), at(10, 12, 0), 4.25),
			},
			days:      []dayWant{{day: 1, in: "07:45", out: "17:30", hours: 8.75, records: 2}},
			wantTotal: 8.75,
		},
		{
			name: "early-morning local check-in is bucketed on the local day",
			records: []odoo.AttendanceRecord{
				// 00:30 CET on Wednesday = 23:30 UTC on Tuesday.
				closed(at(11, 0, 30), at(11, 2, 30), 2.0),
			},
			days:      []dayWant{{day: 2, in: "00:30", out: "02:30", hours: 2.0, records: 1}, {day: 1}},
			wantTotal: 2.0,
		},
		{
			name: "record past midnight counts on its check-in day and flags overnight",
			records: []odoo.AttendanceRecord{
				closed(at(10, 22, 0), at(11, 1, 30), 3.5),
			},
			days: []dayWant{
				{day: 1, in: "22:00", out: "01:30", hours: 3.5, overnight: true, records: 1},
				{day: 2},
			},
			wantTotal: 3.5,
		},
		{
			name: "running record counts elapsed time until now",
			records: []odoo.AttendanceRecord{
				open(at(12, 9, 0)),
			},
			days:      []dayWant{{day: 3, in: "09:00", hours: 6.0, running: true, records: 1}},
			wantTotal: 6.0,
		},
		{
			name: "open record from before the week is ignored per day",
			records: []odoo.AttendanceRecord{
				open(at(8, 23, 0)), // Sunday of the previous week
			},
			days:      []dayWant{{day: 0}},
			wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeWeekAttendance(tt.records, mon, now)
			if !got.Loaded {
				t.Fatal("summary should be marked loaded")
			}
			if diff := got.Total - tt.wantTotal; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("Total = %v, want %v", got.Total, tt.wantTotal)
			}
			for _, w := range tt.days {
				d := got.Days[w.day]
				fmtT := func(p *time.Time) string {
					if p == nil {
						return ""
					}
					return p.In(loc).Format("15:04")
				}
				if fmtT(d.CheckIn) != w.in {
					t.Errorf("day %d CheckIn = %q, want %q", w.day, fmtT(d.CheckIn), w.in)
				}
				if fmtT(d.CheckOut) != w.out {
					t.Errorf("day %d CheckOut = %q, want %q", w.day, fmtT(d.CheckOut), w.out)
				}
				if diff := d.Hours - w.hours; diff > 1e-9 || diff < -1e-9 {
					t.Errorf("day %d Hours = %v, want %v", w.day, d.Hours, w.hours)
				}
				if d.Running != w.running {
					t.Errorf("day %d Running = %v, want %v", w.day, d.Running, w.running)
				}
				if d.Overnight != w.overnight {
					t.Errorf("day %d Overnight = %v, want %v", w.day, d.Overnight, w.overnight)
				}
				if d.Records != w.records {
					t.Errorf("day %d Records = %d, want %d", w.day, d.Records, w.records)
				}
				if d.Future != w.future {
					t.Errorf("day %d Future = %v, want %v", w.day, d.Future, w.future)
				}
			}
		})
	}
}

func TestFormatSignedHours(t *testing.T) {
	tests := []struct {
		h    float64
		want string
	}{
		{0, "0:00"},
		{0.004, "0:00"}, // rounds below a minute to zero
		{-0.004, "0:00"},
		{1.5, "+1:30"},
		{-0.25, "-0:15"},
		{-8, "-8:00"},
		{1.9999, "+2:00"}, // no "1:60"
		{10.5, "+10:30"},
	}
	for _, tt := range tests {
		if got := formatSignedHours(tt.h); got != tt.want {
			t.Errorf("formatSignedHours(%v) = %q, want %q", tt.h, got, tt.want)
		}
	}
}

// renderAttendanceTestGrid renders a grid for the week of Mon 09 Mar 2026
// with the given bookings and attendance summary.
func renderAttendanceTestGrid(t *testing.T, entries []odoo.TimesheetEntry, att WeekAttendance, holidays [7]string) string {
	t.Helper()
	g := BuildWeekGrid(entries, monday(2026, 3, 9))
	return RenderGrid(g, 0, 0, 120, config.DefaultHoursLimits(), holidays, nil, -1, att)
}

// lineWith returns the first rendered line containing label, failing if none.
func lineWith(t *testing.T, out, label string) string {
	t.Helper()
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, label) {
			return l
		}
	}
	t.Fatalf("no line containing %q in:\n%s", label, out)
	return ""
}

// styledCell builds a regexp matching content rendered through style, with
// any amount of right-aligned padding (see #62: never assert byte-exact
// padded lipgloss output).
func styledCell(styleMarker, content string) *regexp.Regexp {
	return regexp.MustCompile(regexp.QuoteMeta(styleMarker) + `\s*` + regexp.QuoteMeta(content))
}

func TestRenderGrid_AttendanceRows(t *testing.T) {
	loc := time.Local
	at := func(day, h, m int) time.Time { return time.Date(2026, 3, day, h, m, 0, 0, loc) }
	out1 := at(9, 17, 0)
	out2 := at(11, 1, 30)
	running := at(12, 9, 0)

	att := WeekAttendance{Loaded: true}
	att.Days[0] = DayAttendance{CheckIn: ptrTime(at(9, 8, 0)), CheckOut: &out1, Hours: 8.0, Records: 1}
	att.Days[1] = DayAttendance{CheckIn: ptrTime(at(10, 22, 0)), CheckOut: &out2, Hours: 3.5, Overnight: true, Records: 1}
	att.Days[3] = DayAttendance{CheckIn: &running, Hours: 2.0, Running: true, Records: 1}
	att.Days[4] = DayAttendance{Future: true}
	att.Days[5] = DayAttendance{Future: true}
	att.Days[6] = DayAttendance{Future: true}
	att.Total = 13.5

	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-09", Project: "Acme", Task: "Dev", Hours: 7.5}, // diff +0:30
		{Date: "2026-03-10", Project: "Acme", Task: "Dev", Hours: 3.5}, // diff 0:00
		{Date: "2026-03-11", Project: "Acme", Task: "Dev", Hours: 2.0}, // no attendance: -2:00
	}
	out := renderAttendanceTestGrid(t, entries, att, [7]string{})

	in := lineWith(t, out, "Check-in")
	for _, want := range []string{"08:00", "22:00", "09:00"} {
		if !strings.Contains(in, want) {
			t.Errorf("check-in row %q should contain %q", in, want)
		}
	}

	co := lineWith(t, out, "Check-out")
	for _, want := range []string{"17:00", "01:30+", "now"} {
		if !strings.Contains(co, want) {
			t.Errorf("check-out row %q should contain %q", co, want)
		}
	}

	dur := lineWith(t, out, "Attendance")
	for _, want := range []string{"8:00", "3:30", "2:00", "13:30"} {
		if !strings.Contains(dur, want) {
			t.Errorf("attendance row %q should contain %q", dur, want)
		}
	}
	// Wednesday is a past workday without attendance → missing marker.
	missing := missingAttendanceStyle.Render("X")
	missingPrefix := missing[:strings.Index(missing, "X")]
	if !styledCell(missingPrefix, attendanceMissingMark).MatchString(dur) {
		t.Errorf("attendance row should mark Wednesday as missing with missingAttendanceStyle: %q", dur)
	}
	if n := strings.Count(dur, attendanceMissingMark); n != 1 {
		t.Errorf("missing marker count = %d, want 1 (future days and weekend are not missing): %q", n, dur)
	}

	diff := lineWith(t, out, "Diff")
	pos := diffPositiveStyle.Render("X")
	neg := diffNegativeStyle.Render("X")
	zero := diffZeroStyle.Render("X")
	prefix := func(s string) string { return s[:strings.Index(s, "X")] }
	if !styledCell(prefix(pos), "+0:30").MatchString(diff) {
		t.Errorf("Monday diff +0:30 should use diffPositiveStyle: %q", diff)
	}
	if !styledCell(prefix(zero), "0:00").MatchString(diff) {
		t.Errorf("Tuesday diff 0:00 should use diffZeroStyle: %q", diff)
	}
	if !styledCell(prefix(neg), "-2:00").MatchString(diff) {
		t.Errorf("Wednesday diff -2:00 should use diffNegativeStyle: %q", diff)
	}
	if !strings.Contains(diff, "+2:00") {
		t.Errorf("Thursday diff (attendance only) should be +2:00: %q", diff)
	}
	// Week: 13:30 attendance − 13:00 booked = +0:30.
	if !styledCell(prefix(pos), "+0:30").MatchString(diff[strings.LastIndex(diff, boxV):]) {
		t.Errorf("week diff column should be +0:30: %q", diff)
	}
}

func TestRenderGrid_AttendanceHolidayAndWeekendNotMissing(t *testing.T) {
	att := WeekAttendance{Loaded: true}
	var hols [7]string
	hols[2] = "Feiertag"
	out := renderAttendanceTestGrid(t, nil, att, hols)

	dur := lineWith(t, out, "Attendance")
	// Mon, Tue, Thu, Fri are missing; Wed is a holiday; Sat/Sun are weekend.
	if n := strings.Count(dur, attendanceMissingMark); n != 4 {
		t.Errorf("missing marker count = %d, want 4: %q", n, dur)
	}
}

func TestRenderGrid_NoAttendanceRowsUntilLoaded(t *testing.T) {
	out := renderAttendanceTestGrid(t, nil, WeekAttendance{}, [7]string{})
	for _, label := range []string{"Check-in", "Check-out", "Attendance", "Diff"} {
		if strings.Contains(out, label) {
			t.Errorf("unloaded attendance should not render %q row", label)
		}
	}
}

// Issue #78: the week's attendance is loaded in one request whose bounds are
// the displayed week's local midnights, so early-morning local check-ins on
// Monday are included and those of the next Monday are not.
func TestLoadAttendance_UsesLocalWeekBounds(t *testing.T) {
	loc := berlin(t)
	client := &mockClient{attendStatus: &odoo.AttendanceStatus{}}
	mon := MondayTime{Time: time.Date(2026, 3, 9, 0, 0, 0, 0, loc)}
	m := NewModel(client, mon, config.DefaultHoursLimits(), "Deutschland", nil, nil)

	msg := m.loadAttendance()().(attendanceLoadedMsg)

	if client.attendCalls != 1 {
		t.Errorf("ListAttendance calls = %d, want 1 (single request per week)", client.attendCalls)
	}
	if !client.attendWeekFrom.Equal(mon.Time) {
		t.Errorf("from = %v, want local Monday midnight %v", client.attendWeekFrom, mon.Time)
	}
	if want := mon.AddDate(0, 0, 7); !client.attendWeekTo.Equal(want) {
		t.Errorf("to = %v, want %v", client.attendWeekTo, want)
	}
	if !msg.weekStart.Equal(mon.Time) {
		t.Errorf("msg.weekStart = %v, want %v", msg.weekStart, mon.Time)
	}
}

func TestModel_AttendanceLoadedWeekGating(t *testing.T) {
	cur := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	rec := []odoo.AttendanceRecord{{ID: 1, CheckIn: cur.Add(9 * time.Hour)}}

	tests := []struct {
		name       string
		weekStart  time.Time
		wantLoaded bool
	}{
		{"current week is applied", cur, true},
		{"stale week from before navigation is dropped", cur.AddDate(0, 0, -7), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(nil, nil)
			status := &odoo.AttendanceStatus{}
			updated, _ := m.Update(attendanceLoadedMsg{status: status, week: rec, weekStart: tt.weekStart})
			um := updated.(Model)
			if um.attendance != status {
				t.Error("status describes today and must always be applied")
			}
			if um.weekAttendanceLoaded != tt.wantLoaded {
				t.Errorf("weekAttendanceLoaded = %v, want %v", um.weekAttendanceLoaded, tt.wantLoaded)
			}
		})
	}
}

func TestModel_WeekNavClearsWeekAttendance(t *testing.T) {
	m := newTestModel(nil, nil)
	m.state = stateGrid
	m.weekAttendance = []odoo.AttendanceRecord{{ID: 1}}
	m.weekAttendanceLoaded = true

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	um := updated.(Model)
	if um.weekAttendanceLoaded || um.weekAttendance != nil {
		t.Error("week navigation must clear the previous week's attendance until the reload arrives")
	}
}

func TestModel_ViewShowsAttendanceRows(t *testing.T) {
	m := newTestModel(nil, nil)
	m.state = stateGrid
	m.width, m.height = 140, 40
	updated, _ := m.Update(attendanceLoadedMsg{
		status:    &odoo.AttendanceStatus{},
		weekStart: m.monday.Time,
	})
	view := updated.(Model).View().Content
	for _, label := range []string{"Check-in", "Check-out", "Attendance", "Diff"} {
		if !strings.Contains(view, label) {
			t.Errorf("view should contain %q row", label)
		}
	}
}
