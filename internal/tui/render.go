package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

var dayNames = [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

// RenderGrid renders the weekly grid as a styled string.
// holidays is a [7]string with holiday names per day (empty = no holiday).
func RenderGrid(grid WeekGrid, cursorRow, cursorCol, width int, limits config.HoursLimits, holidays [7]string) string {
	minColWidth := 7
	minLabelWidth := 20
	maxLabelWidth := 40

	// Label gets up to maxLabelWidth; remaining space goes to day columns.
	labelWidth := maxLabelWidth
	if labelWidth > width/3 {
		labelWidth = width / 3
	}
	if labelWidth < minLabelWidth {
		labelWidth = minLabelWidth
	}
	remaining := width - labelWidth - 2 // 2 for gap
	colWidth := remaining / 8           // 7 days + total
	if colWidth < minColWidth {
		colWidth = minColWidth
	}

	isHoliday := func(d int) bool { return holidays[d] != "" }

	var b strings.Builder

	// Header row.
	header := fmt.Sprintf("%-*s  ", labelWidth, "Project / Task")
	for i, name := range dayNames {
		cell := fmt.Sprintf("%*s", colWidth, name)
		switch {
		case isHoliday(i):
			cell = holidayStyle.Render(cell)
		case i >= 5:
			cell = weekendStyle.Render(cell)
		default:
			cell = headerStyle.Render(cell)
		}
		header += cell
	}
	header += headerStyle.Render(fmt.Sprintf("%*s", colWidth, "Total"))
	b.WriteString(header)
	b.WriteString("\n")

	// Holiday names row (only if any holiday in this week).
	hasHoliday := false
	for _, h := range holidays {
		if h != "" {
			hasHoliday = true
			break
		}
	}
	if hasHoliday {
		hline := fmt.Sprintf("%-*s  ", labelWidth, "")
		for d := 0; d < 7; d++ {
			name := holidays[d]
			if len(name) > colWidth {
				name = name[:colWidth-1] + "…"
			}
			cell := fmt.Sprintf("%*s", colWidth, name)
			cell = holidayStyle.Render(cell)
			hline += cell
		}
		b.WriteString(hline)
		b.WriteString("\n")
	}

	// Separator.
	b.WriteString(strings.Repeat("─", labelWidth+2+8*colWidth))
	b.WriteString("\n")

	// Data rows.
	for ri, row := range grid.Rows {
		label := row.Label
		if len(label) > labelWidth {
			label = label[:labelWidth-1] + "…"
		}
		line := fmt.Sprintf("%-*s  ", labelWidth, label)

		var rowTotal float64
		for d := 0; d < 7; d++ {
			rowTotal += row.Hours[d]
			cell := fmt.Sprintf("%*s", colWidth, FormatHours(row.Hours[d]))
			switch {
			case ri == cursorRow && d == cursorCol:
				cell = cursorStyle.Render(cell)
			case isHoliday(d):
				cell = holidayStyle.Render(cell)
			case d >= 5:
				cell = weekendStyle.Render(cell)
			case row.Hours[d] > 0:
				cell = hoursStyle(grid.DayTotals[d], limits).Render(cell)
			}
			line += cell
		}
		totalCell := fmt.Sprintf("%*s", colWidth, FormatHours(rowTotal))
		line += totalsStyle.Render(totalCell)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Totals row.
	b.WriteString(strings.Repeat("─", labelWidth+2+8*colWidth))
	b.WriteString("\n")
	totalsLine := fmt.Sprintf("%-*s  ", labelWidth, "Total")
	for d := 0; d < 7; d++ {
		cell := fmt.Sprintf("%*s", colWidth, FormatHours(grid.DayTotals[d]))
		if isHoliday(d) {
			cell = holidayStyle.Bold(true).Render(cell)
		} else {
			cell = totalsStyle.Render(cell)
		}
		totalsLine += cell
	}
	weekCell := fmt.Sprintf("%*s", colWidth, FormatHours(grid.WeekTotal))
	totalsLine += weekTotalStyle(grid.WeekTotal, limits).Bold(true).Render(weekCell)
	b.WriteString(totalsLine)
	b.WriteString("\n")

	return b.String()
}

// RenderDetail renders a detail panel for a specific cell showing individual entries.
// Uses the bubbles table widget for the entries listing. The returned string is the
// inner content (no border); use RenderDetailOverlay to composite it as a centered
// overlay box on top of a background. detailCursor highlights the selected entry row.
func RenderDetail(row GridRow, col int, monday time.Time, detailCursor int, width int) string {
	// Inner content width is reduced by border (2) + padding (4).
	innerWidth := width - 8
	if innerWidth < 40 {
		innerWidth = 40
	}

	var b strings.Builder

	day := monday.AddDate(0, 0, col)
	header := fmt.Sprintf("%s — %s", row.Label, day.Format("Mon 02 Jan 2006"))
	b.WriteString(detailHeaderStyle.Render(header))
	b.WriteString("\n")

	entries := row.Entries[col]
	if len(entries) == 0 {
		b.WriteString("\nNo entries")
		return b.String()
	}

	// Build table using bubbles table widget.
	idWidth := 10
	hoursWidth := 6
	statusWidth := 10
	descWidth := innerWidth - idWidth - hoursWidth - statusWidth - 8 // padding between cols
	if descWidth < 10 {
		descWidth = 10
	}

	cols := []table.Column{
		{Title: "ID", Width: idWidth},
		{Title: "Hours", Width: hoursWidth},
		{Title: "Status", Width: statusWidth},
		{Title: "Description", Width: descWidth},
	}

	rows := make([]table.Row, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", e.ID),
			FormatHours(e.Hours),
			e.ValidatedStatus,
			e.Name,
		})
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithHeight(len(entries)+1),
		table.WithWidth(innerWidth),
		table.WithStyles(detailTableStyles()),
		table.WithFocused(true),
	)
	t.SetCursor(detailCursor)

	b.WriteString("\n")
	b.WriteString(t.View())
	b.WriteString("\n\n")
	total := fmt.Sprintf("Total: %s (%d entries)", FormatHours(row.Hours[col]), len(entries))
	b.WriteString(detailHeaderStyle.Render(total))
	b.WriteString("\n")
	b.WriteString(detailHintStyle.Render("j/k: select  e: edit  a: add  esc: back"))

	return b.String()
}

// detailTableStyles returns table styles for the detail overlay.
func detailTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Padding(0, 1)
	s.Cell = s.Cell.Padding(0, 1)
	return s
}

// RenderDetailOverlay composites a bordered detail box centered on top of bg.
func RenderDetailOverlay(bg, detail string, bgWidth, bgHeight int) string {
	box := detailBoxStyle.Render(detail)

	boxLines := strings.Split(box, "\n")
	bgLines := strings.Split(bg, "\n")

	// Pad bg to full height.
	for len(bgLines) < bgHeight {
		bgLines = append(bgLines, "")
	}

	boxH := len(boxLines)
	boxW := lipglossWidth(boxLines)

	// Center vertically and horizontally.
	startRow := (bgHeight - boxH) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (bgWidth - boxW) / 2
	if startCol < 0 {
		startCol = 0
	}

	for i, bline := range boxLines {
		row := startRow + i
		if row >= len(bgLines) {
			break
		}
		bgRunes := []rune(bgLines[row])
		// Pad bg line if needed.
		for len(bgRunes) < startCol+len([]rune(bline)) {
			bgRunes = append(bgRunes, ' ')
		}
		// Overwrite bg runes with box content.
		copy(bgRunes[startCol:], []rune(bline))
		bgLines[row] = string(bgRunes)
	}

	return strings.Join(bgLines, "\n")
}

// renderClockStatus returns a styled clock-in/out status string.
func renderClockStatus(attendance *odoo.AttendanceStatus) string {
	if attendance == nil {
		return ""
	}
	if !attendance.ClockedIn || attendance.CheckIn == nil {
		return clockedOutStyle.Render("🔴 Not clocked in")
	}
	elapsed := time.Since(*attendance.CheckIn)
	h := int(elapsed.Hours())
	m := int(elapsed.Minutes()) % 60
	text := fmt.Sprintf("🟢 Clocked in since %s (%d:%02d)",
		attendance.CheckIn.Local().Format("15:04"), h, m)
	return clockedInStyle.Render(text)
}

// renderEditForm renders the edit form overlay content.
func renderEditForm(row GridRow, day time.Time, hoursInput, descInput textinput.Model, focus int, editErr error, width int, isNew bool) string {
	_ = width // reserved for future layout adjustments
	var b strings.Builder

	verb := "Editing"
	if isNew {
		verb = "Adding"
	}
	header := fmt.Sprintf("%s: %s — %s", verb, row.Label, day.Format("Mon 02 Jan 2006"))
	b.WriteString(detailHeaderStyle.Render(header))
	b.WriteString("\n\n")

	hoursLabel := "  Hours:       "
	descLabel := "  Description: "
	if focus == 0 {
		hoursLabel = editActiveLabelStyle.Render(hoursLabel)
	} else {
		hoursLabel = editLabelStyle.Render(hoursLabel)
	}
	if focus == 1 {
		descLabel = editActiveLabelStyle.Render(descLabel)
	} else {
		descLabel = editLabelStyle.Render(descLabel)
	}

	b.WriteString(hoursLabel)
	b.WriteString(hoursInput.View())
	b.WriteString("\n")
	b.WriteString(descLabel)
	b.WriteString(descInput.View())
	b.WriteString("\n")

	if editErr != nil {
		b.WriteString("\n")
		b.WriteString(editErrorStyle.Render(fmt.Sprintf("  Error: %s", editErr)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(detailHintStyle.Render("Enter: save  Esc: cancel  Tab: next field"))

	return b.String()
}

// lipglossWidth returns the visual width of the widest line.
func lipglossWidth(lines []string) int {
	max := 0
	for _, l := range lines {
		w := len([]rune(l))
		if w > max {
			max = w
		}
	}
	return max
}
