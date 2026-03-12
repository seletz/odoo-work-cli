package entries

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/spf13/cobra"
)

func updateCmd(client *odoo.XMLRPCClient) *cobra.Command {
	ops := &subOps{}
	cmd := &cobra.Command{
		Use:   "update ID",
		Short: "Update an existing timesheet entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseEntryID(args[0])
			if err != nil {
				return err
			}

			fields, err := buildUpdateFields(cmd, ops)
			if err != nil {
				return err
			}

			if err := client.UpdateTimesheet(id, fields); err != nil {
				return err
			}

			fmt.Printf("Updated entry %d\n", id)
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
