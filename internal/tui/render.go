package tui

import (
	"fmt"
	"strings"

	"github.com/seletz/odoo-work-cli/internal/config"
)

var dayNames = [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

// RenderGrid renders the weekly grid as a styled string.
// holidays is a [7]string with holiday names per day (empty = no holiday).
func RenderGrid(grid WeekGrid, cursorRow, cursorCol, width int, limits config.HoursLimits, holidays [7]string) string {
	colWidth := 7
	minLabelWidth := 10

	// Expand label column to fill available width.
	fixedCols := 2 + 8*colWidth // gap + 7 day cols + total col
	labelWidth := width - fixedCols
	if labelWidth < minLabelWidth {
		labelWidth = minLabelWidth
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
