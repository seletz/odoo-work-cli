package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/parsing"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/spf13/cobra"
)

func CMD(deps *app.Deps) *cobra.Command {
	var tuiWeek string

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Interactive weekly timesheet view",
		Long: `Open the interactive weekly grid of your timesheet entries for an ISO
week (--week, default the current week): one row per project and task,
one column per day, with the totals of each day and of the week. Totals
are coloured by the [hours] limits of the configuration, German public
holidays of bundesland are marked, and today's attendance is shown and
updated.

Entries can be added, edited and deleted from the grid, and you can clock
in and out. Press ? for the keys; [keys] in the configuration rebinds
them. The TUI needs a terminal and takes over the screen until you quit.`,
		Example: `  # The current week
  odoo-work-cli tui

  # A past week
  odoo-work-cli tui --week 2026-W14`,
		Annotations: map[string]string{
			manual.Group:          manual.GroupTimesheets,
			manual.StaticExamples: "true",
			manual.SeeAlso:        "tutorials/30-tui-tour.md, reference/50-tui-keys.md, how-to/90-customise-tui.md",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}
			cfg, err := deps.RequireConfig()
			if err != nil {
				return err
			}

			monday, err := parsing.ParseWeekMonday(tuiWeek)
			if err != nil {
				return err
			}

			m := tui.NewModel(client, tui.MondayTime{Time: monday}, cfg.Hours, cfg.Bundesland, cfg.Keys, cfg.CompanyColors)
			p := tea.NewProgram(m)
			_, err = p.Run()
			return err
		},
	}

	cmd.Flags().StringVar(&tuiWeek, "week", "", "ISO week to open, `YYYY-Www` (e.g. 2026-W10)")
	manual.SetFlagDefault(cmd.Flags(), "week", "the current week")

	return cmd
}
