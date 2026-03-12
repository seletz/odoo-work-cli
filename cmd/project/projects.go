package project

import (
	"fmt"
	"strings"

	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/spf13/cobra"
)

func CMD(client *odoo.XMLRPCClient, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List Odoo projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			projects, err := client.ListProjects()
			if err != nil {
				return err
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
			fmt.Println(header)
			fmt.Println(sep)

			for _, p := range projects {
				line := fmt.Sprintf("%-6d %-30s %-20s %-15s %-15s %-20s",
					p.ID, p.Name, p.Customer, p.Company, p.Stage, p.ProjectManager)
				for _, name := range extraNames {
					line += fmt.Sprintf(" %-20s", p.ExtraFields[name])
				}
				line += fmt.Sprintf(" %v", p.Active)
				fmt.Println(line)
			}
			return nil
		},
	}
	return cmd
}
