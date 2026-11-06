package output

import (
	"testing"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/spf13/cobra"
)

// newCmd builds a command with the output flags and parses args into it.
func newCmd(t *testing.T, args ...string) (*cobra.Command, error) {
	t.Helper()
	cmd := &cobra.Command{Use: "x", RunE: func(*cobra.Command, []string) error { return nil }}
	AddFlags(cmd.Flags())
	return cmd, cmd.ParseFlags(args)
}

func TestFormatOf(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		want     Format
		wantKind apperr.Kind
		parseErr bool
	}{
		{name: "default is table", want: FormatTable},
		{name: "json shorthand", args: []string{"--json"}, want: FormatJSON},
		{name: "format json", args: []string{"--format", "json"}, want: FormatJSON},
		{name: "format jsonl", args: []string{"--format=jsonl"}, want: FormatJSONL},
		{name: "format table explicit", args: []string{"--format", "table"}, want: FormatTable},
		{name: "json and format json agree", args: []string{"--json", "--format", "json"}, want: FormatJSON},
		{name: "json conflicts with format table", args: []string{"--json", "--format", "table"}, wantKind: apperr.KindValidation},
		{name: "unknown format rejected at parse", args: []string{"--format", "xml"}, parseErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := newCmd(t, tt.args...)
			if tt.parseErr {
				if err == nil {
					t.Fatal("expected parse error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got, err := FormatOf(cmd)
			if tt.wantKind != "" {
				if apperr.KindOf(err) != tt.wantKind {
					t.Fatalf("err = %v (kind %q), want kind %q", err, apperr.KindOf(err), tt.wantKind)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("FormatOf() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatOf_CommandWithoutFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "plain"}
	got, err := FormatOf(cmd)
	if err != nil || got != FormatTable {
		t.Errorf("FormatOf() = (%q, %v), want (table, nil)", got, err)
	}
}

func TestRequested(t *testing.T) {
	tests := []struct {
		name    string
		parse   []string // args parsed into the command
		rawArgs []string // os.Args[1:] seen by the error handler
		noFlags bool
		want    Format
	}{
		{name: "parsed json", parse: []string{"--json"}, want: FormatJSON},
		{name: "parsed jsonl", parse: []string{"--format", "jsonl"}, want: FormatJSONL},
		{name: "nothing requested", want: FormatTable},
		{name: "flag parse aborted before --json", rawArgs: []string{"entries", "--bogus", "--json"}, want: FormatJSON},
		{name: "raw format=jsonl", rawArgs: []string{"entries", "--bogus", "--format=jsonl"}, want: FormatJSONL},
		{name: "raw format jsonl separate", rawArgs: []string{"entries", "--bogus", "--format", "jsonl"}, want: FormatJSONL},
		{name: "raw json=1", rawArgs: []string{"entries", "--bogus", "--json=1"}, want: FormatJSON},
		{name: "raw json=false", rawArgs: []string{"entries", "--bogus", "--json=false"}, want: FormatTable},
		{name: "raw args after -- ignored", rawArgs: []string{"entries", "--", "--json"}, want: FormatTable},
		{name: "conflicting flags still json", parse: []string{"--json", "--format", "table"}, want: FormatJSON},
		{name: "command without output flags", noFlags: true, rawArgs: []string{"tui", "--json"}, want: FormatTable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd *cobra.Command
			if tt.noFlags {
				cmd = &cobra.Command{Use: "tui"}
			} else {
				var err error
				cmd, err = newCmd(t, tt.parse...)
				if err != nil {
					t.Fatalf("parse: %v", err)
				}
			}
			if got := Requested(cmd, tt.rawArgs); got != tt.want {
				t.Errorf("Requested() = %q, want %q", got, tt.want)
			}
		})
	}
}
