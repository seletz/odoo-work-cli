package clock

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
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
		Long: `Clock out (end attendance).

With --json (or --format json|jsonl) the closed attendance record is
printed.`,
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
		"clock out at a past time (HH:MM or YYYY-MM-DD HH:MM) instead of now; requires attendance officer rights")

	return cmd
}
