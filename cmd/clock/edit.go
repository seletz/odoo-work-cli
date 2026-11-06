package clock

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/seletz/odoo-work-cli/internal/parsing"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/spf13/cobra"
)

// futureGrace tolerates small clock skew when rejecting future times.
const futureGrace = time.Minute

// parseAtFlag resolves an --at flag value ("HH:MM" or "YYYY-MM-DD HH:MM")
// against now's day and rejects times in the future.
func parseAtFlag(at string, now time.Time) (time.Time, error) {
	day, err := parsing.ParseDay("", now)
	if err != nil {
		return time.Time{}, apperr.Wrap(apperr.KindValidation, err)
	}
	t, err := parsing.ParseClockTime(at, day)
	if err != nil {
		return time.Time{}, apperr.Wrap(apperr.KindValidation, err)
	}
	if t.After(now.Add(futureGrace)) {
		return time.Time{}, apperr.New(apperr.KindValidation, "time %s is in the future", t.Format("2006-01-02 15:04"))
	}
	return t, nil
}

// editTimes holds the resolved clock edit target day and new times.
type editTimes struct {
	day      time.Time
	checkIn  *time.Time
	checkOut *time.Time
}

// resolveEditTimes parses and validates the clock edit flags against now.
func resolveEditTimes(dateStr, inStr, outStr string, now time.Time) (*editTimes, error) {
	if inStr == "" && outStr == "" {
		return nil, apperr.New(apperr.KindValidation, "nothing to change: provide --in and/or --out")
	}

	day, err := parsing.ParseDay(dateStr, now)
	if err != nil {
		return nil, apperr.Wrap(apperr.KindValidation, err)
	}

	result := &editTimes{day: day}
	parse := func(s string) (*time.Time, error) {
		if s == "" {
			return nil, nil
		}
		t, err := parsing.ParseClockTime(s, day)
		if err != nil {
			return nil, apperr.Wrap(apperr.KindValidation, err)
		}
		if t.After(now.Add(futureGrace)) {
			return nil, apperr.New(apperr.KindValidation, "time %s is in the future", t.Format("2006-01-02 15:04"))
		}
		return &t, nil
	}

	if result.checkIn, err = parse(inStr); err != nil {
		return nil, err
	}
	if result.checkOut, err = parse(outStr); err != nil {
		return nil, err
	}
	if result.checkIn != nil && result.checkOut != nil && !result.checkOut.After(*result.checkIn) {
		return nil, apperr.New(apperr.KindValidation, "check-out %s is not after check-in %s",
			result.checkOut.Format("15:04"), result.checkIn.Format("15:04"))
	}
	return result, nil
}

// pickEditTarget selects the attendance record to edit from the day's
// records. Returns nil (without error) when there are no records; requires
// an explicit ID when the day has more than one record.
func pickEditTarget(records []odoo.AttendanceRecord, id int64) (*odoo.AttendanceRecord, error) {
	if id > 0 {
		for i := range records {
			if records[i].ID == id {
				return &records[i], nil
			}
		}
		return nil, apperr.New(apperr.KindNotFound, "no attendance record with ID %d on that date", id)
	}

	switch len(records) {
	case 0:
		return nil, nil
	case 1:
		return &records[0], nil
	default:
		var lines []string
		for _, r := range records {
			lines = append(lines, "  "+formatRecord(&r))
		}
		return nil, apperr.New(apperr.KindValidation, "multiple attendance records on that date, select one with --id:\n%s",
			strings.Join(lines, "\n"))
	}
}

// writeAttendanceByID reads back a just-written attendance record and
// prints it in a machine format.
func writeAttendanceByID(w io.Writer, format output.Format, client odoo.Client, id int64, action string) error {
	rec, err := client.GetAttendance(id)
	if err != nil {
		return fmt.Errorf("attendance #%d was %s, but reading it back failed: %w", id, action, err)
	}
	return output.WriteObject(w, format, output.NewAttendance(*rec))
}

// describeChange renders the old and new values of an edited record, so a
// change is never silent.
func describeChange(before, after *odoo.AttendanceRecord) string {
	line := func(label string, r *odoo.AttendanceRecord) string {
		if r.CheckOut == nil {
			return fmt.Sprintf("  %s: %s - --:-- (running)", label, r.CheckIn.Local().Format(dayTimeLayout))
		}
		return fmt.Sprintf("  %s: %s - %s (%s)", label, r.CheckIn.Local().Format(dayTimeLayout),
			formatCheckOut(r), tui.FormatHours(r.WorkedHours))
	}
	return fmt.Sprintf("Updated attendance #%d\n%s\n%s", after.ID, line("was", before), line("now", after))
}

// existingOnDay returns the records whose check-in falls on day. An open
// record from an earlier day, which the range query also returns, does not
// count as a record of that day.
func existingOnDay(records []odoo.AttendanceRecord, day time.Time) []odoo.AttendanceRecord {
	end := day.AddDate(0, 0, 1)
	var out []odoo.AttendanceRecord
	for _, r := range records {
		if !r.CheckIn.Before(day) && r.CheckIn.Before(end) {
			out = append(out, r)
		}
	}
	return out
}

// createOnlyConflict is the error for --create-only on a day that already
// has attendance. It lists the records so nothing needs a second lookup.
func createOnlyConflict(day time.Time, existing []odoo.AttendanceRecord) error {
	var lines []string
	for i := range existing {
		lines = append(lines, "  "+formatRecord(&existing[i]))
	}
	return apperr.New(apperr.KindConflict,
		"attendance record already exists on %s (id %d); nothing was changed. Existing records:\n%s",
		day.Format("2006-01-02"), existing[0].ID, strings.Join(lines, "\n"))
}

// formatRecord renders an attendance record for terminal output.
func formatRecord(rec *odoo.AttendanceRecord) string {
	checkIn := rec.CheckIn.Local().Format("2006-01-02 15:04")
	if rec.CheckOut == nil {
		return fmt.Sprintf("#%d %s - --:-- (running)", rec.ID, checkIn)
	}
	return fmt.Sprintf("#%d %s - %s (%s)", rec.ID, checkIn,
		rec.CheckOut.Local().Format("15:04"), tui.FormatHours(rec.WorkedHours))
}

func editCMD(deps *app.Deps) *cobra.Command {
	var dateStr, inStr, outStr string
	var id int64
	var createOnly bool

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit an attendance record's check-in/check-out time",
		Long: `Edit the check-in and/or check-out time of an attendance record.

By default the record of the given day (--date, default today) is edited.
If the day has several records, select one with --id. If the day has no
record yet, one is created when both --in and --out are given. The old and
the new values are printed. Use 'clock list' to see the existing records first.

With --create-only the command only ever creates: if the day already has a
record it aborts with a "conflict" error and changes nothing. Use it when
backfilling days, so that an existing record is never overwritten.

Editing attendance over the API requires the 'Attendance Officer' access
right, with you set as your own attendance manager.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}
			req, err := resolveEditTimes(dateStr, inStr, outStr, time.Now())
			if err != nil {
				return err
			}

			if createOnly {
				if id > 0 {
					return apperr.New(apperr.KindValidation, "--create-only cannot be combined with --id")
				}
				if req.checkIn == nil || req.checkOut == nil {
					return apperr.New(apperr.KindValidation, "--create-only needs both --in and --out")
				}
			}

			client, err := deps.RequireClient()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()

			records, err := client.ListAttendance(req.day.UTC(), req.day.AddDate(0, 0, 1).UTC())
			if err != nil {
				return err
			}

			if createOnly {
				if existing := existingOnDay(records, req.day); len(existing) > 0 {
					return createOnlyConflict(req.day, existing)
				}
			}

			target, err := pickEditTarget(records, id)
			if err != nil {
				return err
			}

			if target == nil {
				if req.checkIn == nil || req.checkOut == nil {
					return apperr.New(apperr.KindNotFound, "no attendance record on %s: provide both --in and --out to create one",
						req.day.Format("2006-01-02"))
				}
				newID, err := client.CreateAttendance(*req.checkIn, *req.checkOut)
				if err != nil {
					return err
				}
				if format.IsMachine() {
					created, err := client.GetAttendance(newID)
					if err != nil {
						return fmt.Errorf("attendance #%d was created, but reading it back failed: %w", newID, err)
					}
					return output.WriteObject(w, format, output.NewAttendanceEdit(nil, *created))
				}
				_, _ = fmt.Fprintf(w, "Created attendance #%d: %s - %s\n", newID,
					req.checkIn.Format("2006-01-02 15:04"), req.checkOut.Format("15:04"))
				return nil
			}

			rec, err := client.EditAttendance(target.ID, req.checkIn, req.checkOut)
			if err != nil {
				return err
			}
			if format.IsMachine() {
				return output.WriteObject(w, format, output.NewAttendanceEdit(target, *rec))
			}
			_, _ = fmt.Fprintln(w, describeChange(target, rec))
			return nil
		},
	}

	cmd.Flags().StringVar(&dateStr, "date", "", "day of the record to edit (YYYY-MM-DD, default today)")
	cmd.Flags().StringVar(&inStr, "in", "", "new check-in time (HH:MM or YYYY-MM-DD HH:MM)")
	cmd.Flags().StringVar(&outStr, "out", "", "new check-out time (HH:MM or YYYY-MM-DD HH:MM)")
	cmd.Flags().BoolVar(&createOnly, "create-only", false, "only create; abort with a conflict error if the day already has a record")
	cmd.Flags().Int64Var(&id, "id", 0, "attendance record ID (required if the day has several records)")

	return cmd
}
