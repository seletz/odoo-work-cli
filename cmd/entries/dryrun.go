package entries

import (
	"fmt"
	"io"
	"strings"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/seletz/odoo-work-cli/internal/parsing"
	"github.com/seletz/odoo-work-cli/internal/tui"
)

const dryRunFlagUsage = "validate and print the entry that would be written, without writing it"

// resolveProposal resolves the project and task IDs of e against Odoo, so an
// unknown ID fails here with the same kind (validation) as the real write,
// and fills in the project name, task name and company. A task must belong
// to the project; a task-only entry takes the task's project.
func resolveProposal(client odoo.Client, e odoo.TimesheetEntry) (odoo.TimesheetEntry, error) {
	if e.TaskID > 0 {
		tasks, err := client.ListAllTasks(0)
		if err != nil {
			return e, err
		}
		var task *odoo.TaskInfo
		for i := range tasks {
			if tasks[i].ID == e.TaskID {
				task = &tasks[i]
				break
			}
		}
		if task == nil {
			return e, apperr.New(apperr.KindValidation, "task %d does not exist", e.TaskID)
		}
		if e.ProjectID <= 0 {
			e.ProjectID = task.ProjectID
		}
		if task.ProjectID != e.ProjectID {
			return e, apperr.New(apperr.KindValidation, "task %d belongs to project %d, not to project %d", e.TaskID, task.ProjectID, e.ProjectID)
		}
		e.Task = task.Name
	} else {
		e.Task = ""
	}

	projects, err := client.ListAllProjects()
	if err != nil {
		return e, err
	}
	for _, p := range projects {
		if p.ID == e.ProjectID {
			e.Project = p.Name
			e.Company = p.Company
			return e, nil
		}
	}
	return e, apperr.New(apperr.KindValidation, "project %d does not exist", e.ProjectID)
}

// findDuplicates returns the entries of the same day with the same project,
// task and description as e (other than e itself).
func findDuplicates(client odoo.Client, e odoo.TimesheetEntry) ([]odoo.TimesheetEntry, error) {
	from, to, err := parsing.ParseDateRange(e.Date)
	if err != nil {
		return nil, apperr.Wrap(apperr.KindValidation, err)
	}
	day, err := client.ListAllTimesheets(from, to)
	if err != nil {
		return nil, err
	}
	var dups []odoo.TimesheetEntry
	for _, d := range day {
		if d.ID != e.ID && d.Date == e.Date && d.ProjectID == e.ProjectID && d.TaskID == e.TaskID &&
			strings.TrimSpace(d.Name) == strings.TrimSpace(e.Name) {
			dups = append(dups, d)
		}
	}
	return dups, nil
}

// writeDryRun resolves and validates the proposal, then prints it. It never
// writes. verb is "create" or "update" for the table output.
func writeDryRun(w io.Writer, format output.Format, client odoo.Client, verb string, proposal odoo.TimesheetEntry) error {
	proposal, err := resolveProposal(client, proposal)
	if err != nil {
		return err
	}
	dups, err := findDuplicates(client, proposal)
	if err != nil {
		return err
	}
	if format.IsMachine() {
		return output.WriteObject(w, format, output.NewEntryDryRun(proposal, dups))
	}

	_, _ = fmt.Fprintf(w, "Dry run: would %s this entry; nothing was written.\n", verb)
	if proposal.ID > 0 {
		_, _ = fmt.Fprintf(w, "  ID:          %d\n", proposal.ID)
	}
	_, _ = fmt.Fprintf(w, "  Date:        %s\n", proposal.Date)
	_, _ = fmt.Fprintf(w, "  Project:     %s (%d)\n", proposal.Project, proposal.ProjectID)
	if proposal.TaskID > 0 {
		_, _ = fmt.Fprintf(w, "  Task:        %s (%d)\n", proposal.Task, proposal.TaskID)
	}
	_, _ = fmt.Fprintf(w, "  Hours:       %s\n", tui.FormatHours(proposal.Hours))
	_, _ = fmt.Fprintf(w, "  Description: %s\n", proposal.Name)
	for _, d := range dups {
		_, _ = fmt.Fprintf(w, "Warning: possible duplicate of entry %d (same day, project, task and description)\n", d.ID)
	}
	return nil
}

// mergeFields applies the Odoo field map of an update (see buildUpdateFields)
// to the existing entry and returns the result. The project and task names
// are left for resolveProposal to refresh.
func mergeFields(e odoo.TimesheetEntry, fields map[string]interface{}) odoo.TimesheetEntry {
	if v, ok := fields["project_id"].(int64); ok {
		e.ProjectID = v
	}
	if v, ok := fields["task_id"].(int64); ok {
		e.TaskID = v
	}
	if v, ok := fields["date"].(string); ok {
		e.Date = v
	}
	if v, ok := fields["unit_amount"].(float64); ok {
		e.Hours = v
	}
	if v, ok := fields["name"].(string); ok {
		e.Name = v
	}
	return e
}
