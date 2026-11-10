---
title: Connect an MCP client
icon: lucide/plug
---
# Connect an MCP client

How to give an agent host that speaks the
[Model Context Protocol](https://modelcontextprotocol.io/) (Claude Code, an
IDE, a desktop app) the timesheet and attendance commands as MCP tools.

`odoo-work-cli mcp` is an MCP server on standard input and output. The client
starts it as a child process; there is no port and no daemon. Every tool is one
command of the CLI, run with `--json`, so the fields and error kinds are the
ones you already know. The full list is in the
[MCP tool reference](../reference/60-mcp-tools.md).

## Register the server

=== "Claude Code"

    ```bash
    claude mcp add --scope user odoo-work-cli -- odoo-work-cli mcp
    ```

    `odoo-work-cli mcp print --command` prints exactly this line. Check it
    with `claude mcp list`; the server shows as connected.

=== "Project `.mcp.json`"

    ```bash
    odoo-work-cli mcp print
    ```

    prints the `mcpServers` block to merge into the project's `.mcp.json` (or
    another client's MCP settings):

    ```json
    {
      "mcpServers": {
        "odoo-work-cli": {
          "type": "stdio",
          "command": "odoo-work-cli",
          "args": ["mcp"]
        }
      }
    }
    ```

If the client does not search `PATH`, use an absolute path:
`odoo-work-cli mcp print --binary "$(command -v odoo-work-cli)"`.

The server reads the same configuration as the CLI (config files, environment,
1Password). Global flags go before `mcp` and apply to every tool call, for
example `odoo-work-cli --config ~/work.toml mcp`.

## Read only

```bash
claude mcp add --scope user odoo-work-cli -- odoo-work-cli mcp --read-only
```

`--read-only` serves only the read tools (`whoami`, `projects`, `tasks`,
`entries`, `timesheets`, `clock_status`, `clock_list`). The write tools are
not listed and cannot be called. `mcp print --read-only` adds the flag to the
printed configuration.

## What the agent sees

- **Annotations.** Read tools carry `readOnlyHint`. `entries_update`,
  `entries_delete` and `clock_edit` carry `destructiveHint`: they overwrite or
  remove data. Clients use these to decide what needs a confirmation; let the
  read tools run freely and confirm every write.
- **Dry runs.** `entries_add` and `entries_update` take `dry_run: true`: the
  entry is resolved and checked, duplicates are listed, nothing is written.
  The server instructions tell the agent to propose with a dry run first.
- **Errors.** A failed call is a normal tool result with `isError: true` and
  the [error envelope](../reference/30-errors.md):
  `{"error":{"kind":"not_found","message":"timesheet entry 424242 does not exist"}}`.
  Bad arguments are `validation` and name the argument, not a flag.
- **Pages.** A result over 32 KiB comes in pages; the second content item
  says which page and gives the `cursor` for the next.

## 1Password and startup

The server starts without loading the configuration or secrets. They load on
the first tool call that needs them, as in the CLI. If 1Password is locked,
that call returns `secrets_locked` and the server keeps running: unlock
1Password and the next call loads. After that the connection is kept for the
life of the server, so 1Password is not asked again for every call.

## Try it without Odoo

Register the server against the offline fake backend:

```bash
claude mcp add odoo-work-cli-fake -- odoo-work-cli --backend fake mcp
```

Every tool then works with made-up data; see
[Test without Odoo](70-test-without-odoo.md#the-fake-backend).

To watch the protocol by hand:

```bash
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"projects","arguments":{"search":"alpha"}}}' \
  | odoo-work-cli --backend fake mcp
```

## Agents with a shell tool

Coding agents that can run commands need no MCP server: give them the guide
from `llm --skill` and let them call the CLI. See
[Use it from a coding agent](80-agents.md). Both ways use the same commands
and the same JSON; the MCP server is for hosts that only call tools, or when
you want the client's per-tool permissions and the `--read-only` switch.
