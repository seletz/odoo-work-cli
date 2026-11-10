---
title: Test without Odoo
icon: lucide/flask-conical
---
# Test without Odoo

Two switches let you or an agent run the CLI without an unlocked 1Password
and, if needed, without any Odoo server.

## Real Odoo, no 1Password

Set the credentials as environment variables. `op` is only called for
`[op_secrets]` fields that no environment variable sets, so with a complete
environment it is never called:

```sh
set -a; source .env; set +a      # a file with ODOO_URL, ODOO_DATABASE, …
odoo-work-cli whoami --json      # works with 1Password locked
```

`ODOO_WORK_CLI_NO_OP=1` ignores `[op_secrets]` completely. A missing field is
then a `config` error that names its variable, for example
`missing required config fields: Password (ODOO_PASSWORD)`.

## The fake backend

`ODOO_WORK_CLI_BACKEND=fake` (or `--backend fake`) runs every command against
a built-in fake Odoo: no network, no config files, no 1Password. Its data is
made up.

```sh
export ODOO_WORK_CLI_BACKEND=fake
odoo-work-cli projects --json
odoo-work-cli clock edit --date 2026-01-07 --in 08:00 --out 12:00 --create-only --json
odoo-work-cli clock add --date 2026-01-07 --in 13:00 --out 17:00 --json
odoo-work-cli clock list --date 2026-01-07 --json
odoo-work-cli tui
```

| Variable | Meaning |
| --- | --- |
| `ODOO_WORK_CLI_BACKEND` | `fake` or `odoo` (default). `--backend` overrides it. |
| `ODOO_WORK_CLI_FAKE_STATE` | The state file. Default: `$TMPDIR/odoo-work-cli-fake-state.json`. |

Writes persist in the state file, so list → add → list works across calls.

- **Reset:** delete the state file. The next call seeds it again, with
  entries and attendance on the three workdays before today.
- **Isolate test runs:** give each run its own `ODOO_WORK_CLI_FAKE_STATE`.
- **The docs' data:** use [`demo-state.json`](../assets/demo-state.json), the
  state behind every example and screenshot here (week 2026-W15). See
  [Try it offline](../tutorials/10-try-it-offline.md).

### How the fake differs from Odoo

It behaves like Odoo where commands depend on it: unique IDs, `not_found` for
unknown IDs, `conflict` for a double `clock in` or a `clock out` without an
open record, `validation` for an unknown project or task, overlapping
attendance records or a check-out before the check-in.

It differs here:

- `worked_hours` is check-out minus check-in; Odoo subtracts the lunch break.
- Configured `filters` and `extra_fields` are ignored.
- There are no access rights.
- `fields` knows only a few fields of the main models.
