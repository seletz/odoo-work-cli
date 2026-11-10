package tasks

import (
	"fmt"
	"strconv"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func CMD(deps *app.Deps) *cobra.Command {
	var search string
	cmd := &cobra.Command{
		Use:   "tasks [PROJECT-ID]",
		Short: "List Odoo tasks",
		Long: `List the Odoo tasks visible to you: those of the project PROJECT-ID, or
of all projects without it. The [models.task] filters of the
configuration restrict the list.

--search TEXT keeps the tasks whose name contains TEXT, ignoring case. The
ID column is what entries add --task-id takes. A PROJECT-ID that is not an
integer is a validation error.`,
		Example: `  # The tasks of project 1
  odoo-work-cli tasks 1

  # Tasks named like "review" in any project, as JSON
  odoo-work-cli tasks --search review --json`,
		Annotations: map[string]string{
			manual.Group: manual.GroupProjects,
			manual.Output: `The table has the columns ID, Name, Project and Stage. With --search and
no match the command prints "No tasks match TEXT" and exits with 0.

With --json the output is an array of tasks, [] if there are none.`,
			manual.Schemas: "task",
			manual.SeeAlso: "odoo-work-cli projects, how-to/40-find-projects-and-tasks.md",
		},
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}

			var projectID int64
			if len(args) == 1 {
				projectID, err = strconv.ParseInt(args[0], 10, 64)
				if err != nil {
					return apperr.New(apperr.KindValidation, "invalid project ID %q: must be an integer", args[0])
				}
			}
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			var tasks []odoo.TaskInfo
			if search != "" {
				tasks, err = client.SearchTasks(projectID, search)
			} else {
				tasks, err = client.ListTasks(projectID)
			}
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if format.IsMachine() {
				return output.WriteList(w, format, output.NewTasks(tasks))
			}
			if len(tasks) == 0 && search != "" {
				_, _ = fmt.Fprintf(w, "No tasks match %q\n", search)
				return nil
			}
			_, _ = fmt.Fprintf(w, "%-8s %-40s %-30s %s\n", "ID", "Name", "Project", "Stage")
			_, _ = fmt.Fprintf(w, "%-8s %-40s %-30s %s\n", "--------", "----------------------------------------", "------------------------------", "--------------------")
			for _, t := range tasks {
				_, _ = fmt.Fprintf(w, "%-8d %-40s %-30s %s\n", t.ID, t.Name, t.Project, t.Stage)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&search, "search", "", "only tasks whose name contains `TEXT` (case-insensitive)")
	output.AddFlags(cmd.Flags())
	return cmd
}
