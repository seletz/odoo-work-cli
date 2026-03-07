package tui

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// GridRow represents a single project/task row in the weekly grid.
type GridRow struct {
	Label   string
	Hours   [7]float64                // Mon=0 .. Sun=6
	Entries [7][]odoo.TimesheetEntry  // individual entries per day
}

// WeekGrid holds the aggregated weekly timesheet data.
type WeekGrid struct {
	Monday    time.Time
	Rows      []GridRow
	DayTotals [7]float64
	WeekTotal float64
}

// BuildWeekGrid aggregates timesheet entries into a weekly grid.
// Entries are grouped by "Project / Task" and sorted alphabetically.
func BuildWeekGrid(entries []odoo.TimesheetEntry, monday time.Time) WeekGrid {
	g := WeekGrid{Monday: monday}

	rowIndex := make(map[string]int)
	for _, e := range entries {
		t, err := time.Parse("2006-01-02", e.Date)
		if err != nil {
			continue
		}
		dayOffset := int(t.Sub(monday).Hours() / 24)
		if dayOffset < 0 || dayOffset > 6 {
			continue
		}

		label := e.Project
		if e.Task != "" {
			label += " / " + e.Task
		}

		idx, ok := rowIndex[label]
		if !ok {
			idx = len(g.Rows)
			rowIndex[label] = idx
			g.Rows = append(g.Rows, GridRow{Label: label})
		}
		g.Rows[idx].Hours[dayOffset] += e.Hours
		g.Rows[idx].Entries[dayOffset] = append(g.Rows[idx].Entries[dayOffset], e)
	}

	sort.Slice(g.Rows, func(i, j int) bool {
		return g.Rows[i].Label < g.Rows[j].Label
	})

	for _, row := range g.Rows {
		for d := 0; d < 7; d++ {
			g.DayTotals[d] += row.Hours[d]
			g.WeekTotal += row.Hours[d]
		}
	}

	return g
}

// ParseWeekMonday parses an ISO week string (e.g. "2026-W10") and returns
// the Monday of that week. If week is empty, returns the Monday of the
// current week.
func ParseWeekMonday(week string) (time.Time, error) {
	var year, isoWeek int
	if week == "" {
		now := time.Now()
		year, isoWeek = now.ISOWeek()
	} else {
		_, err := fmt.Sscanf(week, "%d-W%d", &year, &isoWeek)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid week format %q (expected YYYY-Www): %w", week, err)
		}
	}
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.Local)
	weekday := jan4.Weekday()
	if weekday == 0 {
		weekday = 7
	}
	monday1 := jan4.AddDate(0, 0, -int(weekday-1))
	return monday1.AddDate(0, 0, (isoWeek-1)*7), nil
}

// FormatHours formats a duration in decimal hours as "H:MM".
// Returns an empty string for zero hours.
func FormatHours(h float64) string {
	if h == 0 {
		return ""
	}
	hours := int(h)
	minutes := int(math.Round((h - float64(hours)) * 60))
	return fmt.Sprintf("%d:%02d", hours, minutes)
}
