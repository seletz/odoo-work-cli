package timesheet

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/parsing"
	"github.com/spf13/cobra"
)

var tsWeek string

func CMD(deps *app.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timesheets",
		Short: "List Odoo timesheets for a week",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			dateFrom, dateTo, err := parsing.WeekDateRange(tsWeek)
			if err != nil {
				return err
			}

			entries, err := client.ListTimesheets(dateFrom, dateTo)
			if err != nil {
				return err
			}
			fmt.Printf("Week: %s to %s\n\n", dateFrom, dateTo)
			fmt.Printf("%-8s %-12s %-8s %-25s %-8s %-25s %-30s %s\n", "ID", "Date", "ProjID", "Project", "TaskID", "Task", "Description", "Hours")
			fmt.Printf("%-8s %-12s %-8s %-25s %-8s %-25s %-30s %s\n",
				"--------", "------------", "--------", "-------------------------", "--------", "-------------------------", "------------------------------", "-----")
			var total float64
			for _, e := range entries {
				fmt.Printf("%-8d %-12s %-8d %-25s %-8d %-25s %-30s %.2f\n", e.ID, e.Date, e.ProjectID, e.Project, e.TaskID, e.Task, e.Name, e.Hours)
				total += e.Hours
			}
			fmt.Printf("\nTotal: %.2f hours\n", total)
			return nil
		},
	}

	cmd.Flags().StringVar(&tsWeek, "week", "", "ISO week (e.g. 2026-W10), defaults to current week")

	return cmd
}
