package config

import (
	"fmt"
	"strings"

	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/spf13/cobra"
)

func installcmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Create a default config file in the platform config directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.DefaultConfigPath()
			if err != nil {
				return err
			}

			fmt.Printf("This will create a default config file at:\n  %s\n\n", path)

			if !confirmPrompt("Continue?") {
				fmt.Println("Aborted.")
				return nil
			}

			if err := config.InstallConfig(path); err != nil {
				return err
			}

			fmt.Printf("Config file created at: %s\n", path)
			fmt.Println("Edit it to set your Odoo URL, database, and username.")
			fmt.Println("Set ODOO_PASSWORD via environment variable (see .env.1p).")
			return nil
		},
	}

	return cmd
}

// confirmPrompt prints msg and reads y/N from stdin.
func confirmPrompt(msg string) bool {
	fmt.Printf("%s [y/N] ", msg)
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
