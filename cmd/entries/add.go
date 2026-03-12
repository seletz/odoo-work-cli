package entries

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/spf13/cobra"
)

func addCmd(client *odoo.XMLRPCClient) *cobra.Command {
	ops := &subOps{}
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new timesheet entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			params, err := buildTimesheetWriteParams(ops.projectID,
				ops.taskID,
				ops.date,
				ops.description,
				ops.hours)
			if err != nil {
				return err
			}

			id, err := client.CreateTimesheet(params)
			if err != nil {
				return err
			}

			fmt.Printf("Created entry %d\n", id)
			return nil
		},
	}

	cmd.Flags().Int64Var(&ops.projectID, "project-id", 0, "Odoo project ID (required)")
	cmd.Flags().Int64Var(&ops.taskID, "task-id", 0, "Odoo task ID (optional)")
	cmd.Flags().StringVar(&ops.date, "date", "", "entry date YYYY-MM-DD (defaults to today)")
	cmd.Flags().StringVar(&ops.hours, "hours", "", "hours worked (e.g. 2.5 or 2:30)")
	cmd.Flags().StringVar(&ops.description, "description", "", "work description (required)")
	_ = cmd.MarkFlagRequired("hours")
	_ = cmd.MarkFlagRequired("description")

	return cmd
}
