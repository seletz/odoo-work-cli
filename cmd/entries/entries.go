package entries

import (
	"fmt"
	"strconv"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/filter"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/parsing"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/spf13/cobra"
)

type entrieOps struct {
	week    string
	date    string
	project string
	task    string
	status  string
}

type subOps struct {
	projectID   int64
	taskID      int64
	date        string
	hours       string
	description string
}

func CMD(deps *app.Deps) *cobra.Command {
	ops := &entrieOps{}

	cmd := &cobra.Command{
		Use:   "entries",
		Short: "List individual timesheet entries with full detail",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			var dateFrom, dateTo string
			if ops.date != "" {
				dateFrom, dateTo, err = parsing.ParseDateRange(ops.date)
			} else {
				dateFrom, dateTo, err = parsing.WeekDateRange(ops.week)
			}
			if err != nil {
				return err
			}

			entries, err := client.ListTimesheets(dateFrom, dateTo)
			if err != nil {
				return err
			}

			entries = filter.Entries(entries, ops.project, ops.task, ops.status)

			if ops.date != "" {
				fmt.Printf("Date: %s\n\n", dateFrom)
			} else {
				fmt.Printf("Week: %s to %s\n\n", dateFrom, dateTo)
			}

			fmt.Printf("%-8s %-12s %-8s %-25s %-8s %-25s %-6s %-10s %s\n",
				"ID", "Date", "ProjID", "Project", "TaskID", "Task", "Hours", "Status", "Description")
			fmt.Printf("%-8s %-12s %-8s %-25s %-8s %-25s %-6s %-10s %s\n",
				"--------", "------------", "--------", "-------------------------", "--------", "-------------------------", "------", "----------", "------------------------------")

			var total float64
			for _, e := range entries {
				fmt.Printf("%-8d %-12s %-8d %-25s %-8d %-25s %-6s %-10s %s\n",
					e.ID, e.Date, e.ProjectID, e.Project, e.TaskID, e.Task, tui.FormatHours(e.Hours), e.ValidatedStatus, e.Name)
				total += e.Hours
			}

			fmt.Printf("\nTotal: %s (%d entries)\n", tui.FormatHours(total), len(entries))
			return nil
		},
	}

	// Set Flags
	cmd.Flags().StringVar(&ops.week, "week", "", "ISO week (e.g. 2026-W10), defaults to current week")
	cmd.Flags().StringVar(&ops.date, "date", "", "specific date (YYYY-MM-DD), overrides --week")
	cmd.Flags().StringVar(&ops.project, "project", "", "filter by project name (substring, case-insensitive)")
	cmd.Flags().StringVar(&ops.task, "task", "", "filter by task name (substring, case-insensitive)")
	cmd.Flags().StringVar(&ops.status, "status", "", "filter by validation status (e.g. draft, validated)")

	// Set Subcommands
	cmd.AddCommand(addCmd(deps))
	cmd.AddCommand(updateCmd(deps))
	cmd.AddCommand(deleteCmd(deps))

	return cmd
}

// buildTimesheetWriteParams constructs and validates TimesheetWriteParams from CLI flag values.
// An empty date defaults to today. Hours accepts both decimal ("2.5") and H:MM ("2:30") formats.
func buildTimesheetWriteParams(projectID, taskID int64, date, description, hoursStr string) (odoo.TimesheetWriteParams, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	} else {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return odoo.TimesheetWriteParams{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", date)
		}
	}
	hours, err := tui.ParseHours(hoursStr)
	if err != nil {
		return odoo.TimesheetWriteParams{}, err
	}
	p := odoo.TimesheetWriteParams{
		ProjectID: projectID,
		TaskID:    taskID,
		Date:      date,
		Name:      description,
		Hours:     hours,
	}
	if err := odoo.ValidateTimesheetParams(p); err != nil {
		return odoo.TimesheetWriteParams{}, err
	}
	return p, nil
}

// buildUpdateFields builds a partial Odoo field map from the flags that were
// explicitly set on the command. Returns an error if a set flag has an invalid value.
func buildUpdateFields(cmd *cobra.Command, ops *subOps) (map[string]interface{}, error) {
	fields := make(map[string]interface{})
	if cmd.Flags().Changed("project-id") {
		fields["project_id"] = ops.projectID
	}
	if cmd.Flags().Changed("task-id") {
		fields["task_id"] = ops.taskID
	}
	if cmd.Flags().Changed("date") {
		if _, err := time.Parse("2006-01-02", ops.date); err != nil {
			return nil, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", ops.date)
		}
		fields["date"] = ops.date
	}
	if cmd.Flags().Changed("hours") {
		h, err := tui.ParseHours(ops.hours)
		if err != nil {
			return nil, err
		}
		fields["unit_amount"] = h
	}
	if cmd.Flags().Changed("description") {
		if ops.description == "" {
			return nil, fmt.Errorf("description must not be empty")
		}
		fields["name"] = ops.description
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("at least one flag is required (--project-id, --task-id, --date, --hours, --description)")
	}
	return fields, nil
}

// parseEntryID parses and validates a timesheet entry ID from a string.
func parseEntryID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid entry ID %q: must be a positive integer", s)
	}
	if id <= 0 {
		return 0, fmt.Errorf("invalid entry ID %q: must be a positive integer", s)
	}
	return id, nil
}
