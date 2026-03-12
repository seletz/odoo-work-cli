package tasks

import (
	"fmt"
	"strconv"

	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/spf13/cobra"
)

func CMD(client *odoo.XMLRPCClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks [project-id]",
		Short: "List Odoo tasks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var projectID int64
			if len(args) == 1 {
				projectID, err = strconv.ParseInt(args[0], 10, 64)
				if err != nil {
					return fmt.Errorf("invalid project ID: %w", err)
				}
			}

			tasks, err := client.ListTasks(projectID)
			if err != nil {
				return err
			}
			fmt.Printf("%-8s %-40s %-30s %s\n", "ID", "Name", "Project", "Stage")
			fmt.Printf("%-8s %-40s %-30s %s\n", "--------", "----------------------------------------", "------------------------------", "--------------------")
			for _, t := range tasks {
				fmt.Printf("%-8d %-40s %-30s %s\n", t.ID, t.Name, t.Project, t.Stage)
			}
			return nil
		},
	}
	return cmd
}
