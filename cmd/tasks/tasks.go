package tasks

import (
	"fmt"
	"strconv"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func CMD(deps *app.Deps) *cobra.Command {
	var search string
	cmd := &cobra.Command{
		Use:   "tasks [project-id]",
		Short: "List Odoo tasks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}
			client, err := deps.RequireClient()
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
	cmd.Flags().StringVar(&search, "search", "", "only tasks whose name contains TEXT (case-insensitive)")
	output.AddFlags(cmd.Flags())
	return cmd
}
