package clock

import (
	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func CMD(deps *app.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clock",
		Short: "Clock in/out and attendance status",
	}

	// --format/--json apply to every clock subcommand.
	output.AddFlags(cmd.PersistentFlags())

	cmd.AddCommand(inCMD(deps))
	cmd.AddCommand(outCMD(deps))
	cmd.AddCommand(statusCMD(deps))
	cmd.AddCommand(listCMD(deps))
	cmd.AddCommand(editCMD(deps))

	return cmd
}
