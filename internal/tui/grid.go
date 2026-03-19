package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// GridRow represents a single project/task row in the weekly grid.
type GridRow struct {
	Key           string
	Label         string
	Company       string                   // company name from first entry
	Hours         [7]float64               // Mon=0 .. Sun=6
	Entries       [7][]odoo.TimesheetEntry // individual entries per day
	HintProjectID int64                    // from previous week, used when row has no entries
	HintTaskID    int64                    // from previous week, used when row has no entries
}

// ProjectTaskIDs scans all days in the row and returns the ProjectID and TaskID
// from the first non-empty entry found. Returns 0, 0 if the row has no entries.
func (r GridRow) ProjectTaskIDs() (projectID, taskID int64) {
	for d := 0; d < 7; d++ {
		if len(r.Entries[d]) > 0 {
			return r.Entries[d][0].ProjectID, r.Entries[d][0].TaskID
		}
	}
	return r.HintProjectID, r.HintTaskID
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

		key := gridRowKey(e.Company, e.ProjectID, e.TaskID, e.Project, e.Task)
		label := gridRowLabel(e.Company, e.Project, e.Task)

		idx, ok := rowIndex[key]
		if !ok {
			idx = len(g.Rows)
			rowIndex[key] = idx
			g.Rows = append(g.Rows, GridRow{Key: key, Label: label, Company: e.Company})
		}
		g.Rows[idx].Hours[dayOffset] += e.Hours
		g.Rows[idx].Entries[dayOffset] = append(g.Rows[idx].Entries[dayOffset], e)
	}

	sort.Slice(g.Rows, func(i, j int) bool {
		if g.Rows[i].Label == g.Rows[j].Label {
			return g.Rows[i].Key < g.Rows[j].Key
		}
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

// HintRow carries label and IDs from a previous week's entries to seed empty rows.
type HintRow struct {
	Key       string
	Label     string
	Company   string
	ProjectID int64
	TaskID    int64
}

// HintLabelsFromEntries extracts unique project/task labels with IDs from entries.
func HintLabelsFromEntries(entries []odoo.TimesheetEntry) []HintRow {
	seen := make(map[string]bool)
	var hints []HintRow
	for _, e := range entries {
		key := gridRowKey(e.Company, e.ProjectID, e.TaskID, e.Project, e.Task)
		if seen[key] {
			continue
		}
		seen[key] = true
		hints = append(hints, HintRow{
			Key:       key,
			Label:     gridRowLabel(e.Company, e.Project, e.Task),
			Company:   e.Company,
			ProjectID: e.ProjectID,
			TaskID:    e.TaskID,
		})
	}
	return hints
}

// BuildWeekGridWithHints works like BuildWeekGrid but pre-seeds empty rows from
// previous week hints for project/task combinations not present in current entries.
func BuildWeekGridWithHints(entries []odoo.TimesheetEntry, monday time.Time, hints []HintRow) WeekGrid {
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

		key := gridRowKey(e.Company, e.ProjectID, e.TaskID, e.Project, e.Task)
		label := gridRowLabel(e.Company, e.Project, e.Task)

		idx, ok := rowIndex[key]
		if !ok {
			idx = len(g.Rows)
			rowIndex[key] = idx
			g.Rows = append(g.Rows, GridRow{Key: key, Label: label, Company: e.Company})
		}
		g.Rows[idx].Hours[dayOffset] += e.Hours
		g.Rows[idx].Entries[dayOffset] = append(g.Rows[idx].Entries[dayOffset], e)
	}

	// Add hint rows for labels not already present.
	for _, h := range hints {
		hintKey := hintIdentity(h)
		if _, ok := rowIndex[hintKey]; ok {
			continue
		}
		idx := len(g.Rows)
		rowIndex[hintKey] = idx
		g.Rows = append(g.Rows, GridRow{
			Key:           hintKey,
			Label:         h.Label,
			Company:       h.Company,
			HintProjectID: h.ProjectID,
			HintTaskID:    h.TaskID,
		})
	}

	sort.Slice(g.Rows, func(i, j int) bool {
		if g.Rows[i].Label == g.Rows[j].Label {
			return g.Rows[i].Key < g.Rows[j].Key
		}
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

func gridRowKey(company string, projectID, taskID int64, project, task string) string {
	if projectID != 0 || taskID != 0 {
		return fmt.Sprintf("%s|%d|%d", company, projectID, taskID)
	}
	return fmt.Sprintf("%s|%s|%s", company, project, task)
}

func gridRowLabel(company, project, task string) string {
	label := project
	if task != "" {
		label += " / " + task
	}
	prefix := companyPrefix(company)
	if prefix == "" {
		return label
	}
	return fmt.Sprintf("[%s] %s", prefix, label)
}

func companyPrefix(company string) string {
	company = strings.TrimSpace(company)
	if company == "" {
		return ""
	}

	var runes []rune
	for _, r := range company {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			runes = append(runes, unicode.ToUpper(r))
			if len(runes) == 3 {
				return string(runes)
			}
		}
	}
	if len(runes) > 0 {
		return string(runes)
	}

	raw := []rune(company)
	if len(raw) > 3 {
		raw = raw[:3]
	}
	for i, r := range raw {
		raw[i] = unicode.ToUpper(r)
	}
	return string(raw)
}

func hintIdentity(h HintRow) string {
	if h.Key != "" {
		return h.Key
	}
	if h.ProjectID != 0 || h.TaskID != 0 {
		return gridRowKey(h.Company, h.ProjectID, h.TaskID, "", "")
	}
	return h.Label
}

// TodayColumn returns the column index (0=Mon .. 6=Sun) for the given time
// relative to the week starting at monday. Returns 0 if now falls outside the
// displayed week.
func TodayColumn(monday, now time.Time) int {
	// Truncate to date-only for comparison.
	monDate := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	offset := int(nowDate.Sub(monDate).Hours() / 24)
	if offset < 0 || offset > 6 {
		return 0
	}
	return offset
}

// ParseHours parses a duration string in either H:MM or decimal format.
// Returns an error for empty strings, negative values, zero, and invalid formats.
func ParseHours(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("hours must be a positive number (e.g. 1.5 or 1:30)")
	}

	var hours float64
	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		if parts[0] == "" || parts[1] == "" {
			return 0, fmt.Errorf("invalid time format %q: expected H:MM", s)
		}
		h, err := strconv.Atoi(parts[0])
		if err != nil || h < 0 {
			return 0, fmt.Errorf("invalid time format %q: expected H:MM", s)
		}
		m, err := strconv.Atoi(parts[1])
		if err != nil || m < 0 || m > 59 {
			return 0, fmt.Errorf("invalid time format %q: minutes must be 0-59", s)
		}
		hours = float64(h) + float64(m)/60.0
	} else {
		var err error
		hours, err = strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("hours must be a positive number (e.g. 1.5 or 1:30)")
		}
	}

	if hours <= 0 {
		return 0, fmt.Errorf("hours must be greater than zero")
	}
	return hours, nil
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
