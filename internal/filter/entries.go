package filter

import (
	"strings"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// Entries returns entries matching the project, task, and status filters.
// Project and task use case-insensitive substring match. Status uses exact match.
// Empty filter matches all.
func Entries(entries []odoo.TimesheetEntry, project, task, status string) []odoo.TimesheetEntry {
	if project == "" && task == "" && status == "" {
		return entries
	}
	projectLower := strings.ToLower(project)
	taskLower := strings.ToLower(task)
	var result []odoo.TimesheetEntry
	for _, e := range entries {
		if project != "" && !strings.Contains(strings.ToLower(e.Project), projectLower) {
			continue
		}
		if task != "" && !strings.Contains(strings.ToLower(e.Task), taskLower) {
			continue
		}
		if status != "" && e.ValidatedStatus != status {
			continue
		}
		result = append(result, e)
	}
	return result
}
