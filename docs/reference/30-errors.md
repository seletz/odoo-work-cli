---
title: Errors and exit codes
icon: lucide/triangle-alert
weight: 30
---
# Errors and exit codes

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | success |
| `1` | any error: usage, validation, Odoo, network, configuration |

## The error envelope

With `--json` or `--format json|jsonl`, a failure prints exactly one JSON
object on **stdout**, nothing on stderr, and exits with `1`:

```console
$ odoo-work-cli entries delete 999 --json
--8<-- "docs/assets/cli/error-not-found.txt"
$ odoo-work-cli entries add --project-id 1 --hours abc --description x --json
--8<-- "docs/assets/cli/error-validation.txt"
```

`kind` is stable and meant for branching. `message` is for humans and may
change. Schema: `error.schema.json`.

Without JSON output, the error goes to stderr as `Error: …`; usage errors add
the command's usage.

## Kinds

| Kind | Meaning | What to do |
| --- | --- | --- |
| `auth` | Authentication failed: wrong API key or login password, TOTP rejected | Fix the credentials; do not retry |
| `access_denied` | Authenticated, but Odoo's access rules forbid it (e.g. `clock edit` without Attendance Officer rights) | Ask an administrator; see [Permissions](../how-to/30-clock-in-out.md#permissions) |
| `not_found` | The record does not exist or is not visible (entry, attendance record, employee) | Look the ID up again |
| `validation` | Invalid input: unknown flag, bad argument, malformed date, time or hours, a time in the future, an unknown project, a value Odoo rejects | Fix the input; the message says what |
| `conflict` | Contradicts the current state: `clock in` while clocked in, `clock out` while not, `clock edit --create-only` on a day with a record, `clock add` overlapping a record | Re-read the state (`clock list`, `clock status`) and decide |
| `network` | Odoo cannot be reached: DNS, connection refused, timeout, TLS | Retry once, then check the URL and network |
| `config` | The local configuration is missing or invalid (no config file, no URL, web password not set) | Fix the config; see [Configuration](40-configuration.md) |
| `secrets_locked` | 1Password is locked or unreachable, so `op://` references could not be resolved (after 5 s, `ODOO_OP_TIMEOUT`) | Unlock 1Password |
| `internal` | Anything else: an unexpected Odoo response or a bug | [Report it](https://github.com/seletz/odoo-work-cli/issues) |
