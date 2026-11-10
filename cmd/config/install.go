package config

import (
	"fmt"
	"strings"

	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/spf13/cobra"
)

func installcmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Create a default config file in the platform config directory",
		Long: `Write a commented default configuration to the global config file,
$XDG_CONFIG_HOME/odoo-work-cli/config.toml (default
~/.config/odoo-work-cli/config.toml), after asking for confirmation on
standard input. Any answer but y or yes aborts, and so does an existing
file: it is never overwritten.

Then set url, database and username in the file, and the secrets in
[op_secrets] or in environment variables.`,
		Example: `  odoo-work-cli config install`,
		Annotations: map[string]string{
			manual.StaticExamples: "true",
			manual.Output:         "The path of the file, the prompt \"Continue? [y/N]\" and the result.",
			manual.Env:            "XDG_CONFIG_HOME",
			manual.Files:          manual.FileGlobalConfig,
			manual.SeeAlso:        "how-to/10-install-and-update.md, reference/40-configuration.md",
		},
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
