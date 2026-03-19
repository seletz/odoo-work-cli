package clock

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/spf13/cobra"
)

func inCMD(deps *app.Deps) *cobra.Command {
	InCmd := &cobra.Command{
		Use:   "in",
		Short: "Clock in (start attendance)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			_, err = client.ClockIn()
			if err != nil {
				return err
			}

			fmt.Printf("Clocked in at %s\n", time.Now().Format("15:04"))
			return nil
		},
	}

	return InCmd
}
