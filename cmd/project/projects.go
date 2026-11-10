package project

import (
	"fmt"
	"strings"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func CMD(deps *app.Deps) *cobra.Command {
	var search string
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List Odoo projects",
		Long: `List the Odoo projects visible to you. The [models.project] filters of
the configuration restrict the list; its extra_fields add columns (and
the extra_fields object of the JSON output).

--search TEXT keeps the projects whose name or customer contains TEXT,
ignoring case. The ID column is what entries add --project-id and the
argument of tasks take.`,
		Example: `  # All projects
  odoo-work-cli projects

  # Projects whose name or customer contains "portal", as JSON
  odoo-work-cli projects --search portal --json`,
		Annotations: map[string]string{
			manual.Group: manual.GroupProjects,
			manual.Output: `The table has the columns ID, Name, Customer, Company, Phase (the
project stage), Project Manager, one column per extra field of
[models.project], and Active. With --search and no match the command
prints "No projects match TEXT" and exits with 0.

With --json the output is an array of projects, [] if there are none.`,
			manual.Schemas: "project",
			manual.SeeAlso: "odoo-work-cli tasks, how-to/40-find-projects-and-tasks.md",
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
			cfg, err := deps.RequireConfig()
			if err != nil {
				return err
			}

			var projects []odoo.ProjectInfo
			if search != "" {
				projects, err = client.SearchProjects(search)
			} else {
				projects, err = client.ListProjects()
			}
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if format.IsMachine() {
				return output.WriteList(w, format, output.NewProjects(projects))
			}

			if len(projects) == 0 && search != "" {
				_, _ = fmt.Fprintf(w, "No projects match %q\n", search)
				return nil
			}

			// Build dynamic column headers from config extra fields.
			var extraNames []string
			if mc, ok := cfg.Models["project"]; ok {
				for _, ef := range mc.ExtraFields {
					extraNames = append(extraNames, ef.Name)
				}
			}

			// Print header.
			header := fmt.Sprintf("%-6s %-30s %-20s %-15s %-15s %-20s",
				"ID", "Name", "Customer", "Company", "Phase", "Project Manager")
			sep := fmt.Sprintf("%-6s %-30s %-20s %-15s %-15s %-20s",
				"------", "------------------------------", "--------------------", "---------------", "---------------", "--------------------")
			for _, name := range extraNames {
				label := strings.ReplaceAll(name, "_", " ")
				header += fmt.Sprintf(" %-20s", label)
				sep += fmt.Sprintf(" %-20s", "--------------------")
			}
			header += fmt.Sprintf(" %s", "Active")
			sep += fmt.Sprintf(" %s", "------")
			_, _ = fmt.Fprintln(w, header)
			_, _ = fmt.Fprintln(w, sep)

			for _, p := range projects {
				line := fmt.Sprintf("%-6d %-30s %-20s %-15s %-15s %-20s",
					p.ID, p.Name, p.Customer, p.Company, p.Stage, p.ProjectManager)
				for _, name := range extraNames {
					line += fmt.Sprintf(" %-20s", p.ExtraFields[name])
				}
				line += fmt.Sprintf(" %v", p.Active)
				_, _ = fmt.Fprintln(w, line)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&search, "search", "", "only projects whose name or customer contains `TEXT` (case-insensitive)")
	output.AddFlags(cmd.Flags())
	return cmd
}
