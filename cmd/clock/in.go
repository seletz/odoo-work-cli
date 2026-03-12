package clock

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/spf13/cobra"
)

func inCMD(client *odoo.XMLRPCClient) *cobra.Command {
	InCmd := &cobra.Command{
		Use:   "in",
		Short: "Clock in (start attendance)",
		RunE: func(cmd *cobra.Command, args []string) error {

			_, err := client.ClockIn()
			if err != nil {
				return err
			}

			fmt.Printf("Clocked in at %s\n", time.Now().Format("15:04"))
			return nil
		},
	}

	return InCmd
}
