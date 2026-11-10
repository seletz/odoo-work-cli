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
		return 0
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
	return 1
}

// newRootCmd builds the command tree. started is set once cobra has parsed
// and validated the command line and command code begins to run.
func newRootCmd(setup setupFunc) (*cobra.Command, *bool) {
	var cfgFile, backend string
	started := new(bool)
	deps := &app.Deps{}

	rootCmd := &cobra.Command{
		Use:     "odoo-work-cli",
		Short:   "CLI for managing Odoo 17 timesheets",
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (skip discovery)")
	rootCmd.PersistentFlags().StringVar(&backend, "backend", "",
		"backend: odoo (default) or fake (offline, made-up data; env ODOO_WORK_CLI_BACKEND)")
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

	return rootCmd, started
}

func isNoneSetupCommand(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		switch current.Name() {
		case "completion", "__complete", "__completeNoDesc":
			return true
		case "config", "llm":
			return true
		}
	}
	return false
}

// defaultSetup loads the config and connects to Odoo. Config errors that
// carry no kind are KindConfig; connection errors are classified by
// apperr (network) and odoo.ClassifyError (auth).
func defaultSetup(cfgFile string) (*config.Config, odoo.Client, error) {
	cfg, err := loadConfig(cfgFile)
	if err != nil {
		var typed *apperr.Error
		if !errors.As(err, &typed) {
			err = apperr.Wrap(apperr.KindConfig, err)
		}
		return nil, nil, err
	}
	client, err := newClient(cfg)
	if err != nil {
		return nil, nil, err
	}
	return cfg, client, nil
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
