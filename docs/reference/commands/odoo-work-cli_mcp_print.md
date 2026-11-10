# odoo-work-cli mcp print

Print the MCP client configuration for this server

### Synopsis

Print the configuration an MCP client needs to start this server. Nothing is
written; copy the output where your client expects it.

By default the output is the JSON "mcpServers" block, as it goes into a
project's .mcp.json (Claude Code) or the MCP settings of another client:

  {"mcpServers": {"odoo-work-cli": {"type": "stdio", "command": "odoo-work-cli", "args": ["mcp"]}}}

With --command the output is the "claude mcp add" command line that
registers the server for the current user in Claude Code.

Use --binary for an absolute path when the client does not search PATH, and
--read-only to register a server without the write tools.

```
odoo-work-cli mcp print [flags]
```

### Examples

```
  # JSON block for .mcp.json
  odoo-work-cli mcp print

  # Register with Claude Code, read tools only
  $(odoo-work-cli mcp print --command --read-only)

  # Absolute path for clients without PATH lookup
  odoo-work-cli mcp print --binary "$(command -v odoo-work-cli)"
```

### Options

```
      --binary string   command used in the entry, e.g. an absolute path (default "odoo-work-cli")
      --command         print the claude mcp add command line instead of JSON
  -h, --help            help for print
      --read-only       register the server with --read-only
```

### Options inherited from parent commands

```
      --backend string   backend: odoo (default) or fake (offline, made-up data; env ODOO_WORK_CLI_BACKEND)
      --config string    config file path (skip discovery)
```

### SEE ALSO

* [odoo-work-cli mcp](odoo-work-cli_mcp.md)	 - Run the MCP server on stdin/stdout (started by the MCP client)

