package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

func TestRenderGrid_ContainsLabels(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "Acme", Task: "Dev", Hours: 8.0},
		{Date: "2026-03-03", Project: "Beta", Task: "QA", Hours: 2.5},
	}
	g := BuildWeekGrid(entries, monday(2026, 3, 2))
	out := RenderGrid(g, 0, 0, 120, config.DefaultHoursLimits(), [7]string{}, nil, -1)

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

func TestRenderGrid_ShowsCompanyPrefix(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "Acme", Task: "Dev", Company: "Digital Team", Hours: 8.0},
	}

	g := BuildWeekGrid(entries, monday(2026, 3, 2))
	out := RenderGrid(g, 0, 0, 120, config.DefaultHoursLimits(), [7]string{}, nil, -1)

	if !strings.Contains(out, "[DIG] Acme / Dev") {
		t.Error("output should contain prefixed company label '[DIG] Acme / Dev'")
	}
}

func TestRenderGrid_EmptyGrid(t *testing.T) {
	g := BuildWeekGrid(nil, monday(2026, 3, 2))
	out := RenderGrid(g, 0, 0, 120, config.DefaultHoursLimits(), [7]string{}, nil, -1)

	if !strings.Contains(out, "Company Prefix / Project / Task") {
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
	out := RenderDetail(row, 0, mon, 0, 80, nil)

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
	out := RenderDetail(row, 0, mon, 0, 80, nil)

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

	detail := RenderDetail(row, 0, monday(2026, 3, 2), 0, 60, nil)
	result := RenderDetailOverlay(bg, detail, 60, 20, detailBoxStyle)

	// The overlay should contain the double border characters.
	if !strings.Contains(result, "╔") {
		t.Error("overlay should contain double border top-left")
	}
	if !strings.Contains(result, "╝") {
		t.Error("overlay should contain double border bottom-right")
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
	out := RenderGrid(g, 0, 0, 120, config.DefaultHoursLimits(), [7]string{}, nil, -1)

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// header + sep + 3 data rows + sep + totals = 7
	if len(lines) != 7 {
		t.Errorf("expected 7 lines, got %d", len(lines))
	}
}

func TestRenderGrid_WrapsLongLabels(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "Infrastruktur und Betrieb", Task: "Infrastructure Management", Company: "Digital", Hours: 8.0},
	}

	g := BuildWeekGrid(entries, monday(2026, 3, 2))
	out := RenderGrid(g, 0, 0, 90, config.DefaultHoursLimits(), [7]string{}, nil, -1)

	if strings.Contains(out, "…") {
		t.Fatal("output should wrap long labels instead of truncating with ellipsis")
	}
	if !strings.Contains(out, "[DIG] Infrastruktur") {
		t.Fatal("output should contain first wrapped label line")
	}
	if !strings.Contains(out, "Betrieb /") {
		t.Fatal("output should contain continuation line for wrapped label")
	}
	if !strings.Contains(out, "8:00") {
		t.Fatal("output should still contain hours for wrapped row")
	}
}

// stylePrefix returns the ANSI escape sequence a style emits before its
// content, so highlight assertions do not depend on exact column widths.
func stylePrefix(t *testing.T, s lipgloss.Style) string {
	t.Helper()
	const marker = "\x00"
	rendered := s.Render(marker)
	i := strings.Index(rendered, marker)
	if i < 0 {
		t.Fatalf("style output %q should contain marker", rendered)
	}
	return rendered[:i]
}

func TestRenderGrid_HighlightsWholeSelectedRow(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{Date: "2026-03-02", Project: "Alpha", Task: "Dev", Hours: 1.0},
		{Date: "2026-03-02", Project: "Beta", Task: "QA", Hours: 2.0},
	}

	g := BuildWeekGrid(entries, monday(2026, 3, 2))
	out := RenderGrid(g, 1, 0, 120, config.DefaultHoursLimits(), [7]string{}, nil, -1)

	var selectedLine string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Beta / QA") {
			selectedLine = line
			break
		}
	}
	if selectedLine == "" {
		t.Fatal("output should contain the selected row")
	}

	// Label is left-aligned, so the row cursor sequence immediately
	// precedes the label text.
	if !strings.Contains(selectedLine, stylePrefix(t, rowCursorStyle)+"Beta / QA") {
		t.Error("output should highlight the selected row label")
	}

	// Cells are right-aligned; match the style sequence followed by
	// padding and the hours, whatever the column width.
	selectedCell := regexp.MustCompile(regexp.QuoteMeta(stylePrefix(t, cursorStyle)) + ` *2:00`)
	if !selectedCell.MatchString(selectedLine) {
		t.Error("output should keep the selected cell highlighted")
	}

	selectedTotal := regexp.MustCompile(
		regexp.QuoteMeta(stylePrefix(t, rowCursorStyle)+stylePrefix(t, totalsStyle)) + ` *2:00`)
	if !selectedTotal.MatchString(selectedLine) {
		t.Error("output should highlight the selected row total")
	}
}

func TestRenderAttendanceSummary(t *testing.T) {
	now := time.Date(2026, 3, 10, 14, 0, 0, 0, time.UTC)
	checkIn := now.Add(-time.Hour)

	closed := func(day int, hours float64) odoo.AttendanceRecord {
		in := time.Date(2026, 3, day, 8, 0, 0, 0, time.UTC)
		out := in.Add(time.Duration(hours * float64(time.Hour)))
		return odoo.AttendanceRecord{CheckIn: in, CheckOut: &out, WorkedHours: hours}
	}

	tests := []struct {
		name       string
		attendance *odoo.AttendanceStatus
		week       []odoo.AttendanceRecord
		want       []string
		wantEmpty  bool
	}{
		{
			name:       "nil attendance renders nothing",
			attendance: nil,
			wantEmpty:  true,
		},
		{
			name: "closed periods and week records",
			attendance: &odoo.AttendanceStatus{
				Periods: []odoo.AttendanceRecord{closed(10, 6.5)},
			},
			week: []odoo.AttendanceRecord{closed(9, 30.0), closed(10, 2.25)},
			want: []string{"Today 6:30", "Week 32:15"},
		},
		{
			name: "open period counts elapsed time",
			attendance: &odoo.AttendanceStatus{
				ClockedIn: true,
				CheckIn:   &checkIn,
				Periods:   []odoo.AttendanceRecord{{CheckIn: checkIn}},
			},
			week: []odoo.AttendanceRecord{{CheckIn: checkIn}},
			want: []string{"Today 1:00", "Week 1:00"},
		},
		{
			name:       "no records shows zero totals",
			attendance: &odoo.AttendanceStatus{},
			week:       nil,
			want:       []string{"Today 0:00", "Week 0:00"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := renderAttendanceSummary(tt.attendance, tt.week, now)
			if tt.wantEmpty {
				if out != "" {
					t.Fatalf("expected empty string, got %q", out)
				}
				return
			}
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("output %q should contain %q", out, want)
				}
			}
		})
	}
}

func TestRenderHeaderBar_ShowsAttendanceTotals(t *testing.T) {
	mon := monday(2026, 3, 9)
	checkIn := time.Now().Add(-30 * time.Minute)
	attendance := &odoo.AttendanceStatus{
		ClockedIn: true,
		CheckIn:   &checkIn,
		Periods:   []odoo.AttendanceRecord{{CheckIn: checkIn}},
	}
	weekOut := checkIn.Add(-time.Hour)
	week := []odoo.AttendanceRecord{
		{CheckIn: weekOut.Add(-8 * time.Hour), CheckOut: &weekOut, WorkedHours: 8.0},
		{CheckIn: checkIn},
	}

	s := spinner.New()
	out := RenderHeaderBar(mon, attendance, week, false, s, 160)

	if !strings.Contains(out, "Today 0:30") {
		t.Errorf("header %q should contain today's attendance total", out)
	}
	if !strings.Contains(out, "Week 8:30") {
		t.Errorf("header %q should contain weekly attendance total", out)
	}
}
