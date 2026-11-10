package entries

import (
	"fmt"
	"strconv"
	"time"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/filter"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
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
		Long: `List your timesheet entries of an ISO week (--week, default the current
week) or of one day (--date, which takes precedence over --week), with the
validation status of each entry and the total.

--project and --task keep the entries whose project or task name contains
the text, ignoring case; --status keeps the entries with exactly that
validation status (draft or validated). The filters combine. The
[models.timesheet] filters of the configuration apply as well.

The subcommands add, update and delete write entries.`,
		Example: `  # The entries of one day
  odoo-work-cli entries --date 2026-04-08

  # The draft entries of a week on projects named like "portal", as JSON
  odoo-work-cli entries --week 2026-W15 --project portal --status draft --json`,
		Annotations: map[string]string{
			manual.Group: manual.GroupTimesheets,
			manual.Synopsis: `odoo-work-cli entries [--week YYYY-Www | --date YYYY-MM-DD] [--project TEXT] [--task TEXT] [--status STATUS] [--json | --format FORMAT]
odoo-work-cli entries add|update|delete [OPTIONS]`,
			manual.Output: `A "Week: FROM to TO" or "Date: DAY" line, then the columns ID, Date,
ProjID, Project, TaskID, Task, Hours (H:MM), Status and Description, then
the total and the number of entries.

With --json the output is an array of entries, [] if none match.`,
			manual.Schemas: "entry",
			manual.SeeAlso: "odoo-work-cli timesheets, how-to/20-book-time.md",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
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
				return apperr.Wrap(apperr.KindValidation, err)
			}
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			entries, err := client.ListTimesheets(dateFrom, dateTo)
			if err != nil {
				return err
			}

			entries = filter.Entries(entries, ops.project, ops.task, ops.status)

			w := cmd.OutOrStdout()
			if format.IsMachine() {
				return output.WriteList(w, format, output.NewEntries(entries))
			}

			if ops.date != "" {
				_, _ = fmt.Fprintf(w, "Date: %s\n\n", dateFrom)
			} else {
				_, _ = fmt.Fprintf(w, "Week: %s to %s\n\n", dateFrom, dateTo)
			}

			_, _ = fmt.Fprintf(w, "%-8s %-12s %-8s %-25s %-8s %-25s %-6s %-10s %s\n",
				"ID", "Date", "ProjID", "Project", "TaskID", "Task", "Hours", "Status", "Description")
			_, _ = fmt.Fprintf(w, "%-8s %-12s %-8s %-25s %-8s %-25s %-6s %-10s %s\n",
				"--------", "------------", "--------", "-------------------------", "--------", "-------------------------", "------", "----------", "------------------------------")

			var total float64
			for _, e := range entries {
				_, _ = fmt.Fprintf(w, "%-8d %-12s %-8d %-25s %-8d %-25s %-6s %-10s %s\n",
					e.ID, e.Date, e.ProjectID, e.Project, e.TaskID, e.Task, tui.FormatHours(e.Hours), e.ValidatedStatus, e.Name)
				total += e.Hours
			}

			_, _ = fmt.Fprintf(w, "\nTotal: %s (%d entries)\n", tui.FormatHours(total), len(entries))
			return nil
		},
	}

	// Set Flags
	cmd.Flags().StringVar(&ops.week, "week", "", "ISO week to list, `YYYY-Www` (e.g. 2026-W10)")
	cmd.Flags().StringVar(&ops.date, "date", "", "single day to list, `YYYY-MM-DD`; overrides --week")
	cmd.Flags().StringVar(&ops.project, "project", "", "only entries whose project name contains `TEXT` (case-insensitive)")
	cmd.Flags().StringVar(&ops.task, "task", "", "only entries whose task name contains `TEXT` (case-insensitive)")
	cmd.Flags().StringVar(&ops.status, "status", "", "only entries with validation `STATUS` draft or validated")
	manual.SetFlagDefault(cmd.Flags(), "week", "the current week")

	// --format/--json apply to the list and to add/update/delete.
	output.AddFlags(cmd.PersistentFlags())

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
			return odoo.TimesheetWriteParams{}, apperr.New(apperr.KindValidation, "invalid date %q: expected YYYY-MM-DD", date)
		}
	}
	hours, err := tui.ParseHours(hoursStr)
	if err != nil {
		return odoo.TimesheetWriteParams{}, apperr.Wrap(apperr.KindValidation, err)
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
			return nil, apperr.New(apperr.KindValidation, "invalid date %q: expected YYYY-MM-DD", ops.date)
		}
		fields["date"] = ops.date
	}
	if cmd.Flags().Changed("hours") {
		h, err := tui.ParseHours(ops.hours)
		if err != nil {
			return nil, apperr.Wrap(apperr.KindValidation, err)
		}
		fields["unit_amount"] = h
	}
	if cmd.Flags().Changed("description") {
		if ops.description == "" {
			return nil, apperr.New(apperr.KindValidation, "description must not be empty")
		}
		fields["name"] = ops.description
	}
	if len(fields) == 0 {
		return nil, apperr.New(apperr.KindValidation, "at least one flag is required (--project-id, --task-id, --date, --hours, --description)")
	}
	return fields, nil
}

// parseEntryID parses and validates a timesheet entry ID from a string.
func parseEntryID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, apperr.New(apperr.KindValidation, "invalid entry ID %q: must be a positive integer", s)
	}
	if id <= 0 {
		return 0, apperr.New(apperr.KindValidation, "invalid entry ID %q: must be a positive integer", s)
	}
	return id, nil
}
