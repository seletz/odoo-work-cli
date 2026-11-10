package entries

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/manual"
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
		Long: `Create a timesheet entry of --hours on the day --date (default today)
with the description --description, on the project --project-id and,
optionally, its task --task-id.

--hours takes decimal hours (2.5) or H:MM (2:30) and must be greater than
zero. The input is checked before any connection is made.

With --dry-run the input is validated, project and task are resolved
(an unknown project, or a task of another project, is a validation
error), and the entry that would be created is printed; nothing is
written. Entries of the same day with the same project, task and
description are reported as possible duplicates.`,
		Example: `  # Preview 90 minutes on a task without writing anything
  odoo-work-cli entries add --project-id 1 --task-id 13 --date 2026-04-09 --hours 1:30 --description "Review fuzzy matching" --dry-run

  # Book 45 minutes
  odoo-work-cli entries add --project-id 2 --task-id 22 --date 2026-04-09 --hours 0:45 --description "Answer support mail"

  # Book and print the created entry as JSON
  odoo-work-cli entries add --project-id 3 --task-id 31 --date 2026-04-09 --hours 0.5 --description "Standup" --json`,
		Annotations: map[string]string{
			manual.Output: `Prints "Created entry ID". With --dry-run, "Dry run: would create this
entry; nothing was written." followed by the entry, and a warning per
possible duplicate.

With --json the created entry. With --dry-run and --json the proposed
entry with id null, dry_run true and duplicates (entry_dry_run.schema.json).`,
			manual.Schemas: "entry, entry_dry_run",
			manual.SeeAlso: "odoo-work-cli projects, odoo-work-cli tasks, how-to/20-book-time.md, how-to/50-dry-run.md",
		},
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

	cmd.Flags().Int64Var(&ops.projectID, "project-id", 0, "`ID` of the project to book on (see projects)")
	cmd.Flags().Int64Var(&ops.taskID, "task-id", 0, "`ID` of the task to book on (see tasks); must belong to the project")
	cmd.Flags().StringVar(&ops.date, "date", "", "day of the entry, `YYYY-MM-DD`")
	cmd.Flags().StringVar(&ops.hours, "hours", "", "hours worked, decimal (2.5) or `H`:MM (2:30)")
	cmd.Flags().StringVar(&ops.description, "description", "", "`TEXT` describing the work")
	manual.SetFlagRequired(cmd.Flags(), "project-id")
	manual.SetFlagDefault(cmd.Flags(), "date", "today")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, dryRunFlagUsage)
	_ = cmd.MarkFlagRequired("hours")
	_ = cmd.MarkFlagRequired("description")

	return cmd
}
