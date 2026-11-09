package entries

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func deleteCmd(deps *app.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a timesheet entry",
		Long: `Delete a timesheet entry.

With --json (or --format json|jsonl) the entry as it was before deletion
is printed.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}
			id, err := parseEntryID(args[0])
			if err != nil {
				return err
			}
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			// Read first: a missing entry fails as not_found before any
			// write, and --json reports the deleted object.
			entry, err := client.GetTimesheet(id)
			if err != nil {
				return err
			}
			if err := client.DeleteTimesheet(id); err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			if format.IsMachine() {
				return output.WriteObject(w, format, output.NewEntry(*entry))
			}
			_, _ = fmt.Fprintf(w, "Deleted entry %d\n", id)
			return nil
		},
	}
	return cmd
}
