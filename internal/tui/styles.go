package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/seletz/odoo-work-cli/internal/config"
)

var (
	headerStyle  = lipgloss.NewStyle().Bold(true)
	weekendStyle = lipgloss.NewStyle().Faint(true)
	totalsStyle  = lipgloss.NewStyle().Bold(true)
	cursorStyle  = lipgloss.NewStyle().Reverse(true)

	holidayStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5")) // magenta
	hoursLow     = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	hoursNormal  = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	hoursHigh    = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red

	detailHeaderStyle = lipgloss.NewStyle().Bold(true)
	detailBoxStyle    = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("4")). // blue
				Padding(1, 2)

	clockedInStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true) // green
	clockedOutStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))            // red

	detailHintStyle      = lipgloss.NewStyle().Faint(true)
	editLabelStyle       = lipgloss.NewStyle().Faint(true)
	editActiveLabelStyle = lipgloss.NewStyle().Bold(true)
	editErrorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red

	searchFilterWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true) // yellow
	searchSectionStyle  = lipgloss.NewStyle().Bold(true).Faint(true)
)

// hoursStyle returns the appropriate style based on daily total hours.
func hoursStyle(total float64, limits config.HoursLimits) lipgloss.Style {
	switch {
	case total > limits.DailyHigh:
		return hoursHigh
	case total >= limits.DailyLow:
		return hoursNormal
	default:
		return hoursLow
	}
}

// weekTotalStyle returns the appropriate style based on weekly total hours.
func weekTotalStyle(total float64, limits config.HoursLimits) lipgloss.Style {
	switch {
	case total > limits.WeeklyHigh:
		return hoursHigh
	case total >= limits.WeeklyLow:
		return hoursNormal
	default:
		return hoursLow
	}
}
