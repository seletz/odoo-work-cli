package clock

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func inCMD(deps *app.Deps) *cobra.Command {
	var at string

	InCmd := &cobra.Command{
		Use:   "in",
		Short: "Clock in (start attendance)",
		Long: `Start an attendance record now, through the Odoo web session (the
controller of the attendance systray). It needs the login password
(ODOO_WEB_PASSWORD, or password in [op_secrets]) and, with two-factor
authentication, the TOTP secret.

--at TIME uses an earlier time instead: HH:MM is today, "YYYY-MM-DD HH:MM"
any day. A time more than one minute in the future is a validation error.
--at writes hr.attendance over XML-RPC with the API key, which requires the
Attendance Officer access right, with you set as your own attendance
manager.

Clocking in while already clocked in is a conflict error.`,
		Example: `  # Back from lunch: close the morning, then clock in again, both afterwards
  odoo-work-cli clock out --at "2026-04-09 12:00"
  odoo-work-cli clock in --at "2026-04-09 12:45"`,
		Annotations: map[string]string{
			manual.Output: `Prints "Clocked in at HH:MM"; with --at "Clocked in at YYYY-MM-DD HH:MM
(#ID)".

With --json the new, running attendance record.`,
			manual.Schemas: "attendance",
			manual.Errors:  "conflict, access_denied",
			manual.SeeAlso: "odoo-work-cli clock out, odoo-work-cli clock status, how-to/30-clock-in-out.md",
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
			w := cmd.OutOrStdout()

			if at != "" {
				t := atTime
				id, err := client.ClockInAt(t)
				if err != nil {
					return err
				}
				if format.IsMachine() {
					return writeAttendanceByID(w, format, client, id, "created")
				}
				_, _ = fmt.Fprintf(w, "Clocked in at %s (#%d)\n", t.Format("2006-01-02 15:04"), id)
				return nil
			}

			id, err := client.ClockIn()
			if err != nil {
				return err
			}
			if format.IsMachine() {
				return writeAttendanceByID(w, format, client, id, "created")
			}
			_, _ = fmt.Fprintf(w, "Clocked in at %s\n", time.Now().Format("15:04"))
			return nil
		},
	}

	InCmd.Flags().StringVar(&at, "at", "",
		"clock in at an earlier `TIME`, HH:MM or \"YYYY-MM-DD HH:MM\", instead of now; requires attendance officer rights")
	manual.SetFlagDefault(InCmd.Flags(), "at", "now")

	return InCmd
}
