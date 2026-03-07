package tui

import (
	"strings"
	"testing"

	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

func TestRenderGrid_ContainsLabels(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 8.0},
		{Date: "2026-03-03", Project: "Beta", Task: "QA", Hours: 2.5},
	}
	g := BuildWeekGrid(entries, monday(2026, 3, 2))
	out := RenderGrid(g, 0, 0, 120, config.DefaultHoursLimits(), [7]string{})

	if !strings.Contains(out, "Acme / Dev") {
		t.Error("output should contain 'Acme / Dev'")
	}
	if !strings.Contains(out, "Beta / QA") {
		t.Error("output should contain 'Beta / QA'")
	}
	if !strings.Contains(out, "8:00") {
		t.Error("output should contain '8:00'")
	}
	if !strings.Contains(out, "2:30") {
		t.Error("output should contain '2:30'")
	}
}

func TestRenderGrid_EmptyGrid(t *testing.T) {
	g := BuildWeekGrid(nil, monday(2026, 3, 2))
	out := RenderGrid(g, 0, 0, 120, config.DefaultHoursLimits(), [7]string{})

	if !strings.Contains(out, "Project / Task") {
		t.Error("output should contain header")
	}
	if !strings.Contains(out, "Total") {
		t.Error("output should contain totals row")
	}
	// Header + separator + separator + totals = 4 lines
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Errorf("expected 4 lines for empty grid, got %d", len(lines))
	}
}

func TestRenderDetail_ShowsEntries(t *testing.T) {
	row := GridRow{
		Label: "Acme Corp / Backend Dev",
	}
	row.Entries[0] = []odoo.TimesheetEntry{
		{ID: 31097, Hours: 2.0, ValidatedStatus: "draft", Name: "Implemented user auth endpoint"},
		{ID: 31098, Hours: 1.5, ValidatedStatus: "validated", Name: "Code review PR #42"},
	}
	row.Hours[0] = 3.5

	mon := monday(2026, 3, 2)
	out := RenderDetail(row, 0, mon, 80)

	checks := []struct {
		substr string
		desc   string
	}{
		{"Acme Corp / Backend Dev", "project/task label"},
		{"Mon 02 Mar", "day name and date"},
		{"31097", "entry ID 31097"},
		{"31098", "entry ID 31098"},
		{"2:00", "formatted hours 2:00"},
		{"1:30", "formatted hours 1:30"},
		{"draft", "status 'draft'"},
		{"validated", "status 'validated'"},
		{"Implemented user auth endpoint", "entry description"},
		{"3:30", "total hours"},
		{"2 entries", "entry count"},
	}
	for _, c := range checks {
		if !strings.Contains(out, c.substr) {
			t.Errorf("output should contain %s (%q)", c.desc, c.substr)
		}
	}
}

func TestRenderDetail_EmptyCell(t *testing.T) {
	row := GridRow{Label: "Acme / Dev"}
	mon := monday(2026, 3, 2)
	out := RenderDetail(row, 0, mon, 80)

	if !strings.Contains(out, "No entries") {
		t.Error("output should indicate no entries")
	}
}

func TestRenderDetailOverlay_CentersBox(t *testing.T) {
	// Create a simple background.
	bgLines := make([]string, 20)
	for i := range bgLines {
		bgLines[i] = strings.Repeat(".", 60)
	}
	bg := strings.Join(bgLines, "\n")

	row := GridRow{Label: "Test"}
	row.Entries[0] = []odoo.TimesheetEntry{
		{ID: 1, Hours: 1.0, ValidatedStatus: "draft", Name: "test entry"},
	}
	row.Hours[0] = 1.0

	detail := RenderDetail(row, 0, monday(2026, 3, 2), 60)
	result := RenderDetailOverlay(bg, detail, 60, 20)

	// The overlay should contain the box border characters.
	if !strings.Contains(result, "╭") {
		t.Error("overlay should contain rounded border top-left")
	}
	if !strings.Contains(result, "╯") {
		t.Error("overlay should contain rounded border bottom-right")
	}
	// Background dots should still be visible outside the box.
	if !strings.Contains(result, "...") {
		t.Error("background should still be visible outside overlay")
	}
}

func TestRenderGrid_CorrectLineCount(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "A", Task: "T1", Hours: 1.0},
		{Date: "2026-03-02", Project: "B", Task: "T2", Hours: 2.0},
		{Date: "2026-03-02", Project: "C", Task: "T3", Hours: 3.0},
	}
	g := BuildWeekGrid(entries, monday(2026, 3, 2))
	out := RenderGrid(g, 0, 0, 120, config.DefaultHoursLimits(), [7]string{})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// header + sep + 3 data rows + sep + totals = 7
	if len(lines) != 7 {
		t.Errorf("expected 7 lines, got %d", len(lines))
	}
}
