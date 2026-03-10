package tui

import (
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

func monday(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func TestBuildWeekGrid_Empty(t *testing.T) {
	g := BuildWeekGrid(nil, monday(2026, 3, 2))
	if len(g.Rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(g.Rows))
	}
	if g.WeekTotal != 0 {
		t.Fatalf("expected 0 total, got %f", g.WeekTotal)
	}
	for i, v := range g.DayTotals {
		if v != 0 {
			t.Fatalf("day %d total should be 0, got %f", i, v)
		}
	}
}

func TestBuildWeekGrid_SingleEntry(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.5},
	}
	g := BuildWeekGrid(entries, monday(2026, 3, 2))

	if len(g.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(g.Rows))
	}
	if g.Rows[0].Label != "Acme / Dev" {
		t.Fatalf("expected label 'Acme / Dev', got %q", g.Rows[0].Label)
	}
	if g.Rows[0].Hours[0] != 2.5 {
		t.Fatalf("expected Mon=2.5, got %f", g.Rows[0].Hours[0])
	}
	if g.DayTotals[0] != 2.5 {
		t.Fatalf("expected day total 2.5, got %f", g.DayTotals[0])
	}
	if g.WeekTotal != 2.5 {
		t.Fatalf("expected week total 2.5, got %f", g.WeekTotal)
	}
}

func TestBuildWeekGrid_SameProjectTaskDay(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-03", Project: "Acme", Task: "Dev", Hours: 1.0},
		{Date: "2026-03-03", Project: "Acme", Task: "Dev", Hours: 2.0},
	}
	g := BuildWeekGrid(entries, monday(2026, 3, 2))

	if len(g.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(g.Rows))
	}
	// Tuesday = index 1
	if g.Rows[0].Hours[1] != 3.0 {
		t.Fatalf("expected Tue=3.0, got %f", g.Rows[0].Hours[1])
	}
	if g.WeekTotal != 3.0 {
		t.Fatalf("expected week total 3.0, got %f", g.WeekTotal)
	}
}

func TestBuildWeekGrid_MultipleProjects(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "Beta", Task: "QA", Hours: 1.0},
		{Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 2.0},
	}
	g := BuildWeekGrid(entries, monday(2026, 3, 2))

	if len(g.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(g.Rows))
	}
	// Rows should be sorted alphabetically
	if g.Rows[0].Label != "Acme / Dev" {
		t.Fatalf("expected first row 'Acme / Dev', got %q", g.Rows[0].Label)
	}
	if g.Rows[1].Label != "Beta / QA" {
		t.Fatalf("expected second row 'Beta / QA', got %q", g.Rows[1].Label)
	}
}

func TestBuildWeekGrid_EmptyTask(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "Acme", Task: "", Hours: 1.5},
	}
	g := BuildWeekGrid(entries, monday(2026, 3, 2))

	if g.Rows[0].Label != "Acme" {
		t.Fatalf("expected label 'Acme', got %q", g.Rows[0].Label)
	}
}

func TestBuildWeekGrid_WeekendEntries(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-07", Project: "Acme", Task: "Dev", Hours: 3.0}, // Saturday
		{Date: "2026-03-08", Project: "Acme", Task: "Dev", Hours: 1.0}, // Sunday
	}
	g := BuildWeekGrid(entries, monday(2026, 3, 2))

	if g.Rows[0].Hours[5] != 3.0 {
		t.Fatalf("expected Sat=3.0, got %f", g.Rows[0].Hours[5])
	}
	if g.Rows[0].Hours[6] != 1.0 {
		t.Fatalf("expected Sun=1.0, got %f", g.Rows[0].Hours[6])
	}
}

func TestBuildWeekGrid_PreservesEntries(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 100, Date: "2026-03-03", Project: "Acme", Task: "Dev", Hours: 1.0, Name: "auth endpoint", ValidatedStatus: "draft"},
		{ID: 101, Date: "2026-03-03", Project: "Acme", Task: "Dev", Hours: 2.0, Name: "code review", ValidatedStatus: "validated"},
		{ID: 102, Date: "2026-03-04", Project: "Acme", Task: "Dev", Hours: 3.0, Name: "bugfix", ValidatedStatus: "draft"},
	}
	g := BuildWeekGrid(entries, monday(2026, 3, 2))

	if len(g.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(g.Rows))
	}
	row := g.Rows[0]

	// Tuesday (index 1) should have 2 entries.
	if len(row.Entries[1]) != 2 {
		t.Fatalf("expected 2 entries on Tue, got %d", len(row.Entries[1]))
	}
	if row.Entries[1][0].ID != 100 {
		t.Errorf("expected first entry ID 100, got %d", row.Entries[1][0].ID)
	}
	if row.Entries[1][1].ID != 101 {
		t.Errorf("expected second entry ID 101, got %d", row.Entries[1][1].ID)
	}

	// Wednesday (index 2) should have 1 entry.
	if len(row.Entries[2]) != 1 {
		t.Fatalf("expected 1 entry on Wed, got %d", len(row.Entries[2]))
	}
	if row.Entries[2][0].Name != "bugfix" {
		t.Errorf("expected entry name 'bugfix', got %q", row.Entries[2][0].Name)
	}

	// Monday (index 0) should have no entries.
	if len(row.Entries[0]) != 0 {
		t.Errorf("expected 0 entries on Mon, got %d", len(row.Entries[0]))
	}
}

func TestHintLabelsFromEntries(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-02-23", Project: "Acme", Task: "Dev", ProjectID: 10, TaskID: 20, Hours: 2.0},
		{Date: "2026-02-24", Project: "Acme", Task: "Dev", ProjectID: 10, TaskID: 20, Hours: 1.0},
		{Date: "2026-02-25", Project: "Beta", Task: "QA", ProjectID: 30, TaskID: 40, Hours: 3.0},
	}
	hints := HintLabelsFromEntries(entries)

	if len(hints) != 2 {
		t.Fatalf("expected 2 hints, got %d", len(hints))
	}

	// Should be deduplicated by label.
	labels := map[string]bool{}
	for _, h := range hints {
		labels[h.Label] = true
	}
	if !labels["Acme / Dev"] {
		t.Error("expected hint for 'Acme / Dev'")
	}
	if !labels["Beta / QA"] {
		t.Error("expected hint for 'Beta / QA'")
	}

	// Check IDs are preserved.
	for _, h := range hints {
		if h.Label == "Acme / Dev" {
			if h.ProjectID != 10 || h.TaskID != 20 {
				t.Errorf("Acme / Dev: expected IDs 10/20, got %d/%d", h.ProjectID, h.TaskID)
			}
		}
		if h.Label == "Beta / QA" {
			if h.ProjectID != 30 || h.TaskID != 40 {
				t.Errorf("Beta / QA: expected IDs 30/40, got %d/%d", h.ProjectID, h.TaskID)
			}
		}
	}
}

func TestHintLabelsFromEntries_EmptyTask(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-02-23", Project: "Acme", Task: "", ProjectID: 10, TaskID: 0, Hours: 1.0},
	}
	hints := HintLabelsFromEntries(entries)
	if len(hints) != 1 {
		t.Fatalf("expected 1 hint, got %d", len(hints))
	}
	if hints[0].Label != "Acme" {
		t.Fatalf("expected label 'Acme', got %q", hints[0].Label)
	}
}

func TestBuildWeekGridWithHints_EmptyWeek(t *testing.T) {
	hints := []HintRow{
		{Label: "Acme / Dev", ProjectID: 10, TaskID: 20},
		{Label: "Beta / QA", ProjectID: 30, TaskID: 40},
	}
	g := BuildWeekGridWithHints(nil, monday(2026, 3, 2), hints)

	if len(g.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(g.Rows))
	}
	// Rows should be sorted.
	if g.Rows[0].Label != "Acme / Dev" {
		t.Fatalf("expected first row 'Acme / Dev', got %q", g.Rows[0].Label)
	}
	if g.Rows[1].Label != "Beta / QA" {
		t.Fatalf("expected second row 'Beta / QA', got %q", g.Rows[1].Label)
	}
	// Hours should be zero.
	for d := 0; d < 7; d++ {
		if g.Rows[0].Hours[d] != 0 {
			t.Errorf("expected 0 hours on day %d, got %f", d, g.Rows[0].Hours[d])
		}
	}
	// HintProjectID/HintTaskID should be set.
	pid, tid := g.Rows[0].ProjectTaskIDs()
	if pid != 10 || tid != 20 {
		t.Fatalf("expected IDs 10/20 from hint, got %d/%d", pid, tid)
	}
	pid, tid = g.Rows[1].ProjectTaskIDs()
	if pid != 30 || tid != 40 {
		t.Fatalf("expected IDs 30/40 from hint, got %d/%d", pid, tid)
	}
	// WeekTotal and DayTotals should be zero.
	if g.WeekTotal != 0 {
		t.Fatalf("expected 0 week total, got %f", g.WeekTotal)
	}
}

func TestBuildWeekGridWithHints_DeduplicatesExisting(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "Acme", Task: "Dev", ProjectID: 10, TaskID: 20, Hours: 2.5},
	}
	hints := []HintRow{
		{Label: "Acme / Dev", ProjectID: 10, TaskID: 20},
	}
	g := BuildWeekGridWithHints(entries, monday(2026, 3, 2), hints)

	if len(g.Rows) != 1 {
		t.Fatalf("expected 1 row (deduplicated), got %d", len(g.Rows))
	}
	if g.Rows[0].Hours[0] != 2.5 {
		t.Fatalf("expected Mon=2.5, got %f", g.Rows[0].Hours[0])
	}
}

func TestBuildWeekGridWithHints_MixedRows(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "Acme", Task: "Dev", ProjectID: 10, TaskID: 20, Hours: 2.0},
	}
	hints := []HintRow{
		{Label: "Acme / Dev", ProjectID: 10, TaskID: 20}, // overlaps
		{Label: "Beta / QA", ProjectID: 30, TaskID: 40},  // new
	}
	g := BuildWeekGridWithHints(entries, monday(2026, 3, 2), hints)

	if len(g.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(g.Rows))
	}
	// Acme should have hours from entries.
	if g.Rows[0].Label != "Acme / Dev" {
		t.Fatalf("expected first row 'Acme / Dev', got %q", g.Rows[0].Label)
	}
	if g.Rows[0].Hours[0] != 2.0 {
		t.Fatalf("expected Mon=2.0, got %f", g.Rows[0].Hours[0])
	}
	// Beta should be empty hint row with IDs.
	if g.Rows[1].Label != "Beta / QA" {
		t.Fatalf("expected second row 'Beta / QA', got %q", g.Rows[1].Label)
	}
	pid, tid := g.Rows[1].ProjectTaskIDs()
	if pid != 30 || tid != 40 {
		t.Fatalf("expected IDs 30/40 from hint, got %d/%d", pid, tid)
	}
}

func TestGridRow_ProjectTaskIDs_FallbackToHint(t *testing.T) {
	row := GridRow{
		Label:         "Acme / Dev",
		HintProjectID: 10,
		HintTaskID:    20,
	}
	pid, tid := row.ProjectTaskIDs()
	if pid != 10 || tid != 20 {
		t.Fatalf("expected hint IDs 10/20, got %d/%d", pid, tid)
	}
}

func TestGridRow_ProjectTaskIDs_EntriesOverrideHint(t *testing.T) {
	row := GridRow{
		Label:         "Acme / Dev",
		HintProjectID: 99,
		HintTaskID:    99,
		Entries: [7][]odoo.TimesheetEntry{
			{{ID: 1, ProjectID: 10, TaskID: 20}},
			{}, {}, {}, {}, {}, {},
		},
	}
	pid, tid := row.ProjectTaskIDs()
	if pid != 10 || tid != 20 {
		t.Fatalf("expected entry IDs 10/20, got %d/%d", pid, tid)
	}
}

func TestParseHours(t *testing.T) {
	tests := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		// H:MM format
		{"2:30", 2.5, false},
		{"1:00", 1.0, false},
		{"0:45", 0.75, false},
		{"8:15", 8.25, false},
		// decimal format
		{"2.5", 2.5, false},
		{"0.75", 0.75, false},
		{"1", 1.0, false},
		// error cases
		{"", 0, true},
		{"abc", 0, true},
		{"-1", 0, true},
		{"-1:30", 0, true},
		{"2:60", 0, true},
		{":30", 0, true},
		{"2:", 0, true},
		{"0", 0, true},
		{"0:00", 0, true},
		{"0.0", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseHours(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseHours(%q) = %v, want error", tt.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseHours(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseHours(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFormatHours(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, ""},
		{1.0, "1:00"},
		{1.5, "1:30"},
		{8.25, "8:15"},
		{0.75, "0:45"},
	}
	for _, tt := range tests {
		got := FormatHours(tt.input)
		if got != tt.want {
			t.Errorf("FormatHours(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
