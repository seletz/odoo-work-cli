package entries

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func addCmd(deps *app.Deps) *cobra.Command {
	ops := &subOps{}
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new timesheet entry",
		Long: `Create a new timesheet entry.

With --json (or --format json|jsonl) the created entry is printed.

With --dry-run the input is validated and project and task are resolved, then
the entry that would be created is printed and nothing is written. With --json
it is the same object as the real call, with "id" null and "dry_run": true,
plus "duplicates" (entries of the same day with the same project, task and
description).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}
			params, err := buildTimesheetWriteParams(ops.projectID,
				ops.taskID,
				ops.date,
				ops.description,
				ops.hours)
			if err != nil {
				return err
			}
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			if dryRun {
				return writeDryRun(cmd.OutOrStdout(), format, client, "create", odoo.TimesheetEntry{
					Date: params.Date, ProjectID: params.ProjectID, TaskID: params.TaskID,
					Name: params.Name, Hours: params.Hours,
				})
			}

			id, err := client.CreateTimesheet(params)
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			if format.IsMachine() {
				entry, err := client.GetTimesheet(id)
				if err != nil {
					return fmt.Errorf("entry %d was created, but reading it back failed: %w", id, err)
				}
				return output.WriteObject(w, format, output.NewEntry(*entry))
			}
			_, _ = fmt.Fprintf(w, "Created entry %d\n", id)
			return nil
		},
	}

	cmd.Flags().Int64Var(&ops.projectID, "project-id", 0, "Odoo project ID (required)")
	cmd.Flags().Int64Var(&ops.taskID, "task-id", 0, "Odoo task ID (optional)")
	cmd.Flags().StringVar(&ops.date, "date", "", "entry date YYYY-MM-DD (defaults to today)")
	cmd.Flags().StringVar(&ops.hours, "hours", "", "hours worked (e.g. 2.5 or 2:30)")
	cmd.Flags().StringVar(&ops.description, "description", "", "work description (required)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, dryRunFlagUsage)
	_ = cmd.MarkFlagRequired("hours")
	_ = cmd.MarkFlagRequired("description")

	return cmd
}
