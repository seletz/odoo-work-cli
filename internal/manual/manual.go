// Package manual holds the parts of the command reference (site pages, man
// pages) that live next to the commands: the cobra annotation keys a
// command sets, and the environment variables and files the pages
// describe. internal/docsgen renders the pages from the cobra tree; the
// command packages only import this package.
//
// A command documents itself with:
//
//   - Short: the NAME line ("odoo-work-cli-entries-add - <Short>").
//   - Long: DESCRIPTION. Paragraphs are separated by blank lines; a
//     paragraph whose lines all start with two spaces is preformatted.
//   - Example: EXAMPLES. "# text" lines describe the next command line;
//     "odoo-work-cli …" lines are run against the fake backend's demo
//     state by the generator and their output is shown below them.
//   - Annotations (keys below) for the other sections.
//
// The flag help text, its type, default (SetFlagDefault) and environment
// variable (SetFlagEnv) make up OPTIONS.
package manual

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo/fake"
)

// Command annotation keys.
const (
	// Group is the group of a top-level command in odoo-work-cli(1); one
	// of Groups. Subcommands inherit it.
	Group = "man:group"
	// Synopsis replaces the generated SYNOPSIS: one form per line, each a
	// full command line ("odoo-work-cli clock list [--date DAY]").
	Synopsis = "man:synopsis"
	// Output is the OUTPUT section: what the table output shows.
	Output = "man:output"
	// Schemas lists the JSON schemas (output.Schema*) of the --json
	// output, comma separated; OUTPUT lists their fields.
	Schemas = "man:schemas"
	// Errors lists the error kinds specific to the command (not_found,
	// conflict, access_denied), comma separated. The kinds every command
	// can fail with are added by KindsOf.
	Errors = "man:errors"
	// SeeAlso lists related commands ("odoo-work-cli clock list") and
	// pages of the site ("how-to/30-clock-in-out.md"), comma separated.
	SeeAlso = "man:see-also"
	// StaticExamples set to "true" shows the examples without running
	// them: for commands whose output depends on the wall clock, the
	// machine or a terminal.
	StaticExamples = "man:static-examples"
	// Offline set to "true" marks a command (and its subcommands) that
	// needs no Odoo connection: no config or secrets are loaded for it.
	Offline = "man:offline"
	// Env replaces the generated ENVIRONMENT list (comma separated
	// variable names; "" for none). Without it, an online command lists
	// every variable of Environment.
	Env = "man:env"
	// Files replaces the generated FILES list (comma separated keys of
	// FileHelp; "" for none). Without it, an online command lists
	// OnlineFiles.
	Files = "man:files"
)

// Flag annotation keys.
const (
	flagDefault  = "man:default"
	flagEnv      = "man:env"
	flagRequired = "man:required"
)

// Groups of odoo-work-cli(1), in order.
const (
	GroupTimesheets = "Timesheets"
	GroupAttendance = "Attendance"
	GroupProjects   = "Projects and tasks"
	GroupConfig     = "Configuration"
	GroupAgents     = "Agents and LLMs"
	GroupCompletion = "Shell completion"
)

// Groups returns the groups of odoo-work-cli(1) in display order.
//
// :returns: the group names
func Groups() []string {
	return []string{GroupTimesheets, GroupAttendance, GroupProjects, GroupConfig, GroupAgents, GroupCompletion}
}

// EnvHelp describes every environment variable the CLI reads.
var EnvHelp = map[string]string{
	"ODOO_URL":          "Odoo server URL; overrides url from the config files.",
	"ODOO_DATABASE":     "Odoo database name; overrides database.",
	"ODOO_USERNAME":     "Odoo login, usually an e-mail address; overrides username.",
	"ODOO_PASSWORD":     "Odoo API key, used for XML-RPC. Despite the name, not the login password.",
	"ODOO_WEB_PASSWORD": "Odoo login password, used for the web session of clock in and clock out.",
	"ODOO_TOTP_SECRET":  "TOTP secret (base32 or otpauth:// URL) for accounts with two-factor authentication.",
	"ODOO_OP_TIMEOUT":   "Timeout of the 1Password call that resolves [op_secrets], a Go duration; default 5s.",
	config.NoOPEnvVar:   "If 1, true, yes or on, [op_secrets] is ignored and the 1Password CLI is never called.",
	fake.BackendEnvVar:  "Backend, odoo (default) or fake; --backend overrides it.",
	fake.StateEnvVar:    "State file of the fake backend; default odoo-work-cli-fake-state.json in the temporary directory.",
	"XDG_CONFIG_HOME":   "Base directory of the global config file; default ~/.config.",
}

// OdooEnv returns the variables of an online command, in display order.
//
// :returns: the variable names
func OdooEnv() []string {
	return []string{"ODOO_URL", "ODOO_DATABASE", "ODOO_USERNAME", "ODOO_PASSWORD", "ODOO_WEB_PASSWORD",
		"ODOO_TOTP_SECRET", "ODOO_OP_TIMEOUT", config.NoOPEnvVar, fake.BackendEnvVar, fake.StateEnvVar, "XDG_CONFIG_HOME"}
}

// Files the reference describes.
const (
	FileGlobalConfig = "~/.config/odoo-work-cli/config.toml"
	FileLocalConfig  = ".odoo-work-cli.toml"
	FileFakeState    = "$TMPDIR/" + fake.DefaultStateFile
)

// FileHelp describes the files in FILES.
var FileHelp = map[string]string{
	FileGlobalConfig: "Global configuration ($XDG_CONFIG_HOME/odoo-work-cli/config.toml if XDG_CONFIG_HOME is set). " +
		"Created by odoo-work-cli config install.",
	FileLocalConfig: "Configuration of a directory tree. Every such file from the filesystem root down to the current " +
		"directory is merged over the global file, the deepest last. --config FILE loads only FILE instead.",
	FileFakeState: "State of the fake backend (--backend fake); " + fake.StateEnvVar + " sets another path. " +
		"Delete it to start over with the seed data.",
}

// OnlineFiles returns the files of an online command, in display order.
//
// :returns: the file names
func OnlineFiles() []string {
	return []string{FileGlobalConfig, FileLocalConfig, FileFakeState}
}

// List returns the comma separated values of annotation key on cmd, with
// blanks trimmed and empty items dropped; nil if the key is unset.
//
// :param cmd: the command
// :param key: the annotation key
// :returns: the values
func List(cmd *cobra.Command, key string) []string {
	var out []string
	for _, v := range strings.Split(cmd.Annotations[key], ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// IsOffline reports whether cmd or one of its parents is marked Offline.
//
// :param cmd: the command
// :returns: true if the command needs no Odoo connection
func IsOffline(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations[Offline] == "true" {
			return true
		}
	}
	return false
}

// GroupOf returns the group of cmd: its own Group annotation or that of
// the nearest parent which has one.
//
// :param cmd: the command
// :returns: the group, "" if none is set
func GroupOf(cmd *cobra.Command) string {
	for c := cmd; c != nil; c = c.Parent() {
		if g := c.Annotations[Group]; g != "" {
			return g
		}
	}
	return ""
}

// KindsOf returns the error kinds cmd can fail with, in apperr.Kinds
// order: validation and internal always, the connection kinds (auth,
// network, config, secrets_locked) for online commands, plus the kinds of
// its Errors annotation.
//
// :param cmd: the command
// :returns: the kinds
func KindsOf(cmd *cobra.Command) []apperr.Kind {
	want := map[apperr.Kind]bool{apperr.KindValidation: true, apperr.KindInternal: true}
	if !IsOffline(cmd) {
		for _, k := range []apperr.Kind{apperr.KindAuth, apperr.KindNetwork, apperr.KindConfig, apperr.KindSecretsLocked} {
			want[k] = true
		}
	}
	for _, k := range List(cmd, Errors) {
		want[apperr.Kind(k)] = true
	}
	var out []apperr.Kind
	for _, k := range apperr.Kinds() {
		if want[k] {
			out = append(out, k)
		}
	}
	return out
}

// SetFlagDefault records the default of flag name as shown in OPTIONS,
// for flags whose zero value stands for something ("today").
//
// :param fs: the flag set
// :param name: the flag name
// :param text: the default, as prose
func SetFlagDefault(fs *pflag.FlagSet, name, text string) {
	_ = fs.SetAnnotation(name, flagDefault, []string{text})
}

// SetFlagEnv records the environment variable that sets flag name.
//
// :param fs: the flag set
// :param name: the flag name
// :param variable: the environment variable
func SetFlagEnv(fs *pflag.FlagSet, name, variable string) {
	_ = fs.SetAnnotation(name, flagEnv, []string{variable})
}

// SetFlagRequired marks flag name as required in SYNOPSIS and OPTIONS,
// for flags the command checks itself rather than with cobra's
// MarkFlagRequired.
//
// :param fs: the flag set
// :param name: the flag name
func SetFlagRequired(fs *pflag.FlagSet, name string) {
	_ = fs.SetAnnotation(name, flagRequired, []string{"true"})
}

// FlagRequired reports whether flag f is required, by cobra's
// MarkFlagRequired or by SetFlagRequired.
//
// :param f: the flag
// :returns: true if the flag is required
func FlagRequired(f *pflag.Flag) bool {
	return first(f.Annotations[flagRequired]) == "true" || first(f.Annotations[cobra.BashCompOneRequiredFlag]) == "true"
}

// HelpSuffix returns what --help appends to the help text of flag f:
// " (required)" and " (default X)" from SetFlagRequired, MarkFlagRequired
// and SetFlagDefault. pflag prints other defaults itself.
//
// :param f: the flag
// :returns: the suffix, "" if there is nothing to add
func HelpSuffix(f *pflag.Flag) string {
	var s string
	if FlagRequired(f) {
		s += " (required)"
	}
	if d := FlagDefault(f); d != "" {
		s += " (default " + d + ")"
	}
	return s
}

// DecorateHelp appends HelpSuffix to the help text of every flag of root
// and its subcommands, once, so --help shows what the reference shows.
//
// :param root: the root command
func DecorateHelp(root *cobra.Command) {
	seen := map[*pflag.Flag]bool{}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, fs := range []*pflag.FlagSet{c.LocalNonPersistentFlags(), c.PersistentFlags()} {
			fs.VisitAll(func(f *pflag.Flag) {
				if !seen[f] {
					seen[f] = true
					f.Usage += HelpSuffix(f)
				}
			})
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}

// ReferenceURL returns the page of cmd on the documentation site.
//
// :param cmd: the command
// :returns: the URL
func ReferenceURL(cmd *cobra.Command) string {
	if !cmd.HasParent() {
		return SiteURL + "reference/commands/"
	}
	return SiteURL + "reference/commands/" + ManName(cmd) + "/"
}

// ManName returns the man page name of cmd: its command path joined with
// hyphens (odoo-work-cli-entries-add).
//
// :param cmd: the command
// :returns: the name, without section
func ManName(cmd *cobra.Command) string {
	return strings.ReplaceAll(cmd.CommandPath(), " ", "-")
}

// FlagDefault returns the default recorded by SetFlagDefault, "" if none.
//
// :param f: the flag
// :returns: the default
func FlagDefault(f *pflag.Flag) string {
	return first(f.Annotations[flagDefault])
}

// FlagEnv returns the variable recorded by SetFlagEnv, "" if none.
//
// :param f: the flag
// :returns: the variable name
func FlagEnv(f *pflag.Flag) string {
	return first(f.Annotations[flagEnv])
}

func first(v []string) string {
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// KindHelp says in one line when each error kind occurs, for EXIT STATUS.
var KindHelp = map[apperr.Kind]string{
	apperr.KindAuth:          "Authentication failed: wrong API key or login password, TOTP rejected.",
	apperr.KindAccessDenied:  "Odoo's access rules forbid the operation, e.g. attendance writes without the Attendance Officer right.",
	apperr.KindNotFound:      "The record does not exist or is not visible.",
	apperr.KindValidation:    "Invalid input: unknown flag, bad argument, malformed date, time or hours, a time in the future, an unknown project.",
	apperr.KindConflict:      "The request contradicts the current state, e.g. clock in while clocked in, or an overlapping attendance record.",
	apperr.KindNetwork:       "Odoo cannot be reached: DNS, connection refused, timeout, TLS.",
	apperr.KindConfig:        "The local configuration is missing or invalid.",
	apperr.KindSecretsLocked: "1Password is locked or unreachable, so op:// references could not be resolved.",
	apperr.KindInternal:      "Anything else: an unexpected Odoo response or a bug.",
}

// SitePages maps the pages of the documentation site a SeeAlso annotation
// may name (path below docs/) to their titles. A docsgen test checks them
// against the files.
var SitePages = map[string]string{
	"tutorials/10-try-it-offline.md":       "Try it offline",
	"tutorials/20-first-booking.md":        "Your first booking",
	"tutorials/30-tui-tour.md":             "A tour of the TUI",
	"how-to/10-install-and-update.md":      "Install and update",
	"how-to/20-book-time.md":               "Book time",
	"how-to/30-clock-in-out.md":            "Clock in and out",
	"how-to/40-find-projects-and-tasks.md": "Find projects and tasks",
	"how-to/50-dry-run.md":                 "Preview a write with --dry-run",
	"how-to/60-json-output.md":             "Script with JSON output",
	"how-to/70-test-without-odoo.md":       "Test without Odoo",
	"how-to/80-agents.md":                  "Use it from a coding agent",
	"how-to/90-customise-tui.md":           "Customise the TUI",
	"how-to/95-troubleshoot.md":            "Troubleshoot",
	"explanation/attendance-model.md":      "The attendance model",
	"explanation/config-layering.md":       "Config layering",
	"explanation/secrets.md":               "Secrets",
	"explanation/transports.md":            "Two transports",
	"reference/20-json-output.md":          "JSON output",
	"reference/30-errors.md":               "Errors and exit codes",
	"reference/40-configuration.md":        "Configuration",
	"reference/50-tui-keys.md":             "TUI keys and views",
}

// SiteURL is the base URL of the documentation site.
const SiteURL = "https://seletz.github.io/odoo-work-cli/"
