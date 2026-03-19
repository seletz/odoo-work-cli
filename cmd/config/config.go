package config

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/spf13/cobra"
)

var configMerged bool

func CMD(cfgFile *string) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show discovered config files and merged configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := config.Discover(*cfgFile)
			if err != nil {
				return err
			}

			if !configMerged {
				if len(result.Files) == 0 {
					fmt.Println("No config files discovered.")
				} else {
					fmt.Println("Discovered config files (merge order):")
					for _, f := range result.Files {
						fmt.Printf("  %s\n", f)
					}
				}
				return nil
			}

			// Password is tagged toml:"-" on Config, so the encoder
			// omits it automatically — output is a valid config file.
			var buf bytes.Buffer
			if err := toml.NewEncoder(&buf).Encode(result.Config); err != nil {
				return fmt.Errorf("encoding config: %w", err)
			}
			fmt.Print(buf.String())
			return nil
		},
	}

	cmd.Flags().BoolVar(&configMerged, "merged", false, "print merged TOML config (password redacted)")

	cmd.AddCommand(installcmd())
	return cmd
}
