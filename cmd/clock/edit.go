package clock

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/odoo"
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
		return time.Time{}, err
	}
	t, err := parsing.ParseClockTime(at, day)
	if err != nil {
		return time.Time{}, err
	}
	if t.After(now.Add(futureGrace)) {
		return time.Time{}, fmt.Errorf("time %s is in the future", t.Format("2006-01-02 15:04"))
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
		return nil, errors.New("nothing to change: provide --in and/or --out")
	}

	day, err := parsing.ParseDay(dateStr, now)
	if err != nil {
		return nil, err
	}

	result := &editTimes{day: day}
	parse := func(s string) (*time.Time, error) {
		if s == "" {
			return nil, nil
		}
		t, err := parsing.ParseClockTime(s, day)
		if err != nil {
			return nil, err
		}
		if t.After(now.Add(futureGrace)) {
			return nil, fmt.Errorf("time %s is in the future", t.Format("2006-01-02 15:04"))
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
		return nil, fmt.Errorf("check-out %s is not after check-in %s",
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
		return nil, fmt.Errorf("no attendance record with ID %d on that date", id)
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
		return nil, fmt.Errorf("multiple attendance records on that date, select one with --id:\n%s",
			strings.Join(lines, "\n"))
	}
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

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit an attendance record's check-in/check-out time",
		Long: `Edit the check-in and/or check-out time of an attendance record.

By default the record of the given day (--date, default today) is edited.
If the day has several records, select one with --id. If the day has no
record yet, one is created when both --in and --out are given.

Editing attendance over the API requires the 'Attendance Officer' access
right, with you set as your own attendance manager.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := resolveEditTimes(dateStr, inStr, outStr, time.Now())
			if err != nil {
				return err
			}

			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			records, err := client.ListAttendance(req.day.UTC(), req.day.AddDate(0, 0, 1).UTC())
			if err != nil {
				return err
			}

			target, err := pickEditTarget(records, id)
			if err != nil {
				return err
			}

			if target == nil {
				if req.checkIn == nil || req.checkOut == nil {
					return fmt.Errorf("no attendance record on %s: provide both --in and --out to create one",
						req.day.Format("2006-01-02"))
				}
				newID, err := client.CreateAttendance(*req.checkIn, *req.checkOut)
				if err != nil {
					return err
				}
				fmt.Printf("Created attendance #%d: %s - %s\n", newID,
					req.checkIn.Format("2006-01-02 15:04"), req.checkOut.Format("15:04"))
				return nil
			}

			rec, err := client.EditAttendance(target.ID, req.checkIn, req.checkOut)
			if err != nil {
				return err
			}
			fmt.Printf("Updated attendance %s\n", formatRecord(rec))
			return nil
		},
	}

	cmd.Flags().StringVar(&dateStr, "date", "", "day of the record to edit (YYYY-MM-DD, default today)")
	cmd.Flags().StringVar(&inStr, "in", "", "new check-in time (HH:MM or YYYY-MM-DD HH:MM)")
	cmd.Flags().StringVar(&outStr, "out", "", "new check-out time (HH:MM or YYYY-MM-DD HH:MM)")
	cmd.Flags().Int64Var(&id, "id", 0, "attendance record ID (required if the day has several records)")

	return cmd
}
