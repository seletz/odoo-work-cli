package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/mcp"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/version"
	"github.com/spf13/cobra"
)

// MCPServerName is the name of the server in initialize and in the client
// configuration printed by `mcp print`.
const MCPServerName = "odoo-work-cli"

// mcpInstructions is what a client shows the model about the server: a
// short version of the `llm --skill` guide.
const mcpInstructions = `Odoo 17 timesheets (project/task bookings) and attendance (clock in/out) of the current user. ` +
	`Every tool runs the odoo-work-cli command of the same name with --json and returns its JSON unchanged; ` +
	`the JSON fields are those of the CLI (see odoo-work-cli llm --skill). ` +
	`Look IDs up, never guess them: projects (search) and tasks for project and task IDs, entries for entry IDs, clock_list for attendance IDs. ` +
	`Before you book, propose with entries_add dry_run=true (it resolves project and task and lists duplicates without writing), show the user, then call entries_add without dry_run. ` +
	`Check state before writing (entries with date, clock_list) and ask the user before every tool that writes, above all entries_update, entries_delete and clock_edit, which overwrite or remove data. ` +
	`A failed call is a tool result with isError true and {"error":{"kind":…,"message":…}}: branch on kind. ` +
	`validation: fix the arguments; not_found: look the ID up again; conflict: re-read the state (clock_status, clock_list) and decide; ` +
	`secrets_locked: stop and ask the user to unlock 1Password; auth, config, access_denied: ask the user; network: retry once. Never retry a write in a loop. ` +
	`Results over 32 KiB come in pages: the second content item has next.cursor; pass it as cursor with the same other arguments.`

// pagingNote ends the description of every list tool.
const pagingNote = " A result over 32 KiB comes in pages: page and next are in the second content block; pass next.cursor as cursor, with the same other arguments, for the next page."

var (
	pCursor = mcp.Param{Name: "cursor", Type: "string", Server: true,
		Description: "next.cursor from the second content block of the previous page of the same call; omit for the first page"}
	pWeek  = mcp.Param{Name: "week", Type: "string", Description: "ISO week, e.g. 2026-W10 (default: the current week)"}
	pHours = mcp.Param{Name: "hours", Type: "string", Number: true, Description: "hours worked: a number (2.5) or H:MM (\"2:30\")"}
	pDry   = mcp.Param{Name: "dry_run", Type: "boolean", Description: "resolve and validate, print the entry that would be written and its duplicates, write nothing"}
	pAt    = mcp.Param{Name: "at", Type: "string", Description: "a past time (HH:MM or YYYY-MM-DD HH:MM) instead of now; needs attendance officer rights"}
)

// mcpTools is the tool interface of `odoo-work-cli mcp`: each tool is one
// command with a fixed set of flags. Names and arguments only grow within
// a major version; an argument is named like its flag with _ for -.
func mcpTools() []mcp.Tool {
	read := func(t mcp.Tool) mcp.Tool {
		t.ReadOnly, t.Idempotent = true, true
		t.FixedFlags = []string{"--json"}
		return t
	}
	write := func(t mcp.Tool) mcp.Tool {
		t.FixedFlags = []string{"--json"}
		return t
	}
	return []mcp.Tool{
		read(mcp.Tool{Name: "whoami", Title: "Current Odoo user", Command: []string{"whoami"},
			Description: "The Odoo user the server acts as: id, name, login, email, company. Same JSON as `odoo-work-cli whoami --json`."}),
		read(mcp.Tool{Name: "projects", Title: "List projects", Command: []string{"projects"}, Units: []string{""},
			Description: "Projects (with the configured filters), with id, name, customer, company, stage. Use search to find the project ID to book on. Same JSON as `odoo-work-cli projects --json`." + pagingNote,
			Params: []mcp.Param{pCursor,
				{Name: "search", Type: "string", Description: "only projects whose name or customer contains this text (case-insensitive)"}}}),
		read(mcp.Tool{Name: "tasks", Title: "List tasks", Command: []string{"tasks"}, Units: []string{""},
			Description: "Tasks with id, name, project_id, project, stage; of one project with project_id. Same JSON as `odoo-work-cli tasks --json`." + pagingNote,
			Params: []mcp.Param{pCursor,
				{Name: "project_id", Type: "integer", Positional: true, Description: "only tasks of this project"},
				{Name: "search", Type: "string", Description: "only tasks whose name contains this text (case-insensitive)"}}}),
		read(mcp.Tool{Name: "entries", Title: "List timesheet entries", Command: []string{"entries"}, Units: []string{""},
			Narrow:      "a single date instead of a week, or a project, task or status filter",
			Description: "The user's timesheet entries of one week (default: the current one) or one date, with id, date, project, task, description, hours, status. Same JSON as `odoo-work-cli entries --json`." + pagingNote,
			Params: []mcp.Param{pCursor, pWeek,
				{Name: "date", Type: "string", Description: "one day, YYYY-MM-DD (overrides week)"},
				{Name: "project", Type: "string", Description: "only entries whose project name contains this text (case-insensitive)"},
				{Name: "task", Type: "string", Description: "only entries whose task name contains this text (case-insensitive)"},
				{Name: "status", Type: "string", Description: "only entries with this validation status, e.g. draft or validated"}}}),
		read(mcp.Tool{Name: "timesheets", Title: "Weekly timesheet", Command: []string{"timesheets"}, Units: []string{""},
			Description: "All timesheet entries of one ISO week (default: the current one). Same JSON as `odoo-work-cli timesheets --json`." + pagingNote,
			Params:      []mcp.Param{pCursor, pWeek}}),
		read(mcp.Tool{Name: "clock_status", Title: "Attendance status", Command: []string{"clock", "status"},
			Description: "Whether the user is clocked in, since when, and today's attendance periods and total. Same JSON as `odoo-work-cli clock status --json`."}),
		read(mcp.Tool{Name: "clock_list", Title: "List attendance records", Command: []string{"clock", "list"}, Units: []string{""},
			Description: "Attendance records of one day (default today) or one ISO week, with id, check_in, check_out, running, worked_hours. Same JSON as `odoo-work-cli clock list --json`." + pagingNote,
			Params: []mcp.Param{pCursor,
				{Name: "date", Type: "string", Description: "the day, YYYY-MM-DD (default today); not together with week"},
				{Name: "week", Type: "string", Description: "an ISO week, e.g. 2026-W10; not together with date"}}}),

		write(mcp.Tool{Name: "entries_add", Title: "Book a timesheet entry", Command: []string{"entries", "add"},
			Description: "Create a timesheet entry and return it. With dry_run=true nothing is written: the entry that would be created comes back with dry_run true and the duplicates (same day, project, task and description). Propose with dry_run first. Same JSON as `odoo-work-cli entries add --json`.",
			Params: []mcp.Param{
				{Name: "project_id", Type: "integer", Required: true, Description: "project ID (from projects)"},
				{Name: "task_id", Type: "integer", Description: "task ID of that project (from tasks)"},
				{Name: "date", Type: "string", Description: "YYYY-MM-DD (default today)"},
				withRequired(pHours),
				{Name: "description", Type: "string", Required: true, Description: "what was done"},
				pDry}}),
		write(mcp.Tool{Name: "entries_update", Title: "Change a timesheet entry", Command: []string{"entries", "update"}, Destructive: true, Idempotent: true,
			Description: "Overwrite the given fields of a timesheet entry and return it; fields not given stay. With dry_run=true nothing is written: the merged entry comes back with dry_run true. Same JSON as `odoo-work-cli entries update --json`.",
			Params: []mcp.Param{
				{Name: "id", Type: "integer", Required: true, Positional: true, Description: "entry ID (from entries)"},
				{Name: "project_id", Type: "integer", Description: "new project ID"},
				{Name: "task_id", Type: "integer", Description: "new task ID"},
				{Name: "date", Type: "string", Description: "new date, YYYY-MM-DD"},
				pHours,
				{Name: "description", Type: "string", Description: "new description"},
				pDry}}),
		write(mcp.Tool{Name: "entries_delete", Title: "Delete a timesheet entry", Command: []string{"entries", "delete"}, Destructive: true, Idempotent: true,
			Description: "Delete a timesheet entry and return it as it was. Same JSON as `odoo-work-cli entries delete --json`.",
			Params: []mcp.Param{
				{Name: "id", Type: "integer", Required: true, Positional: true, Description: "entry ID (from entries)"}}}),
		write(mcp.Tool{Name: "clock_in", Title: "Clock in", Command: []string{"clock", "in"},
			Description: "Start attendance now (or at a past time) and return the running record. conflict when already clocked in. Same JSON as `odoo-work-cli clock in --json`.",
			Params:      []mcp.Param{pAt}}),
		write(mcp.Tool{Name: "clock_out", Title: "Clock out", Command: []string{"clock", "out"},
			Description: "End the running attendance now (or at a past time) and return the record. conflict when not clocked in. Same JSON as `odoo-work-cli clock out --json`.",
			Params:      []mcp.Param{pAt}}),
		write(mcp.Tool{Name: "clock_add", Title: "Add an attendance record", Command: []string{"clock", "add"},
			Description: "Create one attendance record with check-in and check-out, for several records a day (e.g. around a break). conflict, and nothing written, when it overlaps a record. Same JSON as `odoo-work-cli clock add --json`.",
			Params: []mcp.Param{
				{Name: "date", Type: "string", Description: "the day, YYYY-MM-DD (default today)"},
				{Name: "in", Type: "string", Required: true, Description: "check-in, HH:MM or YYYY-MM-DD HH:MM"},
				{Name: "out", Type: "string", Required: true, Description: "check-out, HH:MM or YYYY-MM-DD HH:MM (a later day for overnight)"}}}),
		write(mcp.Tool{Name: "clock_edit", Title: "Set a day's attendance", Command: []string{"clock", "edit"}, Destructive: true, Idempotent: true,
			Description: "Set check-in and/or check-out of the day's attendance record, or create it when the day has none; returns before and after. Overwrites an existing record unless create_only. Same JSON as `odoo-work-cli clock edit --json`.",
			Params: []mcp.Param{
				{Name: "date", Type: "string", Description: "the day, YYYY-MM-DD (default today)"},
				{Name: "in", Type: "string", Description: "new check-in, HH:MM or YYYY-MM-DD HH:MM"},
				{Name: "out", Type: "string", Description: "new check-out, HH:MM or YYYY-MM-DD HH:MM"},
				{Name: "id", Type: "integer", Description: "the record to edit when the day has several (from clock_list)"},
				{Name: "create_only", Type: "boolean", Description: "only create; conflict, and nothing changed, when the day already has a record"}}}),
	}
}

func withRequired(p mcp.Param) mcp.Param {
	p.Required = true
	return p
}

// readOnlyTools keeps the tools that change nothing.
func readOnlyTools(tools []mcp.Tool) []mcp.Tool {
	return slices.DeleteFunc(tools, func(t mcp.Tool) bool { return !t.ReadOnly })
}

// keepOpen shares one client between calls: the per-call Deps.Close must
// not close it. The server closes the real client when it ends.
type keepOpen struct{ odoo.Client }

// Close does nothing; see keepOpen.
func (keepOpen) Close() {}

// sharedSetup returns a setup that loads config and client on its first
// successful call and returns them from then on. A failure (1Password
// locked) is not kept, so the next call tries again. closeFn closes the
// loaded client.
func sharedSetup(setup setupFunc) (shared setupFunc, closeFn func()) {
	var mu sync.Mutex
	var cfg *config.Config
	var client odoo.Client
	shared = func(cfgFile string) (*config.Config, odoo.Client, error) {
		mu.Lock()
		defer mu.Unlock()
		if client == nil {
			c, cl, err := setup(cfgFile)
			if err != nil {
				return nil, nil, err
			}
			cfg, client = c, cl
		}
		return cfg, keepOpen{client}, nil
	}
	closeFn = func() {
		mu.Lock()
		defer mu.Unlock()
		if client != nil {
			client.Close()
			client = nil
		}
	}
	return shared, closeFn
}

// newMCPServer returns the MCP server and a function that releases the
// Odoo client. Every tool call runs the CLI in-process with a fresh command
// tree (no flag state survives a call) and the global flags the server was
// started with; config and secrets load on the first call that needs them.
//
// :param global: persistent flags of the server (--config, --backend)
// :param setup: config + client factory
// :param readOnly: serve only the tools that change nothing
// :returns: the server and its close function
func newMCPServer(global []string, setup setupFunc, readOnly bool) (*mcp.Server, func()) {
	shared, closeFn := sharedSetup(setup)
	tools := mcpTools()
	if readOnly {
		tools = readOnlyTools(tools)
	}
	runCLI := func(_ context.Context, argv []string) (string, bool) {
		var out bytes.Buffer
		code := run(append(slices.Clone(global), argv...), &out, io.Discard, shared)
		return out.String(), code != 0
	}
	return &mcp.Server{Name: MCPServerName, Version: version.Version, Instructions: mcpInstructions,
		Tools: tools, Run: runCLI}, closeFn
}

// mcpServerEntry is the entry of the "mcpServers" object of an MCP client.
type mcpServerEntry struct {
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func mcpCMD(setup setupFunc, cfgFile, backend *string) *cobra.Command {
	var readOnly bool
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run the MCP server on stdin/stdout (started by the MCP client)",
		Long: `Run a Model Context Protocol (MCP) server on standard input and output.

The MCP client (Claude Code, an IDE, any agent host) starts it as a child
process and talks newline-delimited JSON-RPC 2.0 with it: initialize, ping,
tools/list and tools/call. There is no port and no daemon. Protocol revisions
2025-11-25, 2025-06-18, 2025-03-26 and 2024-11-05 are supported.

Every tool is one command of this CLI. A call runs that command in-process
with --json, so validation, output fields and error kinds are exactly those of
the CLI. Read tools: whoami, projects, tasks, entries, timesheets,
clock_status, clock_list. Write tools: entries_add, entries_update,
entries_delete, clock_in, clock_out, clock_add, clock_edit. The tools carry
the MCP annotations readOnlyHint, destructiveHint and idempotentHint;
entries_update, entries_delete and clock_edit are destructive.

A failed call is not a protocol error: it is a tool result with isError set
and the JSON error envelope {"error":{"kind":"…","message":"…"}}. Results
larger than 32 KiB come in pages; the second content item carries the cursor
of the next page.

Configuration and secrets load on the first tool call that needs them, not at
start: the server starts and lists its tools even with 1Password locked, and
such a call reports secrets_locked. The next call tries again; once loaded, the
connection is kept for the life of the server. The global flags --config and
--backend given before mcp apply to every call.

Nothing is written to standard output except protocol messages; standard
error stays empty.`,
		Example: `  # Register the server for Claude Code (user scope)
  claude mcp add --scope user odoo-work-cli -- odoo-work-cli mcp

  # Only the read tools, against the offline fake backend
  odoo-work-cli --backend fake mcp --read-only

  # One request by hand
  echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | odoo-work-cli mcp`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var global []string
			if *cfgFile != "" {
				global = append(global, "--config="+*cfgFile)
			}
			if *backend != "" {
				global = append(global, "--backend="+*backend)
			}
			s, closeFn := newMCPServer(global, setup, readOnly)
			defer closeFn()
			return s.Serve(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&readOnly, "read-only", false, "serve only the read tools (no entries_add, entries_update, entries_delete, clock_in, clock_out, clock_add, clock_edit)")
	cmd.AddCommand(mcpPrintCMD())
	return cmd
}

func mcpPrintCMD() *cobra.Command {
	var binary string
	var asCommand, readOnly bool
	cmd := &cobra.Command{
		Use:   "print",
		Short: "Print the MCP client configuration for this server",
		Long: `Print the configuration an MCP client needs to start this server. Nothing is
written; copy the output where your client expects it.

By default the output is the JSON "mcpServers" block, as it goes into a
project's .mcp.json (Claude Code) or the MCP settings of another client:

  {"mcpServers": {"odoo-work-cli": {"type": "stdio", "command": "odoo-work-cli", "args": ["mcp"]}}}

With --command the output is the "claude mcp add" command line that
registers the server for the current user in Claude Code.

Use --binary for an absolute path when the client does not search PATH, and
--read-only to register a server without the write tools.`,
		Example: `  # JSON block for .mcp.json
  odoo-work-cli mcp print

  # Register with Claude Code, read tools only
  $(odoo-work-cli mcp print --command --read-only)

  # Absolute path for clients without PATH lookup
  odoo-work-cli mcp print --binary "$(command -v odoo-work-cli)"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := []string{"mcp"}
			if readOnly {
				args = append(args, "--read-only")
			}
			w := cmd.OutOrStdout()
			if asCommand {
				_, err := fmt.Fprintln(w, "claude mcp add --scope user "+MCPServerName+" -- "+binary+" "+strings.Join(args, " "))
				return err
			}
			cfg := map[string]any{"mcpServers": map[string]any{
				MCPServerName: mcpServerEntry{Type: "stdio", Command: binary, Args: args},
			}}
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			if err := enc.Encode(cfg); err != nil {
				return apperr.Wrap(apperr.KindInternal, err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&binary, "binary", MCPServerName, "command used in the entry, e.g. an absolute path")
	cmd.Flags().BoolVar(&asCommand, "command", false, "print the claude mcp add command line instead of JSON")
	cmd.Flags().BoolVar(&readOnly, "read-only", false, "register the server with --read-only")
	return cmd
}
