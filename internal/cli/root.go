// Package cli builds the odoo-work-cli command tree and runs it. It lives
// outside package main so tools (the docs generator) can reach the tree.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/seletz/odoo-work-cli/cmd/clock"
	configcmd "github.com/seletz/odoo-work-cli/cmd/config"
	"github.com/seletz/odoo-work-cli/cmd/entries"
	"github.com/seletz/odoo-work-cli/cmd/fields"
	"github.com/seletz/odoo-work-cli/cmd/llm"
	"github.com/seletz/odoo-work-cli/cmd/project"
	"github.com/seletz/odoo-work-cli/cmd/tasks"
	"github.com/seletz/odoo-work-cli/cmd/timesheet"
	"github.com/seletz/odoo-work-cli/cmd/tui"
	"github.com/seletz/odoo-work-cli/internal/app"
	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"

	"github.com/seletz/odoo-work-cli/internal/version"
	"github.com/spf13/cobra"
)

// setupFunc loads the configuration and connects to Odoo. Tests replace it
// with a fake.
type setupFunc func(cfgFile string) (*config.Config, odoo.Client, error)

var registerClassifiers sync.Once

// Main runs the CLI with the process arguments and real setup and returns
// the exit code.
//
// :returns: the exit code
func Main() int {
	return Run(os.Args[1:], os.Stdout, os.Stderr)
}

// Run executes the CLI with the real config and Odoo setup.
//
// :param args: command-line arguments without the program name
// :param stdout: standard output
// :param stderr: standard error
// :returns: the exit code
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, defaultSetup)
}

// NewRootCmd returns the full command tree with the real setup, for
// generating documentation.
//
// :returns: the root command
func NewRootCmd() *cobra.Command {
	cmd, _ := newRootCmd(defaultSetup)
	return cmd
}

// run executes the CLI and returns the process exit code. It is the single
// place where errors are rendered as an output.ErrorEnvelope on stdout
// when --json/--format json|jsonl was requested; otherwise cobra prints
// "Error: …" on stderr. Every failure exits with 1.
//
// :param args: command-line arguments without the program name
// :param stdout: standard output
// :param stderr: standard error
// :param setup: config + client factory
// :returns: the exit code
func run(args []string, stdout, stderr io.Writer, setup setupFunc) int {
	code, _ := runErr(args, stdout, stderr, setup)
	return code
}

// runErr is run that also returns the error it rendered (nil on success),
// for callers that branch on its kind, like the MCP server.
func runErr(args []string, stdout, stderr io.Writer, setup setupFunc) (int, error) {
	registerClassifiers.Do(func() { apperr.RegisterClassifier(odoo.ClassifyError) })

	rootCmd, started := newRootCmd(setup)
	rootCmd.SetArgs(args)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	// cobra never prints errors itself: run() renders every error in one
	// place, after FriendlyError, as "Error: …" on stderr (table mode) or
	// as JSON on stdout (machine mode).
	machine := output.RequestedInArgs(args).IsMachine()
	rootCmd.SilenceErrors = true
	// cobra would print usage to the Out writer (stdout here); usage is
	// printed to stderr below instead.
	rootCmd.SilenceUsage = true

	cmd, err := rootCmd.ExecuteC()
	if err == nil {
		return 0, nil
	}
	if !*started {
		// cobra failed before any command code ran: unknown flag or
		// command, bad argument count, missing required flag.
		var typed *apperr.Error
		if !errors.As(err, &typed) {
			err = apperr.Wrap(apperr.KindValidation, err)
		}
	}
	err = odoo.FriendlyError(err)
	if !machine {
		_, _ = fmt.Fprintln(stderr, "Error: "+err.Error())
	}
	if !*started {
		// As cobra does: usage after errors from flag/arg validation, but
		// not for an unknown command (flags never parsed), whose message
		// already points to --help.
		if !machine && cmd.Flags().Parsed() {
			_, _ = fmt.Fprintln(stderr, cmd.UsageString())
		}
	}
	format := output.Requested(cmd, args)
	if !format.IsMachine() && machine {
		// The resolved command has no output flags ("fields --json",
		// "bogus --json"), but the caller asked for JSON: honour it.
		format = output.RequestedInArgs(args)
	}
	if format.IsMachine() {
		output.WriteError(stdout, format, err)
	}
	return 1, err
}

// newRootCmd builds the command tree. started is set once cobra has parsed
// and validated the command line and command code begins to run.
func newRootCmd(setup setupFunc) (*cobra.Command, *bool) {
	var cfgFile, backend string
	started := new(bool)
	deps := &app.Deps{}

	rootCmd := &cobra.Command{
		Use:   "odoo-work-cli",
		Short: "Book Odoo 17 timesheets and attendance from the terminal",
		Long: `odoo-work-cli reads and writes your own Odoo 17 timesheet entries and
attendance records: list projects and tasks, book, change and delete
timesheet entries, clock in and out, and correct attendance afterwards. The
tui command opens an interactive weekly grid.

Every command that prints data prints a table for humans by default and a
stable JSON contract with --json or --format json|jsonl, for scripts and
coding agents. Write commands validate their input before they connect;
entries add and entries update can show the result first with --dry-run.

Models are read and written over XML-RPC with an Odoo API key. Clock in and
clock out without --at use the web session of the Odoo attendance systray,
which needs the login password and, with two-factor authentication, the
TOTP secret.
Secrets can be resolved from 1Password at startup ([op_secrets]).

--backend fake (or ODOO_WORK_CLI_BACKEND=fake) replaces Odoo by a local file
with made-up data; nothing leaves the machine.`,
		Example: `  # Who am I, on the offline fake backend
  odoo-work-cli --backend fake whoami

  # The bookings of one day
  odoo-work-cli --backend fake entries --date 2026-04-08`,
		Annotations: map[string]string{
			manual.SeeAlso: "tutorials/10-try-it-offline.md, how-to/10-install-and-update.md, " +
				"reference/20-json-output.md, reference/30-errors.md, reference/40-configuration.md",
		},
		Version: version.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// cobra checks required flags only after this hook; check them
			// here so they are reported as usage errors and before any
			// connection is made.
			if err := cmd.ValidateRequiredFlags(); err != nil {
				return apperr.Wrap(apperr.KindValidation, err)
			}
			if err := cmd.ValidateFlagGroups(); err != nil {
				return apperr.Wrap(apperr.KindValidation, err)
			}
			if _, err := output.FormatOf(cmd); err != nil {
				return err
			}
			if err := checkBackendFlag(backend); err != nil {
				return err
			}
			*started = true
			// From here on errors are runtime failures, not usage errors.
			cmd.SilenceUsage = true
			if !isNoneSetupCommand(cmd) {
				// Config and secrets load lazily on first RequireConfig /
				// RequireClient, after the command validated its input.
				deps.Loader = loaderFor(backend, cfgFile, setup)
			}
			return nil
		},

		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			deps.Close()
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "read only the config `FILE` (skip discovery)")
	rootCmd.PersistentFlags().StringVar(&backend, "backend", "",
		"`BACKEND`: odoo or fake (offline, made-up data; env ODOO_WORK_CLI_BACKEND)")
	manual.SetFlagDefault(rootCmd.PersistentFlags(), "backend", "odoo")
	manual.SetFlagEnv(rootCmd.PersistentFlags(), "backend", "ODOO_WORK_CLI_BACKEND")
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return apperr.Wrap(apperr.KindValidation, err)
	})

	rootCmd.AddCommand(project.CMD(deps))
	rootCmd.AddCommand(tasks.CMD(deps))
	rootCmd.AddCommand(timesheet.CMD(deps))
	rootCmd.AddCommand(fields.CMD(deps))
	rootCmd.AddCommand(whoamiCMD(deps))
	rootCmd.AddCommand(configcmd.CMD(&cfgFile))
	rootCmd.AddCommand(tui.CMD(deps))
	rootCmd.AddCommand(entries.CMD(deps))
	rootCmd.AddCommand(clock.CMD(deps))
	rootCmd.AddCommand(llm.CMD())
	rootCmd.AddCommand(mcpCMD(mcpStages, &cfgFile, &backend))
	documentCompletion(rootCmd)
	manual.DecorateHelp(rootCmd)
	rootCmd.SetHelpTemplate(rootCmd.HelpTemplate() + helpFooter)

	return rootCmd, started
}

// isNoneSetupCommand reports whether cmd runs without config, secrets
// and an Odoo connection: cobra's completion machinery and every command
// marked manual.Offline.
func isNoneSetupCommand(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		switch current.Name() {
		case "completion", "__complete", "__completeNoDesc":
			return true
		case "mcp":
			// The server loads config per tool call (newMCPServer); it is
			// not offline, its tools need the connection.
			return true
		}
	}
	return manual.IsOffline(cmd)
}

func init() {
	cobra.AddTemplateFunc("manName", manual.ManName)
	cobra.AddTemplateFunc("referenceURL", manual.ReferenceURL)
}

// helpFooter points from --help to the full reference.
const helpFooter = `
Full reference: man {{manName .}}
or {{referenceURL .}}
`

// documentCompletion adds cobra's completion command now instead of at
// Execute, so it carries the annotations and examples of the reference.
func documentCompletion(root *cobra.Command) {
	root.InitDefaultCompletionCmd()
	c, _, err := root.Find([]string{"completion"})
	if err != nil || c.Name() != "completion" {
		return
	}
	c.Long = `Print the shell completion script of odoo-work-cli for bash, zsh, fish
or PowerShell on standard output. It completes commands and flags.

The Homebrew cask installs the scripts for bash, zsh and fish; the
release archives contain them in completions/. Load a script by hand as
shown for each shell below.`
	c.Example = `  # Load completion into the current zsh session
  source <(odoo-work-cli completion zsh)

  # Install it for every new bash session (Linux)
  odoo-work-cli completion bash > /etc/bash_completion.d/odoo-work-cli`
	c.Annotations = map[string]string{
		manual.Group:          manual.GroupCompletion,
		manual.Offline:        "true",
		manual.StaticExamples: "true",
		manual.SeeAlso:        "how-to/10-install-and-update.md",
	}
}

// defaultSetup loads the config and connects to Odoo. Config errors that
// carry no kind are KindConfig; connection errors are classified by
// apperr (network) and odoo.ClassifyError (auth).
func defaultSetup(cfgFile string) (*config.Config, odoo.Client, error) {
	cfg, err := loadKindedConfig(cfgFile)
	if err != nil {
		return nil, nil, err
	}
	client, err := newClient(cfg)
	if err != nil {
		return nil, nil, err
	}
	return cfg, client, nil
}

// loadKindedConfig is loadConfig with config errors that carry no kind
// classified as KindConfig. It is the first stage of defaultSetup: the
// stage that may call 1Password.
func loadKindedConfig(cfgFile string) (*config.Config, error) {
	cfg, err := loadConfig(cfgFile)
	if err != nil {
		var typed *apperr.Error
		if !errors.As(err, &typed) {
			err = apperr.Wrap(apperr.KindConfig, err)
		}
		return nil, err
	}
	return cfg, nil
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
func newClient(cfg *config.Config) (odoo.Client, error) {
	return odoo.NewXMLRPCClient(cfg.URL, cfg.Database, cfg.Username, cfg.Password, cfg.WebPassword, cfg.TOTPSecret, cfg.Models)
}

func whoamiCMD(deps *app.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show current Odoo user info",
		Long: `Print the Odoo user the configured credentials belong to: ID, name,
login, e-mail and company. Use it to check that the configuration and the
API key work.`,
		Example: `  odoo-work-cli whoami

  # The same as JSON
  odoo-work-cli whoami --json`,
		Annotations: map[string]string{
			manual.Group:   manual.GroupConfig,
			manual.Output:  "One line each for ID, Name, Login, Email and Company.\n\nWith --json one user object.",
			manual.Schemas: "user",
			manual.SeeAlso: "odoo-work-cli config, how-to/95-troubleshoot.md",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.FormatOf(cmd)
			if err != nil {
				return err
			}
			client, err := deps.RequireClient()
			if err != nil {
				return err
			}

			info, err := client.WhoAmI()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if format.IsMachine() {
				return output.WriteObject(w, format, output.NewUser(*info))
			}
			_, _ = fmt.Fprintf(w, "ID:      %d\n", info.ID)
			_, _ = fmt.Fprintf(w, "Name:    %s\n", info.Name)
			_, _ = fmt.Fprintf(w, "Login:   %s\n", info.Login)
			_, _ = fmt.Fprintf(w, "Email:   %s\n", info.Email)
			_, _ = fmt.Fprintf(w, "Company: %s\n", info.Company)
			return nil
		},
	}
	output.AddFlags(cmd.Flags())
	return cmd
}
