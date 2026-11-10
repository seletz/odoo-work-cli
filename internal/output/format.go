// Package output implements the CLI's machine-readable output contract:
// the --format/--json flags, stable JSON shapes for every command (see
// types.go and the embedded JSON Schemas), and the JSON error envelope.
package output

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Format is an output format selected with --format or --json.
type Format string

// Output formats.
const (
	// FormatTable is the human-readable default.
	FormatTable Format = "table"
	// FormatJSON prints one indented JSON document.
	FormatJSON Format = "json"
	// FormatJSONL prints one compact JSON object per line.
	FormatJSONL Format = "jsonl"
)

const (
	flagFormat = "format"
	flagJSON   = "json"
)

// IsMachine reports whether f is a machine-readable format.
func (f Format) IsMachine() bool { return f == FormatJSON || f == FormatJSONL }

// String implements pflag.Value.
func (f *Format) String() string { return string(*f) }

// Type implements pflag.Value.
func (f *Format) Type() string { return "table|json|jsonl" }

// Set implements pflag.Value and rejects unknown formats.
func (f *Format) Set(s string) error {
	switch Format(s) {
	case FormatTable, FormatJSON, FormatJSONL:
		*f = Format(s)
		return nil
	}
	return fmt.Errorf("invalid format %q: expected table, json or jsonl", s)
}

// AddFlags registers --format and --json on fs. Use cmd.Flags() for a
// single command or cmd.PersistentFlags() for a command group.
//
// :param fs: flag set to register the flags on
func AddFlags(fs *pflag.FlagSet) {
	f := FormatTable
	fs.Var(&f, flagFormat, "output `FORMAT`: table, json or jsonl")
	fs.Bool(flagJSON, false, "shorthand for --format json")
}

// FormatOf returns the format selected on cmd. Commands without the output
// flags always use FormatTable.
//
// :param cmd: the executing command (flags already parsed)
// :returns: the selected format, or a KindValidation error if --json and
//
//	--format disagree
func FormatOf(cmd *cobra.Command) (Format, error) {
	formatFlag := cmd.Flags().Lookup(flagFormat)
	jsonFlag := cmd.Flags().Lookup(flagJSON)
	if formatFlag == nil || jsonFlag == nil {
		return FormatTable, nil
	}
	format := Format(formatFlag.Value.String())
	if jsonFlag.Value.String() == "true" {
		if formatFlag.Changed && format != FormatJSON {
			return FormatTable, apperr.New(apperr.KindValidation,
				"--json conflicts with --format %s", format)
		}
		return FormatJSON, nil
	}
	return format, nil
}

// Requested returns the machine format the user asked for, for rendering
// an error. Unlike FormatOf it never fails: a conflict between --json and
// --format resolves to JSON, and when flag parsing was aborted it scans
// rawArgs (os.Args[1:]) for --json or --format json|jsonl.
//
// :param cmd: the command cobra resolved (may have unparsed flags)
// :param rawArgs: the raw command-line arguments
// :returns: the requested format
func Requested(cmd *cobra.Command, rawArgs []string) Format {
	if cmd == nil || cmd.Flags().Lookup(flagFormat) == nil {
		return FormatTable
	}
	if f, err := FormatOf(cmd); err != nil {
		return FormatJSON
	} else if f.IsMachine() {
		return f
	}
	return scanArgs(rawArgs)
}

// RequestedInArgs reports the machine format requested in raw arguments
// alone, before any command has been resolved.
//
// :param rawArgs: the raw command-line arguments
// :returns: the requested format, FormatTable if none
func RequestedInArgs(rawArgs []string) Format { return scanArgs(rawArgs) }

// scanArgs finds --json / --format <machine format> in raw arguments.
func scanArgs(args []string) Format {
	for i, a := range args {
		switch {
		case a == "--":
			return FormatTable
		case a == "--"+flagJSON:
			return FormatJSON
		case strings.HasPrefix(a, "--"+flagJSON+"="):
			// Same spellings as strconv.ParseBool, which pflag uses.
			if on, err := strconv.ParseBool(strings.TrimPrefix(a, "--"+flagJSON+"=")); err == nil && on {
				return FormatJSON
			}
		case strings.HasPrefix(a, "--"+flagFormat+"="):
			if f := Format(strings.TrimPrefix(a, "--"+flagFormat+"=")); f.IsMachine() {
				return f
			}
		case a == "--"+flagFormat && i+1 < len(args):
			if f := Format(args[i+1]); f.IsMachine() {
				return f
			}
		}
	}
	return FormatTable
}
