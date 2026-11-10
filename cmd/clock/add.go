package clock

import (
	"fmt"
	"strings"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

// resolveAddTimes parses and validates the clock add flags against now.
// Both --in and --out are required; parsing, the future check and the
// check-out-after-check-in rule are those of clock edit.
//
// :param dateStr: --date value (YYYY-MM-DD, empty for today)
// :param inStr: --in value
// :param outStr: --out value
// :param now: the current time
// :returns: the resolved day and times, or a validation error
func resolveAddTimes(dateStr, inStr, outStr string, now time.Time) (*editTimes, error) {
	if inStr == "" || outStr == "" {
		return nil, apperr.New(apperr.KindValidation, "clock add needs both --in and --out")
	}
	return resolveEditTimes(dateStr, inStr, outStr, now)
}

// findOverlaps returns the records that overlap the half-open span
// [in, out). A running record (no check-out) extends indefinitely. Records
// that only touch the span at an edge (12:00 out / 12:00 in) do not
// overlap.
//
// :param records: candidate attendance records
// :param in: check-in of the new record
// :param out: check-out of the new record
// :returns: the overlapping records, in input order
func findOverlaps(records []odoo.AttendanceRecord, in, out time.Time) []odoo.AttendanceRecord {
	var overlaps []odoo.AttendanceRecord
	for _, r := range records {
		if !r.CheckIn.Before(out) {
			continue
		}
		if r.CheckOut != nil && !r.CheckOut.After(in) {
			continue
		}
		overlaps = append(overlaps, r)
	}
	return overlaps
}

// formatSpan renders a check-in/check-out span; a check-out on a later day
// carries its date (as in clock list).
func formatSpan(in, out time.Time) string {
	return in.Local().Format(dayTimeLayout) + " - " + formatCheckOut(&odoo.AttendanceRecord{CheckIn: in, CheckOut: &out})
}

// overlapConflict is the error for clock add when the new span overlaps
// existing records. It names the earliest overlapping record and lists all
// of them oldest first.
func overlapConflict(in, out time.Time, overlaps []odoo.AttendanceRecord) error {
	sortByCheckIn(overlaps)
	var lines []string
	for i := range overlaps {
		lines = append(lines, "  "+formatRecord(&overlaps[i]))
	}
	return apperr.New(apperr.KindConflict,
		"%s overlaps attendance record #%d; nothing was changed. Overlapping records:\n%s",
		formatSpan(in, out), overlaps[0].ID, strings.Join(lines, "\n"))
}

// overlapCandidates fetches every record that can touch [in, out): those
// checked in from the previous day up to out (covers a record from the day
// before that runs past midnight) plus running records from any earlier day,
// which ListAttendance always includes.
func overlapCandidates(client odoo.Client, day, out time.Time) ([]odoo.AttendanceRecord, error) {
	to := day.AddDate(0, 0, 1)
	if out.After(to) {
		to = out
	}
	return client.ListAttendance(day.AddDate(0, 0, -1).UTC(), to.UTC())
}

func addCMD(deps *app.Deps) *cobra.Command {
	var dateStr, inStr, outStr string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an attendance record (e.g. the second half of a day with a break)",
		Long: `Add a new attendance record with check-in --in and check-out --out on the
given day (--date, default today).

Unlike 'clock edit', it always creates a new record, also on days that
already have records, so a day with a break is two 'clock add' calls.
If the new span overlaps an existing record (including a record from the
day before that runs past midnight, or a running record) the command
aborts with a "conflict" error, names the overlapping record and changes
nothing. Adjacent records (12:00 out, 12:00 in) do not overlap.

Times are HH:MM on --date or "YYYY-MM-DD HH:MM". A time more than one
minute in the future, or a check-out not after the check-in, is a
validation error.

Adding attendance over the API requires the 'Attendance Officer' access
right, with you set as your own attendance manager.`,
		Example: `  # The afternoon of a day with a break: a second record
  odoo-work-cli clock add --date 2026-04-07 --in 17:00 --out 18:30

  # An overlapping span is a conflict, and nothing is changed
  odoo-work-cli clock add --date 2026-04-08 --in 11:00 --out 13:00`,
		Annotations: map[string]string{
			manual.Output: `Prints "Created attendance #ID: FROM - TO".

With --json the created record.`,
			manual.Schemas: "attendance",
			manual.Errors:  "conflict, access_denied",
			manual.SeeAlso: "odoo-work-cli clock list, odoo-work-cli clock edit, how-to/30-clock-in-out.md",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}
			req, err := resolveAddTimes(dateStr, inStr, outStr, time.Now())
			if err != nil {
				return err
			}
			in, out := *req.checkIn, *req.checkOut

			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			candidates, err := overlapCandidates(client, req.day, out)
			if err != nil {
				return err
			}
			if overlaps := findOverlaps(candidates, in, out); len(overlaps) > 0 {
				return overlapConflict(in, out, overlaps)
			}

			newID, err := client.CreateAttendance(in, out)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if format.IsMachine() {
				return writeAttendanceByID(w, format, client, newID, "created")
			}
			_, _ = fmt.Fprintf(w, "Created attendance #%d: %s\n", newID, formatSpan(in, out))
			return nil
		},
	}

	cmd.Flags().StringVar(&dateStr, "date", "", "day of the new record, `YYYY-MM-DD`")
	cmd.Flags().StringVar(&inStr, "in", "", "check-in `TIME`, HH:MM or \"YYYY-MM-DD HH:MM\"")
	cmd.Flags().StringVar(&outStr, "out", "", "check-out `TIME`, HH:MM or \"YYYY-MM-DD HH:MM\"")
	manual.SetFlagDefault(cmd.Flags(), "date", "today")
	manual.SetFlagRequired(cmd.Flags(), "in")
	manual.SetFlagRequired(cmd.Flags(), "out")

	return cmd
}
