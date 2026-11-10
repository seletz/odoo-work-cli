---
title: Use it from an MCP client
icon: lucide/plug
---
# Use it from an MCP client

How an agent that talks to tools over the
[Model Context Protocol](https://modelcontextprotocol.io/) uses
`odoo-work-cli`.

`odoo-work-cli` has **no MCP server of its own**. It is a CLI with a stable
JSON contract, and that is enough for MCP-capable agents in two ways.

## Agents with a shell tool

Most coding agents (Claude Code, Codex and others) can run commands
themselves, next to their MCP tools. For them, `odoo-work-cli` needs no MCP
at all: give them the guide from `llm --skill` and let them call the CLI.
See [Use it from a coding agent](80-agents.md).

## Clients that only call MCP tools

If your client can only call MCP tools, connect it to an MCP server that runs
commands (a "shell" or "command runner" server) and allow it exactly one
program: `odoo-work-cli`. The agent then calls the CLI through that server's
tool. Configure it so that:

- only the `odoo-work-cli` binary may run, with arguments as a list, not a
  shell string;
- the environment carries the credentials (or `ODOO_WORK_CLI_BACKEND=fake`
  while you try it out) and nothing else;
- stdout and the exit code reach the agent; stderr may be dropped, because
  with `--json` nothing is written there.

Put the output of `odoo-work-cli llm --skill` into the server's or the
agent's instructions, so the agent knows the commands, the fields and the
rules.

## The contract the agent relies on

| What | Contract |
| --- | --- |
| Call | `odoo-work-cli <command> … --json` |
| Success | exit code `0`, one JSON value on stdout ([JSON output](../reference/20-json-output.md)) |
| Failure | exit code `1`, `{"error":{"kind":"…","message":"…"}}` on stdout ([Errors](../reference/30-errors.md)) |
| Empty result | `[]` |
| Schemas | `internal/output/schemas/*.schema.json`, draft 2020-12 |

If you write a thin tool layer of your own, map tools to commands one to one
and keep the JSON as it is:

| Tool | Command | Writes |
| --- | --- | --- |
| find projects | `projects --search TEXT --json` | no |
| list tasks | `tasks PROJECT_ID [--search TEXT] --json` | no |
| list entries | `entries --date D --json` / `--week W` | no |
| propose entry | `entries add … --dry-run --json` | no |
| book entry | `entries add … --json` | yes |
| list attendance | `clock list --date D --json` / `--week W` | no |
| backfill a day | `clock edit --date D --in T --out T --create-only --json` | yes |

Let the agent call the read-only tools freely and ask the user before every
tool that writes.

## Try it without Odoo

Set `ODOO_WORK_CLI_BACKEND=fake` in the environment of the command server.
Every command then works offline with made-up data; see
[Test without Odoo](70-test-without-odoo.md#the-fake-backend).
