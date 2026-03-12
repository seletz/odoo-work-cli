package main

import (
	"fmt"
	"os"

	"github.com/seletz/odoo-work-cli/cmd/clock"
	configcmd "github.com/seletz/odoo-work-cli/cmd/config"
	"github.com/seletz/odoo-work-cli/cmd/entries"
	"github.com/seletz/odoo-work-cli/cmd/fields"
	"github.com/seletz/odoo-work-cli/cmd/project"
	"github.com/seletz/odoo-work-cli/cmd/tasks"
	"github.com/seletz/odoo-work-cli/cmd/timesheet"
	"github.com/seletz/odoo-work-cli/cmd/tui"
	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"

	"github.com/seletz/odoo-work-cli/internal/version"
	"github.com/spf13/cobra"
)

func main() {
	var cfgFile string
	deps := &app.Deps{}

	rootCmd := &cobra.Command{
		Use:     "odoo-work-cli",
		Short:   "CLI for managing Odoo 17 timesheets",
		Version: version.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if isNoneSetupCommand(cmd) {
				return nil
			}

			cfg, err := loadConfig(cfgFile)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			client, err := newClient(cfg)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			deps.Config = cfg
			deps.Client = client
			return nil
		},

		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if deps.Client != nil {
				deps.Client.Close()
				deps.Client = nil
			}
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (skip discovery)")

	rootCmd.AddCommand(project.CMD(deps))
	rootCmd.AddCommand(tasks.CMD(deps))
	rootCmd.AddCommand(timesheet.CMD(deps))
	rootCmd.AddCommand(fields.CMD(deps))
	rootCmd.AddCommand(whoamiCMD(deps))
	rootCmd.AddCommand(configcmd.CMD(&cfgFile))
	rootCmd.AddCommand(tui.CMD(deps))
	rootCmd.AddCommand(entries.CMD(deps))
	rootCmd.AddCommand(clock.CMD(deps))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func isNoneSetupCommand(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		switch current.Name() {
		case "completion", "__complete", "__completeNoDesc":
			return true
		case "config":
			return true
		}
	}
	return false
}

// loadConfig loads and merges config using file discovery and env vars.
func loadConfig(cfgFile string) (*config.Config, error) {
	result, err := config.Discover(cfgFile)
	if err != nil {
		return nil, err
	}
	if err := result.Config.Validate(); err != nil {
		return nil, err
	}
	return result.Config, nil
}

// newClient creates a new Odoo client from the merged config.
func newClient(cfg *config.Config) (*odoo.XMLRPCClient, error) {
	return odoo.NewXMLRPCClient(cfg.URL, cfg.Database, cfg.Username, cfg.Password, cfg.WebPassword, cfg.TOTPSecret, cfg.Models)
}

func whoamiCMD(deps *app.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current Odoo user info",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			info, err := client.WhoAmI()
			if err != nil {
				return err
			}
			fmt.Printf("ID:      %d\n", info.ID)
			fmt.Printf("Name:    %s\n", info.Name)
			fmt.Printf("Login:   %s\n", info.Login)
			fmt.Printf("Email:   %s\n", info.Email)
			fmt.Printf("Company: %s\n", info.Company)
			return nil
		},
	}
}
