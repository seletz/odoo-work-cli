package tui

import (
	"fmt"
	"strings"
)

var dayNames = [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

// RenderGrid renders the weekly grid as a styled string.
func RenderGrid(grid WeekGrid, cursorRow, cursorCol, width int) string {
	labelWidth := 30
	colWidth := 7

	// Clamp label width so grid fits in terminal.
	gridWidth := labelWidth + 2 + 7*colWidth + colWidth // +colWidth for total col
	if width > 0 && gridWidth > width {
		labelWidth = width - 2 - 8*colWidth
		if labelWidth < 10 {
			labelWidth = 10
		}
	}

	var b strings.Builder

	// Header row.
	header := fmt.Sprintf("%-*s  ", labelWidth, "Project / Task")
	for i, name := range dayNames {
		cell := fmt.Sprintf("%*s", colWidth, name)
		if i >= 5 {
			cell = weekendStyle.Render(cell)
		} else {
			cell = headerStyle.Render(cell)
		}
		header += cell
	}
	header += headerStyle.Render(fmt.Sprintf("%*s", colWidth, "Total"))
	b.WriteString(header)
	b.WriteString("\n")

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
			if ri == cursorRow && d == cursorCol {
				cell = cursorStyle.Render(cell)
			} else if d >= 5 {
				cell = weekendStyle.Render(cell)
			} else if row.Hours[d] > 0 {
				cell = hoursStyle(grid.DayTotals[d]).Render(cell)
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
		cell = totalsStyle.Render(cell)
		totalsLine += cell
	}
	totalsLine += totalsStyle.Render(fmt.Sprintf("%*s", colWidth, FormatHours(grid.WeekTotal)))
	b.WriteString(totalsLine)
	b.WriteString("\n")

	return b.String()
}
