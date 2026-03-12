package entries

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/spf13/cobra"
)

func deleteCmd(client *odoo.XMLRPCClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a timesheet entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseEntryID(args[0])
			if err != nil {
				return err
			}

			if err := client.DeleteTimesheet(id); err != nil {
				return err
			}

			fmt.Printf("Deleted entry %d\n", id)
			return nil
		},
	}
	return cmd
}
