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
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"

	"github.com/seletz/odoo-work-cli/internal/version"
	"github.com/spf13/cobra"
)

var cfgFile string
var cfg *config.Config
var client *odoo.XMLRPCClient

func main() {

	rootCmd := &cobra.Command{
		Use:     "odoo-work-cli",
		Short:   "CLI for managing Odoo 17 timesheets",
		Version: version.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = loadConfig()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			client, err = newClient(cfg)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			return err
		},

		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if client != nil {
				client.Close()
			}
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (skip discovery)")

	rootCmd.AddCommand(project.CMD(client, cfg))
	rootCmd.AddCommand(tasks.CMD(client))
	rootCmd.AddCommand(timesheet.CMD(client))
	rootCmd.AddCommand(fields.CMD(client))
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(configcmd.CMD(cfgFile))
	rootCmd.AddCommand(tui.CMD(client, cfg))
	rootCmd.AddCommand(entries.CMD(client))
	rootCmd.AddCommand(clock.CMD(client))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// loadConfig loads and merges config using file discovery and env vars.
func loadConfig() (*config.Config, error) {
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

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current Odoo user info",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		client, err := newClient(cfg)
		if err != nil {
			return err
		}
		defer client.Close()
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
