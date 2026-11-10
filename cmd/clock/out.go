package clock

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/spf13/cobra"
)

func outCMD(deps *app.Deps) *cobra.Command {
	var at string

	cmd := &cobra.Command{
		Use:   "out",
		Short: "Clock out (end attendance)",
		Long: `End the running attendance record now, through the Odoo web session
(the controller of the attendance systray). It needs the login password
(ODOO_WEB_PASSWORD, or password in [op_secrets]) and, with two-factor
authentication, the TOTP secret. The running record may have started on
an earlier day.

--at TIME uses an earlier time instead: HH:MM is today, "YYYY-MM-DD HH:MM"
any day. A time more than one minute in the future is a validation error.
--at writes hr.attendance over XML-RPC with the API key, which requires the
Attendance Officer access right, with you set as your own attendance
manager. The check-out must be after the check-in.

Clocking out while not clocked in is a conflict error.`,
		Example: `  # Clock out afterwards, at the time you left
  odoo-work-cli clock out --at "2026-04-09 17:15"`,
		Annotations: map[string]string{
			manual.Output: `Prints "Clocked out at HH:MM" (with --at the date as well) and the
duration of the record, as H:MM and in decimal hours.

With --json the closed attendance record.`,
			manual.Schemas: "attendance",
			manual.Errors:  "conflict, access_denied",
			manual.SeeAlso: "odoo-work-cli clock in, odoo-work-cli clock status, how-to/30-clock-in-out.md",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}
			var atTime time.Time
			if at != "" {
				var err error
				if atTime, err = parseAtFlag(at, time.Now()); err != nil {
					return err
				}
			}
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			var rec *odoo.AttendanceRecord
			var label string
			if at != "" {
				t := atTime
				if rec, err = client.ClockOutAt(t); err != nil {
					return err
				}
				label = t.Format("2006-01-02 15:04")
			} else {
				if rec, err = client.ClockOut(); err != nil {
					return err
				}
				label = time.Now().Format("15:04")
			}

			w := cmd.OutOrStdout()
			if format.IsMachine() {
				return output.WriteObject(w, format, output.NewAttendance(*rec))
			}
			_, _ = fmt.Fprintf(w, "Clocked out at %s\n", label)
			_, _ = fmt.Fprintf(w, "Duration: %s (%.2fh)\n", tui.FormatHours(rec.WorkedHours), rec.WorkedHours)
			return nil
		},
	}

	cmd.Flags().StringVar(&at, "at", "",
		"clock out at an earlier `TIME`, HH:MM or \"YYYY-MM-DD HH:MM\", instead of now; requires attendance officer rights")
	manual.SetFlagDefault(cmd.Flags(), "at", "now")

	return cmd
}
