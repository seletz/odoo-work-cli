package timesheet

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/seletz/odoo-work-cli/internal/parsing"
	"github.com/spf13/cobra"
)

func CMD(deps *app.Deps) *cobra.Command {
	var tsWeek string

	cmd := &cobra.Command{
		Use:   "timesheets",
		Short: "List Odoo timesheets for a week",
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

	cmd.Flags().StringVar(&tsWeek, "week", "", "ISO week (e.g. 2026-W10), defaults to current week")

	output.AddFlags(cmd.Flags())
	return cmd
}
