package fields

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/spf13/cobra"
)

func CMD(deps *app.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fields MODEL",
		Short: "Inspect Odoo model fields",
		Long: `Print the fields of the Odoo model MODEL with their type, label and
whether they are required. Use it to find the technical field names for
extra_fields and filters in the configuration, e.g. a custom
x_studio_ field.

Common models: project.project (projects), project.task (tasks),
account.analytic.line (timesheet entries), hr.attendance (attendance).`,
		Example: `  # The fields of a project
  odoo-work-cli fields project.project`,
		Args: cobra.ExactArgs(1),
		Annotations: map[string]string{
			manual.Group:   manual.GroupConfig,
			manual.Output:  "The columns Field, Type, Label and Required. There is no JSON output.",
			manual.SeeAlso: "odoo-work-cli projects, reference/40-configuration.md",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			fields, err := client.GetFields(args[0])
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(w, "%-30s %-15s %-30s %s\n", "Field", "Type", "Label", "Required")
			_, _ = fmt.Fprintf(w, "%-30s %-15s %-30s %s\n", "------------------------------", "---------------", "------------------------------", "--------")
			for _, f := range fields {
				_, _ = fmt.Fprintf(w, "%-30s %-15s %-30s %v\n", f.Name, f.Type, f.String, f.Required)
			}
			return nil
		},
	}

	return cmd
}
