package clock

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/spf13/cobra"
)

func outCMD(deps *app.Deps) *cobra.Command {
	var at string

	cmd := &cobra.Command{
		Use:   "out",
		Short: "Clock out (end attendance)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			if at != "" {
				t, err := parseAtFlag(at, time.Now())
				if err != nil {
					return err
				}
				rec, err := client.ClockOutAt(t)
				if err != nil {
					return err
				}
				fmt.Printf("Clocked out at %s\n", t.Format("2006-01-02 15:04"))
				fmt.Printf("Duration: %s (%.2fh)\n", tui.FormatHours(rec.WorkedHours), rec.WorkedHours)
				return nil
			}

			rec, err := client.ClockOut()
			if err != nil {
				return err
			}

			fmt.Printf("Clocked out at %s\n", time.Now().Format("15:04"))
			fmt.Printf("Duration: %s (%.2fh)\n", tui.FormatHours(rec.WorkedHours), rec.WorkedHours)
			return nil
		},
	}

	cmd.Flags().StringVar(&at, "at", "",
		"clock out at a past time (HH:MM or YYYY-MM-DD HH:MM) instead of now; requires attendance officer rights")

	return cmd
}
