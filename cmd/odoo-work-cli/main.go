package main

import (
	"fmt"
	"os"

	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/spf13/cobra"
)

var cfgFile string

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "odoo-work-cli",
	Short: "CLI for managing Odoo 17 timesheets",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.config/odoo-work-cli/config.toml)")

	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(tasksCmd)
	rootCmd.AddCommand(timesheetsCmd)
	rootCmd.AddCommand(fieldsCmd)
	rootCmd.AddCommand(whoamiCmd)
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List Odoo projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadFromEnv()
		if err != nil {
			return err
		}
		client, err := odoo.NewXMLRPCClient(cfg.URL, cfg.Database, cfg.Username, cfg.Password)
		if err != nil {
			return err
		}
		defer client.Close()
		projects, err := client.ListProjects()
		if err != nil {
			return err
		}
		fmt.Printf("%-8s %-40s %s\n", "ID", "Name", "Active")
		fmt.Printf("%-8s %-40s %s\n", "--------", "----------------------------------------", "------")
		for _, p := range projects {
			fmt.Printf("%-8d %-40s %v\n", p.ID, p.Name, p.Active)
		}
		return nil
	},
}

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "List Odoo tasks",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("not implemented yet")
	},
}

var timesheetsCmd = &cobra.Command{
	Use:   "timesheets",
	Short: "Manage Odoo timesheets",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("not implemented yet")
	},
}

var fieldsCmd = &cobra.Command{
	Use:   "fields",
	Short: "Inspect Odoo model fields",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("not implemented yet")
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current Odoo user info",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadFromEnv()
		if err != nil {
			return err
		}
		client, err := odoo.NewXMLRPCClient(cfg.URL, cfg.Database, cfg.Username, cfg.Password)
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
