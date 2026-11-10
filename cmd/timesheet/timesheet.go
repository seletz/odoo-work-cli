package timesheet

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/seletz/odoo-work-cli/internal/parsing"
	"github.com/spf13/cobra"
)

func CMD(deps *app.Deps) *cobra.Command {
	var tsWeek string

	cmd := &cobra.Command{
		Use:   "timesheets",
		Short: "List Odoo timesheets for a week",
		Long: `List your timesheet entries of an ISO week, Monday to Sunday, with the
total of the week. Without --week, the current week. The
[models.timesheet] filters of the configuration apply.

For one day, for filters by project, task or validation status, and for
the status column, use entries.`,
		Example: `  # The entries of week 15 of 2026
  odoo-work-cli timesheets --week 2026-W15`,
		Annotations: map[string]string{
			manual.Group: manual.GroupTimesheets,
			manual.Output: `A "Week: FROM to TO" line, then the columns ID, Date, ProjID, Project,
TaskID, Task, Description and Hours (decimal), then the total in hours.

With --json the output is an array of entries, [] for an empty week.`,
			manual.Schemas: "entry",
			manual.SeeAlso: "odoo-work-cli entries, odoo-work-cli tui",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			dateFrom, dateTo, err := parsing.WeekDateRange(tsWeek)
			if err != nil {
				return apperr.Wrap(apperr.KindValidation, err)
			}

			entries, err := client.ListTimesheets(dateFrom, dateTo)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if format.IsMachine() {
				return output.WriteList(w, format, output.NewEntries(entries))
			}
			_, _ = fmt.Fprintf(w, "Week: %s to %s\n\n", dateFrom, dateTo)
			_, _ = fmt.Fprintf(w, "%-8s %-12s %-8s %-25s %-8s %-25s %-30s %s\n", "ID", "Date", "ProjID", "Project", "TaskID", "Task", "Description", "Hours")
			_, _ = fmt.Fprintf(w, "%-8s %-12s %-8s %-25s %-8s %-25s %-30s %s\n",
				"--------", "------------", "--------", "-------------------------", "--------", "-------------------------", "------------------------------", "-----")
			var total float64
			for _, e := range entries {
				_, _ = fmt.Fprintf(w, "%-8d %-12s %-8d %-25s %-8d %-25s %-30s %.2f\n", e.ID, e.Date, e.ProjectID, e.Project, e.TaskID, e.Task, e.Name, e.Hours)
				total += e.Hours
			}
			_, _ = fmt.Fprintf(w, "\nTotal: %.2f hours\n", total)
			return nil
		},
	}

	cmd.Flags().StringVar(&tsWeek, "week", "", "ISO week to list, `YYYY-Www` (e.g. 2026-W10)")
	manual.SetFlagDefault(cmd.Flags(), "week", "the current week")

	output.AddFlags(cmd.Flags())
	return cmd
}
