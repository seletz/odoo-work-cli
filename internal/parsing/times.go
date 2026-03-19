package parsing

import (
	"fmt"
	"time"
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
	monday, err := ParseWeekMonday(week)
	if err != nil {
		return "", "", err
	}
	sunday := monday.AddDate(0, 0, 6)
	return monday.Format("2006-01-02"), sunday.Format("2006-01-02"), nil
}

// ParseWeekMonday parses an ISO week string (e.g. "2026-W10") and returns
// the Monday of that week. If week is empty, returns the Monday of the
// current week.
func ParseWeekMonday(week string) (time.Time, error) {
	var year, isoWeek int
	if week == "" {
		now := time.Now()
		year, isoWeek = now.ISOWeek()
	} else {
		_, err := fmt.Sscanf(week, "%d-W%d", &year, &isoWeek)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid week format %q (expected YYYY-Www): %w", week, err)
		}
	}
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.Local)
	weekday := jan4.Weekday()
	if weekday == 0 {
		weekday = 7
	}
	monday1 := jan4.AddDate(0, 0, -int(weekday-1))
	return monday1.AddDate(0, 0, (isoWeek-1)*7), nil
}
