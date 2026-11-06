package entries

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func updateCmd(deps *app.Deps) *cobra.Command {
	ops := &subOps{}
	cmd := &cobra.Command{
		Use:   "update ID",
		Short: "Update an existing timesheet entry",
		Long: `Update an existing timesheet entry.

With --json (or --format json|jsonl) the updated entry is printed.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			id, err := parseEntryID(args[0])
			if err != nil {
				return err
			}

			fields, err := buildUpdateFields(cmd, ops)
			if err != nil {
				return err
			}

			// A missing entry fails as not_found before any write.
			if _, err := client.GetTimesheet(id); err != nil {
				return err
			}
			if err := client.UpdateTimesheet(id, fields); err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			if format.IsMachine() {
				entry, err := client.GetTimesheet(id)
				if err != nil {
					return fmt.Errorf("entry %d was updated, but reading it back failed: %w", id, err)
				}
				return output.WriteObject(w, format, output.NewEntry(*entry))
			}
			_, _ = fmt.Fprintf(w, "Updated entry %d\n", id)
			return nil
		},
	}

	cmd.Flags().Int64Var(&ops.projectID, "project-id", 0, "Odoo project ID")
	cmd.Flags().Int64Var(&ops.taskID, "task-id", 0, "Odoo task ID")
	cmd.Flags().StringVar(&ops.date, "date", "", "entry date YYYY-MM-DD")
	cmd.Flags().StringVar(&ops.hours, "hours", "", "hours worked (e.g. 2.5 or 2:30)")
	cmd.Flags().StringVar(&ops.description, "description", "", "work description")

	return cmd
}
