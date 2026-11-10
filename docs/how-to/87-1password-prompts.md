---
title: 1Password prompts
icon: lucide/key-round
---
# Predictable 1Password prompts

How to know when 1Password asks for approval, and how to set things up so it
does not ask while you or an agent work.

## When prompts come

Secrets in `[op_secrets]` are resolved with one `op inject` call.

| How you run it | `op inject` calls |
| --- | --- |
| A CLI command | one per process, so one per call |
| `odoo-work-cli mcp`, lazy (default) | one per server process, on the first tool call |
| `odoo-work-cli mcp --secrets eager` | one per server process, at server start |

An agent that runs `odoo-work-cli whoami --json` three times causes three
calls. The MCP server keeps the resolved secrets for its lifetime, even when
the Odoo login fails; only an `auth` error makes it resolve again, once.

1Password's app integration authorizes per process and terminal, so each new
process without an approved terminal can prompt. See the
[1Password CLI documentation](https://developer.1password.com/docs/cli/app-integration/)
for how approval and its timeout work. odoo-work-cli does not cache secrets
across processes, on disk or in a keychain.

## Setups

### (a) Register the MCP server, resolve at start

```bash
claude mcp add --scope user odoo-work-cli -- odoo-work-cli mcp --secrets eager
```

One prompt per client session, when the client starts the server, not in the
middle of a task. If you miss it, calls return `secrets_locked`; unlock
1Password and the next call tries again.

### (b) Start the agent under `op run`

`op run` puts the secrets in the environment of the agent. odoo-work-cli does
not call `op` for fields that environment variables already set, so there are
zero prompts during work: one prompt when you start the agent.

Create an env file with references only (use your own vault and item):

```ini
# odoo-work-cli.env
ODOO_URL=op://VAULT/ITEM/url
ODOO_DATABASE=op://VAULT/ITEM/database
ODOO_USERNAME=op://VAULT/ITEM/username
ODOO_PASSWORD=op://VAULT/ITEM/api-key
ODOO_WEB_PASSWORD=op://VAULT/ITEM/password
ODOO_TOTP_SECRET=op://VAULT/ITEM/totp-secret
```

`ODOO_PASSWORD` is the API key; `ODOO_WEB_PASSWORD` and `ODOO_TOTP_SECRET` are
only needed for plain `clock in` and `clock out`. Then start the agent:

```bash
op run --env-file=odoo-work-cli.env -- claude
```

Every `odoo-work-cli` call and MCP server below it inherits the variables.

### (c) No 1Password at runtime

```bash
export ODOO_WORK_CLI_NO_OP=1
```

with the `ODOO_*` variables set by your own means. `[op_secrets]` is then
ignored and `op` is never called; a missing field is a `config` error that
names the variable. See [Test without Odoo](70-test-without-odoo.md).
