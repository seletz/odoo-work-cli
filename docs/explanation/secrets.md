---
title: Secrets
icon: lucide/key-round
weight: 20
---
# Secrets

`odoo-work-cli` needs up to three credentials. Which ones depends on the
commands you use, because the commands talk to Odoo in two different ways
([Two transports](transports.md)).

| Credential | `[op_secrets]` key | Environment variable | Needed for |
| --- | --- | --- | --- |
| **API key** | `api-key` | `ODOO_PASSWORD` | everything that reads or writes models: `projects`, `tasks`, `entries`, `timesheets`, `whoami`, `fields`, `clock status`, `clock list`, `clock in/out --at`, `clock edit`, `clock add`, the TUI |
| **Login password** | `password` | `ODOO_WEB_PASSWORD` | `clock in` / `clock out` without `--at` (and ++c++ in the TUI) |
| **TOTP secret** | `totp_secret` | `ODOO_TOTP_SECRET` | the 2FA step of that login, only if 2FA is enabled |

If you never clock in or out from the terminal, the API key is all you need.

Mind the names: `ODOO_PASSWORD` holds the **API key**, `ODOO_WEB_PASSWORD`
the actual password. The names are historical.

## Where secrets come from

From the environment or from 1Password, never from the config files.

**Environment variables** are the simplest way and have the highest
priority. They suit CI, containers and agents.

**1Password**: put `op://` references into `[op_secrets]`:

```toml
[op_secrets]
api-key     = "op://Employee/odoo/api-key"
password    = "op://Employee/odoo/password"
totp_secret = "op://Employee/odoo/totp_secret"
```

At startup, if the `op` CLI is installed, the CLI resolves all references in
one `op inject` call (a shell-out, no library). It asks only for the fields
that no environment variable sets, so with a complete environment `op` is not
called at all. Values without `op://` are used as they are.

## Why it fails fast when 1Password is locked

A locked 1Password makes `op inject` wait about 60 seconds for an unlock
prompt. For a command line tool, and even more for an agent, that looks like
a hang. The CLI bounds the call to 5 seconds (`ODOO_OP_TIMEOUT`) and fails
with the error kind `secrets_locked`: unlock and retry. Other `op` failures,
such as a wrong reference, stay `config` errors.

`ODOO_WORK_CLI_NO_OP=1` ignores `[op_secrets]` completely, for example on a
machine without 1Password.

## Why a TOTP secret

With 2FA, Odoo's login asks for a six-digit code after the password. A
command line tool cannot ask you every time you clock in, and an agent
cannot at all. With the TOTP secret, the CLI computes the code itself, like
your authenticator app. Treat the secret like the password: anyone with both
can log in as you.
