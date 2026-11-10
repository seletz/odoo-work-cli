# odoo-work-cli mcp

Run the MCP server on stdin/stdout (started by the MCP client)

### Synopsis

Run a Model Context Protocol (MCP) server on standard input and output.

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
error stays empty.

```
odoo-work-cli mcp [flags]
```

### Examples

```
  # Register the server for Claude Code (user scope)
  claude mcp add --scope user odoo-work-cli -- odoo-work-cli mcp

  # Only the read tools, against the offline fake backend
  odoo-work-cli --backend fake mcp --read-only

  # One request by hand
  echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | odoo-work-cli mcp
```

### Options

```
  -h, --help        help for mcp
      --read-only   serve only the read tools (no entries_add, entries_update, entries_delete, clock_in, clock_out, clock_add, clock_edit)
```

### Options inherited from parent commands

```
      --backend string   backend: odoo (default) or fake (offline, made-up data; env ODOO_WORK_CLI_BACKEND)
      --config string    config file path (skip discovery)
```

### SEE ALSO

* [odoo-work-cli](odoo-work-cli.md)	 - CLI for managing Odoo 17 timesheets
* [odoo-work-cli mcp print](odoo-work-cli_mcp_print.md)	 - Print the MCP client configuration for this server

