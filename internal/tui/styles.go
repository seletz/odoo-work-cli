package tui

import "charm.land/lipgloss/v2"

var (
	headerStyle  = lipgloss.NewStyle().Bold(true)
	weekendStyle = lipgloss.NewStyle().Faint(true)
	totalsStyle  = lipgloss.NewStyle().Bold(true)
	cursorStyle = lipgloss.NewStyle().Reverse(true)

	hoursLow    = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow, <6h
	hoursNormal = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green, 6-9h
	hoursHigh   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red, >9h
)

// hoursStyle returns the appropriate style based on daily total hours.
func hoursStyle(total float64) lipgloss.Style {
	switch {
	case total > 9:
		return hoursHigh
	case total >= 6:
		return hoursNormal
	default:
		return hoursLow
	}
}

// weekTotalStyle returns the appropriate style based on weekly total hours.
func weekTotalStyle(total float64) lipgloss.Style {
	switch {
	case total > 40:
		return hoursHigh
	case total >= 35:
		return hoursNormal
	default:
		return hoursLow
	}
}
