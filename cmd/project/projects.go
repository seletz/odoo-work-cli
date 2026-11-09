package project

import (
	"fmt"
	"strings"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func CMD(deps *app.Deps) *cobra.Command {
	var search string
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List Odoo projects",
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
	cmd.Flags().StringVar(&search, "search", "", "only projects whose name or customer contains TEXT (case-insensitive)")
	output.AddFlags(cmd.Flags())
	return cmd
}
