// Command odoo-work-cli manages Odoo 17 timesheets and attendance.
package main

import (
	"os"

	"github.com/seletz/odoo-work-cli/internal/cli"
)

func main() {
	os.Exit(cli.Main())
}
