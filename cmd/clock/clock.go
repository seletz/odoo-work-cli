package clock

import (
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/spf13/cobra"
)

func ClockCMD(client *odoo.XMLRPCClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clock",
		Short: "Clock in/out and attendance status",
	}

	cmd.AddCommand(inCMD(client))
	cmd.AddCommand(outCMD(client))
	cmd.AddCommand(statusCMD(client))

	return cmd
}
