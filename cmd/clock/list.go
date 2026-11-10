package clock

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/seletz/odoo-work-cli/internal/parsing"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/spf13/cobra"
)

const (
	dayTimeLayout = "2006-01-02 15:04"
	clockLayout   = "15:04"
)

// resolveListRange turns the --date/--week flags into the local-time range
// [from, to) to list. With neither flag the range is today. A week starts
// at local Monday midnight and spans seven calendar days (DST-safe).
func resolveListRange(dateStr, weekStr string, now time.Time) (time.Time, time.Time, error) {
	if dateStr != "" && weekStr != "" {
		return time.Time{}, time.Time{}, apperr.New(apperr.KindValidation, "--date and --week are mutually exclusive")
	}
	if weekStr != "" {
		monday, err := parsing.ParseWeekMonday(weekStr)
		if err != nil {
			return time.Time{}, time.Time{}, apperr.Wrap(apperr.KindValidation, err)
		}
		return monday, monday.AddDate(0, 0, 7), nil
	}
	day, err := parsing.ParseDay(dateStr, now)
	if err != nil {
		return time.Time{}, time.Time{}, apperr.Wrap(apperr.KindValidation, err)
	}
	return day, day.AddDate(0, 0, 1), nil
}

// formatCheckOut renders a check-out time. A check-out on a later calendar
// day than the check-in (an overnight record) carries its date.
func formatCheckOut(rec *odoo.AttendanceRecord) string {
	if rec.CheckOut == nil {
		return "--:--"
	}
	in, out := rec.CheckIn.Local(), rec.CheckOut.Local()
	if in.YearDay() != out.YearDay() || in.Year() != out.Year() {
		return out.Format(dayTimeLayout)
	}
	return out.Format(clockLayout)
}

// duration returns the worked time of a record; for a running record the
// time elapsed until now.
func duration(rec *odoo.AttendanceRecord, now time.Time) float64 {
	if rec.CheckOut == nil {
		return now.Sub(rec.CheckIn).Hours()
	}
	return rec.WorkedHours
}

// sortByCheckIn orders records chronologically; Odoo returns them newest
// first.
func sortByCheckIn(records []odoo.AttendanceRecord) {
	sort.SliceStable(records, func(i, j int) bool { return records[i].CheckIn.Before(records[j].CheckIn) })
}

// writeListTable prints attendance records as a table with a total.
func writeListTable(w io.Writer, records []odoo.AttendanceRecord, now time.Time) error {
	if len(records) == 0 {
		_, err := fmt.Fprintln(w, "No attendance records.")
		return err
	}
	_, _ = fmt.Fprintf(w, "%-7s %-16s %-16s %s\n", "ID", "Check In", "Check Out", "Duration")
	_, _ = fmt.Fprintf(w, "%-7s %-16s %-16s %s\n", "-------", "----------------", "----------------", "--------")
	var total float64
	for i := range records {
		r := &records[i]
		d := duration(r, now)
		total += d
		dur := tui.FormatHours(d)
		if r.CheckOut == nil {
			dur += " (running)"
		}
		_, _ = fmt.Fprintf(w, "%-7d %-16s %-16s %s\n", r.ID, r.CheckIn.Local().Format(dayTimeLayout), formatCheckOut(r), dur)
	}
	_, err := fmt.Fprintf(w, "\nTotal: %s\n", tui.FormatHours(total))
	return err
}

func listCMD(deps *app.Deps) *cobra.Command {
	var dateStr, weekStr string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List attendance records of a day or week",
		Long: `List the attendance records of a day (--date, default today) or an ISO
week (--week YYYY-Www) with ID, check-in, check-out and duration.

Records are attributed to the day of their check-in. A record that crosses
midnight is listed on its check-in day and shows the check-out date. Use it
before 'clock edit' to see what is already recorded.

--date and --week together are a validation error.`,
		Example: `  # A week, with the total
  odoo-work-cli clock list --week 2026-W14

  # One day as JSON
  odoo-work-cli clock list --date 2026-04-08 --json`,
		Annotations: map[string]string{
			manual.Synopsis: "odoo-work-cli clock list [--date YYYY-MM-DD | --week YYYY-Www] [--json | --format FORMAT]",
			manual.Output: `The columns ID, Check In, Check Out (with the date if it is on a later
day, --:-- while running) and Duration (H:MM, "(running)" for a running
record), then the total; or "No attendance records."

With --json an array of attendance records, oldest first, [] if there are
none.`,
			manual.Schemas: "attendance",
			manual.SeeAlso: "odoo-work-cli clock edit, odoo-work-cli clock status",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}
			now := time.Now()
			from, to, err := resolveListRange(dateStr, weekStr, now)
			if err != nil {
				return err
			}
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}
			records, err := client.ListAttendance(from.UTC(), to.UTC())
			if err != nil {
				return err
			}
			sortByCheckIn(records)
			w := cmd.OutOrStdout()
			if format.IsMachine() {
				items := make([]output.Attendance, 0, len(records))
				for _, r := range records {
					items = append(items, output.NewAttendance(r))
				}
				return output.WriteList(w, format, items)
			}
			return writeListTable(w, records, now)
		},
	}

	cmd.Flags().StringVar(&dateStr, "date", "", "day to list, `YYYY-MM-DD`")
	cmd.Flags().StringVar(&weekStr, "week", "", "ISO week to list, `YYYY-Www`")
	manual.SetFlagDefault(cmd.Flags(), "date", "today")
	return cmd
}
