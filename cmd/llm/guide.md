---
name: odoo-work-cli
description: Read and write Odoo 17 timesheets (project/task bookings) and attendance (clock in/out) with the odoo-work-cli CLI. Use when the user asks to book hours, backfill or check working time, or look up Odoo projects and tasks.
---

# odoo-work-cli

`odoo-work-cli` manages Odoo timesheet entries and attendance records. This guide is printed by `odoo-work-cli llm --skill`; it matches the installed binary.

## Ground rules

- **Always pass `--json`** and parse stdout only. Table output is for humans.
- Exit code is `0` on success and `1` on any error. With `--json` (or `--format json|jsonl`) an error is printed on **stdout** as `{"error":{"kind":"…","message":"…"}}`, and nothing goes to stderr. Branch on `kind`, not on the message.
- Look IDs up, never guess them: project and task IDs via `projects` / `tasks`, entry IDs via `entries`, attendance IDs via `clock list`.
- Check state before you write (`clock list`, `entries --date`), and show the user what you are about to change when it overwrites something.
- Never retry a failing write in a loop. Read the error kind (see below) and decide.
- Do not use the `tui` command: it is interactive.

## Commands

Global flags on every command: `--config FILE` (config file path, skips discovery; almost nobody needs it) and `--backend odoo|fake` (see [Testing without Odoo](#testing-without-odoo)).

Dates are `YYYY-MM-DD`, weeks are ISO weeks `YYYY-Www` (`2026-W38`). Hours are decimal (`2.5`) or `H:MM` (`2:30`). Times are entered in the **local** timezone.

### Reading

- `odoo-work-cli whoami --json`: the current user.
- `odoo-work-cli projects [--search TEXT] --json`: projects with their IDs. `--search` is a case-insensitive substring match on the project name or the customer, done by Odoo. Prefer it over fetching all projects and filtering yourself. No match gives `[]`. Example: `odoo-work-cli projects --search 2026-0042 --json`.
- `odoo-work-cli tasks [project-id] [--search TEXT] --json`: tasks, optionally of one project. `--search` is a case-insensitive substring match on the task name, combined with the project ID. Examples: `odoo-work-cli tasks 42 --json`, `odoo-work-cli tasks 42 --search review --json`.
- `odoo-work-cli timesheets [--week YYYY-Www] --json`: weekly summary (default: current week).
- `odoo-work-cli entries [--week YYYY-Www | --date D] [--project NAME] [--task NAME] [--status S] --json`: individual timesheet entries. `--date` overrides `--week`. `--project` and `--task` are case-insensitive substrings; `--status` is the validation status (`draft`, `validated`, …). Example: `odoo-work-cli entries --date 2026-09-14 --json`.
- `odoo-work-cli fields MODEL`: field metadata of an Odoo model (debugging only), e.g. `odoo-work-cli fields project.project`.

### Timesheet entries

- `odoo-work-cli entries add --project-id N --hours H --description "…" [--task-id N] [--date D] --json`: create an entry. `--project-id`, `--hours` and `--description` are required; `--date` defaults to today. Example: `odoo-work-cli entries add --project-id 42 --task-id 10 --date 2026-09-14 --hours 2:30 --description "Code review" --json`. Add `--dry-run` to show the proposal without writing it (see below).
- `odoo-work-cli entries update ID [--hours H] [--description "…"] [--date D] [--project-id N] [--task-id N] --json`: change an entry; only the flags you set are sent. Example: `odoo-work-cli entries update 100 --hours 1:15 --json`. `--dry-run` shows the merged result without writing.
- `--dry-run` (on `entries add` and `entries update`): runs the full validation, resolves project and task (an unknown ID gives the same error kind as the real call), and checks that the task belongs to the project (`validation` otherwise) and prints the entry that would be written; **nothing is written**. With `--json` it is the `entry` object plus `"dry_run": true` and `duplicates` (entries of the same day with the same project, task and description; `[]` if none); `id` is `null` for `add`. `employee`, `status` and `company` are only known after Odoo stored the entry and may be `null`.
- `odoo-work-cli entries delete ID --json`: delete an entry. Prints the entry as it was before deletion. A wrong ID gives `not_found` before anything is written.

### Attendance

- `odoo-work-cli clock status --json`: is the user clocked in, and today's periods.
- `odoo-work-cli clock in [--at TIME] --json`: start an attendance period now. `--at` backdates it (`HH:MM` today, or `YYYY-MM-DD HH:MM`); `--at` needs Attendance Officer rights.
- `odoo-work-cli clock out [--at TIME] --json`: end the running period; same `--at` rules. Example: `odoo-work-cli clock out --at "2026-09-14 17:15" --json`.
- `odoo-work-cli clock list [--date D | --week YYYY-Www] --json`: the records of a day (default today) or week, oldest first, each with ID, check-in, check-out and `worked_hours`. A record belongs to the day of its **check-in**. `--date` and `--week` together is a `validation` error. No records gives `[]`.
- `odoo-work-cli clock edit [--in TIME] [--out TIME] [--date D] [--id N] [--create-only] --json`: change or create a day's record. Needs at least one of `--in`/`--out`. Rules:
  - exactly one record on the day: it is **updated**;
  - no record: it is **created**, but only if both `--in` and `--out` are given;
  - several records: fails with `validation`; repeat with `--id N`;
  - `--create-only`: never overwrites. If the day already has a record the command changes nothing and fails with `conflict` (the message names the existing record). Requires both `--in` and `--out`; cannot be combined with `--id`.
  - `TIME` is `HH:MM` (on `--date`, default today) or `YYYY-MM-DD HH:MM`. Future times are rejected. Check-out must be after check-in.
  - Example: `odoo-work-cli clock edit --date 2026-09-14 --in 08:00 --out 16:30 --create-only --json`.

### Attendance: several records per day (`clock add`)

- `odoo-work-cli clock add --in TIME --out TIME [--date D] --json`: **always creates a new record**, also on a day that already has records (a day with a break is two records). Needs both `--in` and `--out`; `TIME` rules as for `clock edit`. Accepts `--format`/`--json`; returns the created record (`attendance` fields).
  - If the new span overlaps an existing record it fails with `conflict`, changes nothing, and the message names the overlapping record (ID and times). Checked: the day's records, a record from the day before that runs past midnight, and a running record (no check-out).
  - Touching edges are fine: `--out 12:00` on one record and `--in 12:00` on the next do not overlap.
  - Needs Attendance Officer rights, like `clock edit` (`access_denied` otherwise).
  - Example: `odoo-work-cli clock add --date 2026-09-14 --in 13:00 --out 17:30 --json`.

Which attendance write to use:

| Goal | Command |
| --- | --- |
| Add a record, keep what is there (day with a break, a missing half day) | `clock add` |
| Fill a day that must be empty; never touch an existing day | `clock edit --create-only` |
| Correct the times of an existing record | `clock edit` (with `--id N` when the day has several) |

### Setup and meta

- `odoo-work-cli config [--merged]`: show discovered config files, or the merged config (secrets omitted). `odoo-work-cli config install` writes a default config file. Do not edit config on your own; ask the user.
- `odoo-work-cli llm --skill`: print this guide. Needs no config or connection.
- `odoo-work-cli tui [--week YYYY-Www]`: interactive terminal UI. Not for agents.
- `odoo-work-cli completion`: shell completion scripts (not useful for agents).
- `odoo-work-cli --version`: print the installed version. To update a Homebrew install run `brew update && brew upgrade odoo-work-cli`; `brew upgrade` alone does not see a new release until `brew update` has refreshed the tap. Check the version afterwards with `odoo-work-cli --version`. Ask the user before updating.

## Output formats

Commands that list or write data (`projects`, `tasks`, `timesheets`, `entries` and its subcommands, `whoami`, `clock in|out|status|list|edit`) accept `--format table|json|jsonl` and `--json` (same as `--format json`). `--json` together with `--format table` is a `validation` error. `fields`, `config`, `llm` and `tui` have no JSON output.

- `json`: one JSON value. List commands print a top-level array, write commands the written object.
- `jsonl`: one object per line.
- **An empty result is `[]`** in both `json` and `jsonl`, never empty output.
- Names are snake_case. Fields may be added but are never renamed or removed. IDs are numbers; an unset optional value is `null`, never `0` or `""`. Durations come as `hours` and rounded `minutes`. Times are RFC 3339 with the local offset.
- `clock edit --json` returns the record as it is **now**, plus `before`: the record as it was, or `null` if the edit created it. Use `before` to tell the user what you overwrote.

Fields per object (`?` = may be absent):

{{SCHEMAS}}

## Errors

Envelope: `{"error":{"kind":"<kind>","message":"<text>"}}`, exit code 1. Kinds:

{{ERROR_KINDS}}

## Workflows

### Check what is booked

```sh
odoo-work-cli clock list --week 2026-W38 --json
odoo-work-cli entries --week 2026-W38 --json
```

### Backfill attendance for a day

1. `odoo-work-cli clock list --date 2026-09-14 --json`. `[]` means nothing is recorded.
2. `odoo-work-cli clock edit --create-only --date 2026-09-14 --in 08:00 --out 16:30 --json`.
3. On `conflict` the day already has a record: nothing was changed. Show the user the existing record (`clock list` output) and ask whether to keep it. Only if they say to replace it, run `clock edit --id N --in … --out …` (this overwrites; report `before`).

Always use `--create-only` when backfilling; do several days one by one and stop on the first unexpected error.

### Day with a break

1. `odoo-work-cli clock list --date 2026-09-14 --json` to see what is there.
2. `odoo-work-cli clock add --date 2026-09-14 --in 08:00 --out 12:00 --json`.
3. `odoo-work-cli clock add --date 2026-09-14 --in 13:00 --out 17:30 --json`.
4. `odoo-work-cli clock list --date 2026-09-14 --json` shows both records.

On `conflict` nothing was written: show the user the record named in the message and ask. Do not "fix" an overlap with `clock edit` unless the user says which record is wrong. If the day already has one record (e.g. 08:00–12:00), only step 3 is needed.

### Book timesheet entries

1. Find the project: `odoo-work-cli projects --search TEXT --json` (TEXT is part of the project name or customer, e.g. an order number), pick the `id`. Do not fetch all projects.
2. Find the task: `odoo-work-cli tasks PROJECT_ID --json`, or narrow it with `--search TEXT`, pick the `id` (optional).
3. Check for what is already booked that day: `odoo-work-cli entries --date 2026-09-14 --json`, to avoid duplicates.
4. Show a proposal: `odoo-work-cli entries add --project-id 42 --task-id 10 --date 2026-09-14 --hours 2:30 --description "…" --dry-run --json`. Nothing is written. Show the user the project, task, hours and description, and any `duplicates`, and wait for their approval.
5. Book: run the same command without `--dry-run`.
6. Fix a mistake with `entries update ID …` (preview it with `--dry-run`), remove with `entries delete ID`.

Ask the user when the project or task is ambiguous; do not pick one.

## MCP server

If your host connects to MCP servers, you can use the same commands as MCP tools instead of shell calls.

- `odoo-work-cli mcp`: an MCP server on stdin/stdout, started by the MCP client. Each tool is one command, run with `--json`: same fields, same error kinds. Read tools: `whoami`, `projects`, `tasks`, `entries`, `timesheets`, `clock_status`, `clock_list`. Write tools: `entries_add`, `entries_update`, `entries_delete`, `clock_in`, `clock_out`, `clock_add`, `clock_edit`. Arguments are named like the flags with `_` for `-` (`project_id`, `dry_run`, `create_only`); IDs are integers, `hours` is a number or `"H:MM"`.
- `--read-only` serves only the read tools.
- A failed call is a tool result with `isError: true` and the error envelope above, never a protocol error. Results over 32 KiB come in pages (`cursor`).
- Config and secrets load on the first tool call; a locked 1Password gives `secrets_locked` for that call, and the next call tries again. Once resolved, the secrets are kept for the life of the server; an `auth` error discards them and the next call resolves again.
- `--secrets eager` (env `ODOO_WORK_CLI_MCP_SECRETS`) resolves them at server start instead; the default is `lazy`.
- `odoo-work-cli mcp print [--binary PATH] [--command] [--read-only]`: print the client configuration (`mcpServers` JSON, or with `--command` the `claude mcp add` line). Writes nothing.

## Testing without Odoo

Use this to try commands or rehearse a workflow. Never use it for the user's real bookings.

- **Fake backend:** `ODOO_WORK_CLI_BACKEND=fake` (or `--backend fake`) runs every command against an offline fake: no network, no config, no 1Password. Its projects, tasks, entries and attendance records are **made up** and have nothing to do with the user's Odoo. Writes persist in a state file (`ODOO_WORK_CLI_FAKE_STATE`, default `$TMPDIR/odoo-work-cli-fake-state.json`), so list → add → list works across calls. Delete the file to reset to the seed data. Error kinds match the real backend (`not_found`, `conflict`, `validation`). Differences: `worked_hours` has no lunch deduction, configured filters are ignored, no access rights.
- **Real Odoo without 1Password:** when the env vars `ODOO_URL`, `ODOO_DATABASE`, `ODOO_USERNAME`, `ODOO_PASSWORD` (and, for plain `clock in|out`, `ODOO_WEB_PASSWORD`, `ODOO_TOTP_SECRET`) are set, 1Password is not asked for them. `ODOO_WORK_CLI_NO_OP=1` ignores `[op_secrets]` completely; a missing field is then a `config` error that names the env var.

## Pitfalls

- **`clock edit` changes an existing record** instead of adding a new one when the day already has exactly one. To add without risk of overwriting, use `--create-only`.
- **A check-out after midnight** (a shift that ends the next day) only works with the full format on `--out`: `--in "2026-09-14 22:00" --out "2026-09-15 01:30"`. A bare `HH:MM` means the same day as `--date`, and check-out must be after check-in. `clock list` shows such a record on its check-in day.
- **Attendance writes need rights.** `clock edit` and `clock in|out --at` write `hr.attendance` directly and require the Attendance Officer access right **and** the user set as their own attendance manager (`attendance_manager_id` on the employee). Without it you get `access_denied`. This is an Odoo admin task: tell the user, do not try to work around it. Plain `clock in`/`clock out` (no `--at`) work for everyone.
- **Locked 1Password**: secrets come from 1Password. When it is locked the CLI fails after about 5 s (`ODOO_OP_TIMEOUT` changes that) with `secrets_locked`. Stop and ask the user to unlock it. Do not retry in a loop. If the credentials are in env vars, `ODOO_WORK_CLI_NO_OP=1` skips 1Password.
- `worked_hours` excludes the lunch break of the working calendar, so it can be shorter than check-out minus check-in.
- Future times are rejected; plain `clock in` records "now".
- The CLI talks to whatever Odoo instance the config points to; writes are real bookings. Confirm with the user before bulk changes.
