---
title: Configuration
icon: lucide/settings
weight: 40
---
# Configuration

Configuration comes from TOML files and environment variables. How the
layers combine and why: [Config layering](../explanation/config-layering.md).

## Files

| Order | File |
| --- | --- |
| 1 | `$XDG_CONFIG_HOME/odoo-work-cli/config.toml`, default `~/.config/odoo-work-cli/config.toml` |
| 2 | every `.odoo-work-cli.toml` from the filesystem root down to the current directory |
| 3 | `[op_secrets]`, resolved through 1Password |
| 4 | environment variables (highest priority) |

`--config FILE` skips 1 and 2 and loads only that file, then 3 and 4.

| Command | Does |
| --- | --- |
| `odoo-work-cli config` | list the files found, in merge order |
| `odoo-work-cli config --merged` | print the merged config, secrets left out |
| `odoo-work-cli config install` | write a commented default config to the global path |

## Top-level keys

| Key | Meaning |
| --- | --- |
| `url` | Odoo server URL, e.g. `https://odoo.example.com` |
| `database` | database name |
| `username` | login, usually an e-mail address |
| `bundesland` | German federal state for public holidays in the TUI; default `Baden-Württemberg` |

A `password` key outside `[op_secrets]` is rejected at load time. Secrets
come from `[op_secrets]` or environment variables only.

## `[op_secrets]` { #op_secrets }

Each value is an `op://` reference, resolved by the 1Password CLI at startup,
or a plain value used as is.

| Key | Meaning |
| --- | --- |
| `url`, `database`, `username` | as above |
| `api-key` | Odoo API key (XML-RPC) |
| `password` | Odoo login password (web session: `clock in` / `clock out`) |
| `totp_secret` | TOTP secret, base32 or `otpauth://` URL (only with 2FA) |

```toml
[op_secrets]
api-key     = "op://Employee/odoo/api-key"
password    = "op://Employee/odoo/password"
totp_secret = "op://Employee/odoo/totp_secret"
```

Which credential is needed when: [Secrets](../explanation/secrets.md).

## `[hours]` { #hours }

Limits for the colours of day and week totals in the TUI.

| Key | Default | Meaning |
| --- | --- | --- |
| `daily_low` | `6.0` | below: yellow |
| `daily_high` | `9.0` | above: red |
| `weekly_low` | `35.0` | below: yellow |
| `weekly_high` | `40.0` | above: red |

## `[company_colors]` { #company_colors }

Company name → colour of the row prefix in the TUI. A colour is an ANSI 256
colour number as a string (`"5"`) or a hex value (`"#ff8800"`). Companies not
listed use the default colour.

## `[keys]` { #keys }

Action → key or list of keys. See [TUI keys and views](50-tui-keys.md).

## `[models.<name>]` { #models }

Per-model settings for `project`, `task` and `timesheet`.

`filters` restrict every query of that model. They accumulate across config
files (AND); a child file's filter on the same field replaces the parent's.

```toml
[models.project]
filters = [
  { field = "company_id.name", op = "=", value = "Example Consulting" },
]

[models.task]
filters = [
  { field = "stage_id.name", op = "=", value = "In Progress" },
]
```

`op` is an Odoo domain operator: `=`, `!=`, `ilike`, `>`, `<`, `>=`, `<=`, …

`extra_fields` fetch additional Odoo fields; they show up as columns and in
`extra_fields` of the JSON output. Currently used for projects:

```toml
[models.project]
extra_fields = [
  { name = "product_owner", field = "x_studio_productowner", type = "many2one" },
]
```

`type` is the Odoo field type: `many2one`, `char`, `boolean`, `integer`,
`float`.

The TUI grid ignores `[models.timesheet]` filters and always shows all of
your own bookings, because entries on another company's project carry that
company's `company_id`. The `entries` and `timesheets` commands apply them.

## Environment variables

| Variable | Meaning |
| --- | --- |
| `ODOO_URL` | server URL |
| `ODOO_DATABASE` | database name |
| `ODOO_USERNAME` | login |
| `ODOO_PASSWORD` | the **API key** (XML-RPC) |
| `ODOO_WEB_PASSWORD` | the login password (web session) |
| `ODOO_TOTP_SECRET` | TOTP secret, base32 or `otpauth://` URL |
| `ODOO_WORK_CLI_NO_OP` | `1`: ignore `[op_secrets]`, never call `op` |
| `ODOO_OP_TIMEOUT` | timeout of the 1Password call, a Go duration; default `5s` |
| `ODOO_WORK_CLI_BACKEND` | `odoo` (default) or `fake`; `--backend` overrides it |
| `ODOO_WORK_CLI_FAKE_STATE` | state file of the fake backend |
| `XDG_CONFIG_HOME` | base directory of the global config file |

## Example

`odoo-work-cli config install` writes this file:

```toml
--8<-- "internal/config/default_config.toml"
```
