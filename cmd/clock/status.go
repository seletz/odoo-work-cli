package clock

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/spf13/cobra"
)

func statusCMD(deps *app.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current attendance status",
		Long: `Show whether you are clocked in and since when, and today's attendance
periods with their durations and the total. A running period counts up to
now. Today is the local calendar day; a record of an earlier day that is
still running is included.`,
		Example: `  odoo-work-cli clock status

  # As JSON, e.g. for a status bar
  odoo-work-cli clock status --json`,
		Annotations: map[string]string{
			manual.Output: `"Status: Clocked in since HH:MM (H:MM elapsed)" or "Status: Not clocked
in", then today's periods with the columns #, Check In, Check Out (--:--
while running) and Duration, and the total; or "No attendance records
today."

With --json the clock status with today's periods.`,
			manual.Schemas:        "clock_status",
			manual.StaticExamples: "true",
			manual.SeeAlso:        "odoo-work-cli clock list, explanation/attendance-model.md",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			status, err := client.AttendanceStatus()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if format.IsMachine() {
				return output.WriteObject(w, format, output.NewClockStatus(status, time.Now()))
			}

			if status.ClockedIn && status.CheckIn != nil {
				elapsed := time.Since(*status.CheckIn).Hours()
				_, _ = fmt.Fprintf(w, "Status: Clocked in since %s (%s elapsed)\n",
					status.CheckIn.Local().Format("15:04"),
					tui.FormatHours(elapsed))
			} else {
				_, _ = fmt.Fprintln(w, "Status: Not clocked in")
			}

			if len(status.Periods) > 0 {
				_, _ = fmt.Fprint(w, "\nToday's attendance:\n\n")
				_, _ = fmt.Fprintf(w, "%-3s %-10s %-10s %s\n", "#", "Check In", "Check Out", "Duration")
				_, _ = fmt.Fprintf(w, "%-3s %-10s %-10s %s\n", "---", "----------", "----------", "--------")
				for i, p := range status.Periods {
					checkIn := p.CheckIn.Local().Format("15:04")
					var checkOut, duration string
					if p.CheckOut != nil {
						checkOut = p.CheckOut.Local().Format("15:04")
						duration = tui.FormatHours(p.WorkedHours)
					} else {
						checkOut = "--:--"
						elapsed := time.Since(p.CheckIn).Hours()
						duration = tui.FormatHours(elapsed) + " (running)"
					}
					_, _ = fmt.Fprintf(w, "%-3d %-10s %-10s %s\n", i+1, checkIn, checkOut, duration)
				}
				_, _ = fmt.Fprintf(w, "\nTotal: %s\n", tui.FormatHours(status.TotalHours))
			} else {
				_, _ = fmt.Fprintln(w, "\nNo attendance records today.")
			}
			return nil
		},
	}
	return cmd
}
