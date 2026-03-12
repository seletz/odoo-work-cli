package fields

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/spf13/cobra"
)

var tsWeek string

func CMD(client *odoo.XMLRPCClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fields <model>",
		Short: "Inspect Odoo model fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fields, err := client.GetFields(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("%-30s %-15s %-30s %s\n", "Field", "Type", "Label", "Required")
			fmt.Printf("%-30s %-15s %-30s %s\n", "------------------------------", "---------------", "------------------------------", "--------")
			for _, f := range fields {
				fmt.Printf("%-30s %-15s %-30s %v\n", f.Name, f.Type, f.String, f.Required)
			}
			return nil
		},
	}

	return cmd
}
