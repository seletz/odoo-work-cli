package config

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/spf13/cobra"
)

var configMerged bool

func CMD(cfgFile *string) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show discovered config files and merged configuration",
		Long: `Without flags, list the config files that were found, in merge order
(later files override earlier ones): the global file, then every
.odoo-work-cli.toml from the filesystem root down to the current
directory. With --config FILE only FILE is listed. Nothing connects to
Odoo.

With --merged, print the merged configuration as TOML instead: the files,
then [op_secrets] resolved through 1Password, then the environment
variables. The API key, the login password and the TOTP secret are left
out, so the output is a valid config file.`,
		Example: `  # Which files are read, from here
  odoo-work-cli config

  # The configuration as the other commands see it
  odoo-work-cli config --merged`,
		Annotations: map[string]string{
			manual.Group:          manual.GroupConfig,
			manual.Offline:        "true",
			manual.StaticExamples: "true",
			manual.Output: `"Discovered config files (merge order):" and one path per line, or "No
config files discovered."; with --merged a TOML document. There is no
JSON output.`,
			manual.Env:     "XDG_CONFIG_HOME, ODOO_URL, ODOO_DATABASE, ODOO_USERNAME, ODOO_PASSWORD, ODOO_WEB_PASSWORD, ODOO_TOTP_SECRET, ODOO_OP_TIMEOUT, ODOO_WORK_CLI_NO_OP",
			manual.Files:   manual.FileGlobalConfig + ", " + manual.FileLocalConfig,
			manual.Errors:  "config, secrets_locked",
			manual.SeeAlso: "reference/40-configuration.md, explanation/config-layering.md, explanation/secrets.md",
		},
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

	cmd.Flags().BoolVar(&configMerged, "merged", false, "print the merged configuration as TOML, without secrets")

	cmd.AddCommand(installcmd())
	return cmd
}
