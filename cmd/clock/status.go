package clock

import (
	"fmt"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/spf13/cobra"
)

func statusCMD(client *odoo.XMLRPCClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current attendance status",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := client.AttendanceStatus()
			if err != nil {
				return err
			}

			if status.ClockedIn && status.CheckIn != nil {
				elapsed := time.Since(*status.CheckIn).Hours()
				fmt.Printf("Status: Clocked in since %s (%s elapsed)\n",
					status.CheckIn.Local().Format("15:04"),
					tui.FormatHours(elapsed))
			} else {
				fmt.Println("Status: Not clocked in")
			}

			if len(status.Periods) > 0 {
				fmt.Print("\nToday's attendance:\n\n")
				fmt.Printf("%-3s %-10s %-10s %s\n", "#", "Check In", "Check Out", "Duration")
				fmt.Printf("%-3s %-10s %-10s %s\n", "---", "----------", "----------", "--------")
				for i, p := range status.Periods {
					checkIn := p.CheckIn.Local().Format("15:04")
					var checkOut, duration string
					if p.CheckOut != nil {
						checkOut = p.CheckOut.Local().Format("15:04")
						duration = tui.FormatHours(p.WorkedHours)
					} else {
						checkOut = "--:--"
						elapsed := time.Since(p.CheckIn).Hours()
						duration = tui.FormatHours(elapsed) + " (running)"
					}
					fmt.Printf("%-3d %-10s %-10s %s\n", i+1, checkIn, checkOut, duration)
				}
				fmt.Printf("\nTotal: %s\n", tui.FormatHours(status.TotalHours))
			} else {
				fmt.Println("\nNo attendance records today.")
			}
			return nil
		},
	}
	return cmd
}
