package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/seletz/odoo-work-cli/internal/config"
)

// Color palette — cohesive set of named colors used throughout the TUI.
var (
	colorBlue    = lipgloss.Color("#5f87ff")
	colorCyan    = lipgloss.Color("#5fd7ff")
	colorGreen   = lipgloss.Color("#5fd75f")
	colorYellow  = lipgloss.Color("#d7d75f")
	colorRed     = lipgloss.Color("#ff5f5f")
	colorMagenta = lipgloss.Color("#d75fd7")
	colorGold    = lipgloss.Color("#d7af5f")
	colorFg      = lipgloss.Color("#d0d0d0")
	colorBarBg   = lipgloss.Color("#262626")
)

var (
	headerStyle  = lipgloss.NewStyle().Bold(true)
	weekendStyle = lipgloss.NewStyle().Faint(true)
	totalsStyle  = lipgloss.NewStyle().Bold(true)

	// Cursor: bold white on blue background instead of plain Reverse.
	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(colorBlue)

	holidayStyle = lipgloss.NewStyle().Foreground(colorMagenta)
	hoursLow     = lipgloss.NewStyle().Foreground(colorYellow)
	hoursNormal  = lipgloss.NewStyle().Foreground(colorGreen)
	hoursHigh    = lipgloss.NewStyle().Foreground(colorRed)

	// Hour cell background tints for data cells.
	hoursBgLow    = lipgloss.NewStyle().Foreground(colorYellow).Background(lipgloss.Color("#2a2a1a"))
	hoursBgNormal = lipgloss.NewStyle().Foreground(colorGreen).Background(lipgloss.Color("#1a2a1a"))
	hoursBgHigh   = lipgloss.NewStyle().Foreground(colorRed).Background(lipgloss.Color("#2a1a1a"))

	// Week number style — bold cyan.
	weekNumberStyle = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)

	// Header bar: full-width background bar.
	headerBarStyle = lipgloss.NewStyle().
			Background(colorBarBg).
			Foreground(colorFg).
			Bold(true).
			Padding(0, 1)

	// Today column header: underlined.
	todayHeaderStyle = lipgloss.NewStyle().Bold(true).
				Foreground(colorCyan).
				Underline(true)

	// Today column cells: subtle background tint.
	todayCellStyle = lipgloss.NewStyle().Background(lipgloss.Color("#1a2a2a"))

	// Totals row: bold with a subtle background.
	totalsRowStyle = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#262626"))

	// Week total in totals row: extra emphasis.
	weekTotalBoldStyle = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#303030"))

	// Grid separator: faint box-drawing lines.
	gridSepStyle = lipgloss.NewStyle().Faint(true)

	// Detail overlay: double border with blue foreground.
	detailHeaderStyle = lipgloss.NewStyle().Bold(true)
	detailBoxStyle    = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(colorBlue).
				Padding(1, 2)

	// Edit overlay: rounded border with green foreground.
	editBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			Padding(1, 2)

	// Search overlay: thick border with magenta foreground.
	searchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(colorMagenta).
			Padding(1, 2)

	// Help overlay: rounded border with gold foreground.
	helpBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGold).
			Padding(1, 2)

	clockedInStyle  = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	clockedOutStyle = lipgloss.NewStyle().Foreground(colorRed)

	detailHintStyle      = lipgloss.NewStyle().Faint(true)
	editLabelStyle       = lipgloss.NewStyle().Faint(true)
	editActiveLabelStyle = lipgloss.NewStyle().Bold(true)
	editErrorStyle       = lipgloss.NewStyle().Foreground(colorRed)

	searchFilterWarning = lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
	searchSectionStyle  = lipgloss.NewStyle().Bold(true).Faint(true)

	// Search result badges.
	searchProjectBadge = lipgloss.NewStyle().Foreground(colorBlue).Bold(true)
	searchTaskBadge    = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)

	// Status bar at bottom.
	statusBarStyle = lipgloss.NewStyle().
			Background(colorBarBg).
			Foreground(colorFg).
			Padding(0, 1)
)

// companyLabelStyle returns a style with the given lipgloss color as foreground.
func companyLabelStyle(color string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

// hoursBgStyle returns hour style with a faint background tint.
func hoursBgStyle(total float64, limits config.HoursLimits) lipgloss.Style {
	switch {
	case total > limits.DailyHigh:
		return hoursBgHigh
	case total >= limits.DailyLow:
		return hoursBgNormal
	default:
		return hoursBgLow
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
