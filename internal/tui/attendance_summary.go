package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// attendanceMissingMark marks a past workday without any attendance record.
const attendanceMissingMark = "—"

// DayAttendance summarizes one day's attendance records for the weekly grid.
//
// A record belongs to the local day of its check-in, even when it runs past
// midnight (Odoo's own attendance reports group the same way, and the
// record's lunch-adjusted worked_hours cannot be split across days).
type DayAttendance struct {
	CheckIn   *time.Time // earliest check-in of the day, nil if none
	CheckOut  *time.Time // latest check-out of the day, nil if none or running
	Running   bool       // a record of this day is still open
	Overnight bool       // the latest check-out falls on a later day
	Hours     float64    // summed attendance hours (see SummarizeWeekAttendance)
	Records   int        // number of attendance records
	Future    bool       // the day lies after today
}

// WeekAttendance is the per-day attendance summary for the displayed week.
type WeekAttendance struct {
	Loaded bool // false until the week's attendance has been fetched
	Days   [7]DayAttendance
	Total  float64 // sum of Days[*].Hours
}

// SummarizeWeekAttendance buckets attendance records into the days of the
// week starting at monday (local midnight; its location defines the day
// boundaries).
//
// Closed records contribute Odoo's worked_hours, which already excludes the
// working calendar's lunch break and is therefore comparable to booked
// timesheet hours. Open records contribute the elapsed time until now, like
// the header totals. Records whose check-in lies outside the week (an open
// record from the previous week) are not assigned to any day.
//
// :param records: attendance records as returned by odoo.Client.ListAttendance
// :param monday: local midnight of the week's Monday
// :param now: current time, for running records and future-day detection
// :returns: the week's summary, marked as loaded
func SummarizeWeekAttendance(records []odoo.AttendanceRecord, monday, now time.Time) WeekAttendance {
	loc := monday.Location()
	att := WeekAttendance{Loaded: true}

	today := dayIndex(monday, now.In(loc))
	for d := range att.Days {
		att.Days[d].Future = d > today
	}

	for _, r := range records {
		in := r.CheckIn.In(loc)
		d := dayIndex(monday, in)
		if d < 0 || d > 6 {
			continue
		}
		day := &att.Days[d]
		day.Records++
		if day.CheckIn == nil || in.Before(*day.CheckIn) {
			day.CheckIn = &in
		}

		if r.CheckOut == nil {
			day.Running = true
			if elapsed := now.Sub(r.CheckIn).Hours(); elapsed > 0 {
				day.Hours += elapsed
			}
			continue
		}

		day.Hours += r.WorkedHours
		out := r.CheckOut.In(loc)
		if day.CheckOut == nil || out.After(*day.CheckOut) {
			day.CheckOut = &out
			day.Overnight = dayIndex(monday, out) > d
		}
	}

	for _, day := range att.Days {
		att.Total += day.Hours
	}
	return att
}

// dayIndex returns the calendar-day offset of t from monday, counted in
// monday's location. DST-safe: it compares calendar dates, not durations.
func dayIndex(monday, t time.Time) int {
	t = t.In(monday.Location())
	a := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return int(b.Sub(a).Hours() / 24)
}

// roundMinutes converts hours to whole minutes, rounding to nearest.
func roundMinutes(h float64) int {
	return int(math.Round(h * 60))
}

// formatMinutes formats a non-negative minute count as "H:MM".
func formatMinutes(min int) string {
	return fmt.Sprintf("%d:%02d", min/60, min%60)
}

// formatSignedHours formats an hour difference as "+H:MM" / "-H:MM", or
// "0:00" when it rounds to less than a minute.
func formatSignedHours(h float64) string {
	min := roundMinutes(h)
	switch {
	case min > 0:
		return "+" + formatMinutes(min)
	case min < 0:
		return "-" + formatMinutes(-min)
	default:
		return "0:00"
	}
}

// renderAttendanceRows renders the attendance block below the grid totals:
// check-in, check-out, attendance duration and the difference between
// attendance and booked hours, per day and for the week.
//
// :param att: the week's attendance summary
// :param grid: the week grid (for booked day and week totals)
// :param labelWidth: width of the label column
// :param colWidth: width of a day column including its separator
// :param holidays: holiday names per day; holidays are never "missing"
// :returns: the rendered lines, each terminated by a newline
func renderAttendanceRows(att WeekAttendance, grid WeekGrid, labelWidth, colWidth int, holidays [7]string) string {
	cellW := colWidth - 1
	sep := gridSepStyle.Render(boxV)
	pad := func(s string) string { return fmt.Sprintf("%*s", cellW, s) }

	var b strings.Builder
	b.WriteString(buildGridSeparator(labelWidth, colWidth, false))
	b.WriteString("\n")

	row := func(label string, cell func(d int) string, total string) {
		line := attendanceLabelStyle.Render(fmt.Sprintf("%-*s", labelWidth, label)) + "  "
		for d := 0; d < 7; d++ {
			line += sep + cell(d)
		}
		line += sep + total
		b.WriteString(line)
		b.WriteString("\n")
	}

	missing := func(d int) bool {
		day := att.Days[d]
		return day.Records == 0 && !day.Future && d < 5 && holidays[d] == ""
	}
	dayStyle := func(d int) func(string) string {
		if d >= 5 || holidays[d] != "" {
			return func(s string) string { return weekendStyle.Render(s) }
		}
		return func(s string) string { return s }
	}

	row("Check-in", func(d int) string {
		day := att.Days[d]
		if day.CheckIn == nil {
			return pad("")
		}
		return dayStyle(d)(pad(day.CheckIn.Format("15:04")))
	}, pad(""))

	row("Check-out", func(d int) string {
		day := att.Days[d]
		switch {
		case day.Running:
			return clockedInStyle.Render(pad("now"))
		case day.CheckOut == nil:
			return pad("")
		case day.Overnight:
			return dayStyle(d)(pad(day.CheckOut.Format("15:04") + "+"))
		default:
			return dayStyle(d)(pad(day.CheckOut.Format("15:04")))
		}
	}, pad(""))

	row("Attendance", func(d int) string {
		day := att.Days[d]
		switch {
		case day.Records > 0:
			return dayStyle(d)(pad(formatMinutes(roundMinutes(day.Hours))))
		case missing(d):
			return missingAttendanceStyle.Render(pad(attendanceMissingMark))
		default:
			return pad("")
		}
	}, totalsStyle.Render(pad(formatMinutes(roundMinutes(att.Total)))))

	diffCell := func(diff float64) string {
		text := formatSignedHours(diff)
		switch {
		case roundMinutes(diff) > 0:
			return diffPositiveStyle.Render(pad(text))
		case roundMinutes(diff) < 0:
			return diffNegativeStyle.Render(pad(text))
		default:
			return diffZeroStyle.Render(pad(text))
		}
	}
	row("Diff", func(d int) string {
		day := att.Days[d]
		booked := grid.DayTotals[d]
		if day.Records == 0 && booked == 0 {
			return pad("")
		}
		return diffCell(day.Hours - booked)
	}, diffCell(att.Total-grid.WeekTotal))

	return b.String()
}
