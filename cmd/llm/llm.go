// Package llm implements `odoo-work-cli llm --skill`, which prints a guide
// that lets coding agents use the CLI correctly without trial and error.
package llm

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/manual"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/spf13/cobra"
)

//go:embed guide.md
var guideTemplate string

// kindHelp tells an agent what to do about each error kind. Every kind in
// apperr.Kinds must have an entry (enforced by a test); the list itself is
// generated from apperr.Kinds, never hard-coded in guide.md.
var kindHelp = map[apperr.Kind]string{
	apperr.KindAuth:          "credentials rejected (API key, web password, TOTP). Do not retry; ask the user to fix the credentials.",
	apperr.KindAccessDenied:  "Odoo access rules forbid it. For attendance writes see the Attendance Officer pitfall. Do not retry.",
	apperr.KindNotFound:      "the record ID does not exist. Look the ID up again (`entries`, `clock list`, `projects`, `tasks`).",
	apperr.KindValidation:    "bad flags, arguments or values (also: time in the future, unknown project ID). Fix the input; the message says what.",
	apperr.KindConflict:      "contradicts the current state (already clocked in, not clocked in, `clock edit --create-only` on a day that has a record). Re-read the state with `clock list`/`clock status` and decide; do not blindly retry.",
	apperr.KindNetwork:       "Odoo is not reachable. Retrying once is fine; if it persists, ask the user.",
	apperr.KindConfig:        "local configuration is missing or invalid. Ask the user; see `odoo-work-cli config`.",
	apperr.KindSecretsLocked: "1Password is locked or unreachable (fails after about 5 s, see `ODOO_OP_TIMEOUT`). Stop and ask the user to unlock it. Do not retry in a loop.",
	apperr.KindInternal:      "unexpected failure or bug. Report the message to the user.",
}

// Guide renders the agent guide: the embedded Markdown with the error kinds
// and the JSON output fields generated from their single sources of truth.
func Guide() (string, error) {
	schemas, err := schemaSummary()
	if err != nil {
		return "", err
	}
	r := strings.NewReplacer(
		"{{ERROR_KINDS}}", kindList(),
		"{{SCHEMAS}}", schemas,
	)
	return r.Replace(guideTemplate), nil
}

func kindList() string {
	var b strings.Builder
	for _, k := range apperr.Kinds() {
		fmt.Fprintf(&b, "- `%s`: %s\n", k, kindHelp[k])
	}
	return strings.TrimRight(b.String(), "\n")
}

// schemaOrder fixes the order of the generated field list.
var schemaOrder = []struct{ name, used string }{
	{output.SchemaProject, "`projects`"},
	{output.SchemaTask, "`tasks`"},
	{output.SchemaEntry, "`entries`, `timesheets`, `entries add/update/delete`"},
	{output.SchemaUser, "`whoami`"},
	{output.SchemaAttendance, "`clock in/out`, `clock list` (array)"},
	{output.SchemaClockStatus, "`clock status`"},
	{output.SchemaAttendanceEdit, "`clock edit`"},
	{output.SchemaEntryDryRun, "`entries add/update --dry-run`"},
}

// schemaSummary lists the top-level fields of each embedded JSON Schema.
// Fields marked `?` are not in "required".
func schemaSummary() (string, error) {
	var b strings.Builder
	for _, s := range schemaOrder {
		raw, err := output.Schemas.ReadFile("schemas/" + s.name + ".schema.json")
		if err != nil {
			return "", err
		}
		var doc struct {
			Required   []string                   `json:"required"`
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			return "", fmt.Errorf("schema %s: %w", s.name, err)
		}
		req := map[string]bool{}
		for _, r := range doc.Required {
			req[r] = true
		}
		// Required fields first, in schema order; optional ones after.
		names := make([]string, 0, len(doc.Properties))
		names = append(names, doc.Required...)
		var opt []string
		for p := range doc.Properties {
			if !req[p] {
				opt = append(opt, p+"?")
			}
		}
		sort.Strings(opt)
		names = append(names, opt...)
		fmt.Fprintf(&b, "- `%s` (%s): %s\n", s.name, s.used, strings.Join(names, ", "))
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// CMD returns the `llm` command. It needs no config and no Odoo connection.
func CMD() *cobra.Command {
	var skill bool
	cmd := &cobra.Command{
		Use:   "llm",
		Short: "Print a usage guide for LLM agents (--skill)",
		Long: `With --skill, print a Markdown guide for coding agents on standard
output: how to call the CLI, the JSON contract, every error kind with what
to do about it, and the usual pitfalls. It starts with skill front matter
(name, description), so it can be saved as a Claude Code skill as is.

The error kinds and the JSON fields in the guide are generated from the
same sources as the CLI, so the guide always matches the installed
version. Without --skill the command fails with a validation error.`,
		Example: `  # Install the guide as a Claude Code skill
  mkdir -p ~/.claude/skills/odoo-work-cli
  odoo-work-cli llm --skill > ~/.claude/skills/odoo-work-cli/SKILL.md`,
		Annotations: map[string]string{
			manual.Group:          manual.GroupAgents,
			manual.Offline:        "true",
			manual.StaticExamples: "true",
			manual.Output:         "The guide, as Markdown.",
			manual.SeeAlso:        "how-to/80-agents.md, reference/20-json-output.md",
		},
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !skill {
				return apperr.New(apperr.KindValidation, "nothing to print: use `odoo-work-cli llm --skill`")
			}
			g, err := Guide()
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), g)
			return err
		},
	}
	cmd.Flags().BoolVar(&skill, "skill", false, "print the agent guide (Markdown) on stdout")
	return cmd
}
