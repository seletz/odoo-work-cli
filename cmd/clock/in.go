package clock

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func inCMD(deps *app.Deps) *cobra.Command {
	var at string

	InCmd := &cobra.Command{
		Use:   "in",
		Short: "Clock in (start attendance)",
		Long: `Clock in (start attendance).

With --json (or --format json|jsonl) the new, running attendance record is
printed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()

			if at != "" {
				t, err := parseAtFlag(at, time.Now())
				if err != nil {
					return err
				}
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
		"clock in at a past time (HH:MM or YYYY-MM-DD HH:MM) instead of now; requires attendance officer rights")

	return InCmd
}
