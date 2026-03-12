package clock

import (
	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/spf13/cobra"
)

func CMD(deps *app.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clock",
		Short: "Clock in/out and attendance status",
	}

	cmd.AddCommand(inCMD(deps))
	cmd.AddCommand(outCMD(deps))
	cmd.AddCommand(statusCMD(deps))

	return cmd
}
