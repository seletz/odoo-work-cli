---
title: Troubleshoot
icon: lucide/life-buoy
---
# Troubleshoot

## `auth` error, although the credentials look right

Most commands need an **API key**, not your login password. Odoo rejects a
password over XML-RPC without saying why (especially with 2FA enabled).
Create an API key and put it in `api-key` (`[op_secrets]`) or
`ODOO_PASSWORD`. See [Secrets](../explanation/secrets.md).

## `clock in` fails with "authentication failed"

`clock in` and `clock out` (without `--at`) log in like the browser, and that
login **rejects API keys**. Set your real login password as `password` in
`[op_secrets]` or `ODOO_WEB_PASSWORD`.

## "2FA is enabled but no TOTP secret configured"

Set `totp_secret` in `[op_secrets]` or `ODOO_TOTP_SECRET`: the base32 secret,
or the whole `otpauth://` URL, that Odoo showed when you enabled 2FA.

## "TOTP verification failed"

The secret is wrong, or your computer's clock is off. TOTP codes depend on
the time.

## "You are not allowed to create/modify 'Attendance'"

`clock in --at`, `clock out --at`, `clock edit` and `clock add` need the
*Attendance Officer* right. See
[Clock in and out → Permissions](30-clock-in-out.md#permissions). Plain
`clock in` and `clock out` are not affected.

## `secrets_locked` after 5 seconds

1Password is locked or not reachable, so the `op://` references in
`[op_secrets]` could not be resolved. Unlock 1Password and retry. Set
`ODOO_OP_TIMEOUT=10s` for a longer wait, or provide the credentials as
environment variables; then `op` is not called for them.

## `op inject` errors at startup

The config has `[op_secrets]`, the 1Password CLI is installed but not signed
in. Run `op signin`, or set the credentials as environment variables, or set
`ODOO_WORK_CLI_NO_OP=1`.

## `clock in` fails against a server with several databases

A known limitation: the web session is not reliably bound to the requested
database on multi-database servers
([#48](https://github.com/seletz/odoo-work-cli/issues/48)). `clock status`
and all other commands work.

## The TUI does not show an entry I booked on another company

It does: the TUI grid ignores `[models.timesheet]` filters and shows all of
your bookings. The `entries` and `timesheets` commands apply those filters;
check them with `odoo-work-cli config --merged`.

## The TUI says "terminal too narrow"

The weekly grid needs at least 68 columns. Widen the terminal window or reduce
the font size. On narrower terminals the TUI squeezes the project/task
column and shortens long names in the middle (`[EXA] …pha: Design`); open the
cell with ++enter++ to see the full name.

## Still stuck

Run the command with `--json` and look at `kind`; the table in
[Errors and exit codes](../reference/30-errors.md) says what each kind means.
Report bugs (`internal`) on
[GitHub](https://github.com/seletz/odoo-work-cli/issues).
