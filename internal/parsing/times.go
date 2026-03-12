package parsing

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/tui"
)

// ParseDateRange returns a single-day date range for the given YYYY-MM-DD string.
func ParseDateRange(date string) (string, string, error) {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", "", fmt.Errorf("invalid date %q: expected YYYY-MM-DD", date)
	}
	s := d.Format("2006-01-02")
	return s, s, nil
}

// WeekDateRange returns the Monday and Sunday of the ISO week specified
// as "2006-W02" format, or the current week if empty.
func WeekDateRange(week string) (string, string, error) {
	monday, err := tui.ParseWeekMonday(week)
	if err != nil {
		return "", "", err
	}
	sunday := monday.AddDate(0, 0, 6)
	return monday.Format("2006-01-02"), sunday.Format("2006-01-02"), nil
}
