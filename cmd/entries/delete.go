package entries

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

func deleteCmd(deps *app.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a timesheet entry",
		Long: `Delete the timesheet entry ID. The entry is read first: an unknown ID
fails with not_found and nothing is deleted. There is no confirmation
prompt and no undo. Odoo may refuse to delete validated entries (access_denied).`,
		Example: `  # Delete an entry
  odoo-work-cli entries delete 4019

  # An unknown ID is a not_found error
  odoo-work-cli entries delete 999 --json`,
		Annotations: map[string]string{
			manual.Output:  "Prints \"Deleted entry ID\".\n\nWith --json the entry as it was before the deletion.",
			manual.Schemas: "entry",
			manual.Errors:  "not_found, access_denied",
			manual.SeeAlso: "odoo-work-cli entries",
		},
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
