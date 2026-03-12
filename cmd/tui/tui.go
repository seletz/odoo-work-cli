package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/spf13/cobra"
)

var tuiWeek string

func CMD(client *odoo.XMLRPCClient, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Interactive weekly timesheet view",
		RunE: func(cmd *cobra.Command, args []string) error {

			monday, err := tui.ParseWeekMonday(tuiWeek)
			if err != nil {
				return err
			}

			m := tui.NewModel(client, tui.MondayTime{Time: monday}, cfg.Hours, cfg.Bundesland, cfg.Keys, cfg.CompanyColors)
			p := tea.NewProgram(m)
			_, err = p.Run()
			return err
		},
	}

	cmd.Flags().StringVar(&tuiWeek, "week", "", "ISO week (e.g. 2026-W10), defaults to current week")

	return cmd
}
