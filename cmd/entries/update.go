package entries

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func updateCmd(deps *app.Deps) *cobra.Command {
	ops := &subOps{}
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "update ID",
		Short: "Update an existing timesheet entry",
		Long: `Change the timesheet entry ID. Only the fields whose flags are given are
written; at least one flag is required. --description must not be empty.
The entry is read first, so an unknown ID fails with not_found before
anything is written. Odoo may refuse changes to validated entries (access_denied).

With --dry-run the flags are merged into the entry, the result is
validated (project and task are resolved) and printed; nothing is
written. Entries of the same day with the same project, task and
description are reported as possible duplicates.`,
		Example: `  # Preview a correction of the hours
  odoo-work-cli entries update 4018 --hours 2:00 --dry-run

  # Change the description
  odoo-work-cli entries update 4018 --description "Review fuzzy matching, round 2"`,
		Annotations: map[string]string{
			manual.Output: `Prints "Updated entry ID". With --dry-run, "Dry run: would update this
entry; nothing was written." followed by the entry, and a warning per
possible duplicate.

With --json the updated entry. With --dry-run and --json the entry as it
would be, with dry_run true and duplicates (entry_dry_run.schema.json).`,
			manual.Schemas: "entry, entry_dry_run",
			manual.Errors:  "not_found, access_denied",
			manual.SeeAlso: "odoo-work-cli entries, how-to/50-dry-run.md",
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
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
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			// A missing entry fails as not_found before any write.
			existing, err := client.GetTimesheet(id)
			if err != nil {
				return err
			}
			if dryRun {
				return writeDryRun(cmd.OutOrStdout(), format, client, "update", mergeFields(*existing, fields))
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

	cmd.Flags().Int64Var(&ops.projectID, "project-id", 0, "new project `ID`")
	cmd.Flags().Int64Var(&ops.taskID, "task-id", 0, "new task `ID`")
	cmd.Flags().StringVar(&ops.date, "date", "", "new day, `YYYY-MM-DD`")
	cmd.Flags().StringVar(&ops.hours, "hours", "", "new hours, decimal (2.5) or `H`:MM (2:30)")
	cmd.Flags().StringVar(&ops.description, "description", "", "new description `TEXT`")

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, dryRunFlagUsage)

	return cmd
}
