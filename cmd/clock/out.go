package clock

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/spf13/cobra"
)

func outCMD(deps *app.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "out",
		Short: "Clock out (end attendance)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := deps.RequireClient()
			if err != nil {
				return err
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
	return cmd
}
