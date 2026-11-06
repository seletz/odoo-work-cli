package clock

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/spf13/cobra"
)

func statusCMD(deps *app.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current attendance status",
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
