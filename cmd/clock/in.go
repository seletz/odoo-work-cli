package clock

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/spf13/cobra"
)

func inCMD(deps *app.Deps) *cobra.Command {
	var at string

	InCmd := &cobra.Command{
		Use:   "in",
		Short: "Clock in (start attendance)",
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
				id, err := client.ClockInAt(t)
				if err != nil {
					return err
				}
				fmt.Printf("Clocked in at %s (#%d)\n", t.Format("2006-01-02 15:04"), id)
				return nil
			}

			_, err = client.ClockIn()
			if err != nil {
				return err
			}

			fmt.Printf("Clocked in at %s\n", time.Now().Format("15:04"))
			return nil
		},
	}

	InCmd.Flags().StringVar(&at, "at", "",
		"clock in at a past time (HH:MM or YYYY-MM-DD HH:MM) instead of now; requires attendance officer rights")

	return InCmd
}
