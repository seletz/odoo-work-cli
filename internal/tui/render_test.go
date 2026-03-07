package tui

import (
	"strings"
	"testing"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

func TestRenderGrid_ContainsLabels(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 8.0},
		{Date: "2026-03-03", Project: "Beta", Task: "QA", Hours: 2.5},
	}
	g := BuildWeekGrid(entries, monday(2026, 3, 2))
	out := RenderGrid(g, 0, 0, 120)

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
	out := RenderGrid(g, 0, 0, 120)

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

func TestRenderGrid_CorrectLineCount(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "A", Task: "T1", Hours: 1.0},
		{Date: "2026-03-02", Project: "B", Task: "T2", Hours: 2.0},
		{Date: "2026-03-02", Project: "C", Task: "T3", Hours: 3.0},
	}
	g := BuildWeekGrid(entries, monday(2026, 3, 2))
	out := RenderGrid(g, 0, 0, 120)

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// header + sep + 3 data rows + sep + totals = 7
	if len(lines) != 7 {
		t.Errorf("expected 7 lines, got %d", len(lines))
	}
}
