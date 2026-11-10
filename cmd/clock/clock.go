package clock

import (
	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func CMD(deps *app.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clock",
		Short: "Clock in/out and attendance status",
		Long: `Record attendance: clock in and out now or at an earlier time, show
today's status, list the records of a day or week, and correct or add
records afterwards.

A record belongs to the day of its check-in. Clock in and clock out
without --at go through the Odoo attendance systray of the web session,
which every employee may use; it needs the login password and, with
two-factor authentication, the TOTP secret. --at, edit and add write
hr.attendance over XML-RPC, which requires the Attendance Officer access
right.`,
		Example: `  # What is recorded for a day
  odoo-work-cli clock list --date 2026-04-08`,
		Annotations: map[string]string{
			manual.Group:   manual.GroupAttendance,
			manual.SeeAlso: "how-to/30-clock-in-out.md, explanation/attendance-model.md, explanation/transports.md",
		},
	}

	// --format/--json apply to every clock subcommand.
	output.AddFlags(cmd.PersistentFlags())

	cmd.AddCommand(inCMD(deps))
	cmd.AddCommand(outCMD(deps))
	cmd.AddCommand(statusCMD(deps))
	cmd.AddCommand(listCMD(deps))
	cmd.AddCommand(editCMD(deps))
	cmd.AddCommand(addCMD(deps))

	return cmd
}
