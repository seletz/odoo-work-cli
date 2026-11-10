---
title: JSON output
icon: lucide/braces
weight: 20
---
# JSON output

## Formats

Every command that reads or writes data accepts `--format table|json|jsonl`
and `--json` (= `--format json`). `--json` together with `--format table` is
a `validation` error. `fields`, `config`, `llm` and `tui` have no JSON
output.

| Format | Output |
| --- | --- |
| `table` | for humans; the default |
| `json` | one indented JSON value: an array for list commands, an object for writes |
| `jsonl` | one compact object per line |

## Conventions

- **Field names are stable** (`snake_case`). New fields may be added;
  existing fields are never renamed or removed.
- **IDs are numbers.** An unset optional relation or string (no task, no
  customer, no `status`) is `null`, never `0` or `""`.
- **Durations come twice:** fractional `hours` (Odoo's value) and whole
  `minutes` (rounded). Attendance uses `worked_hours` and `worked_minutes`;
  `clock status` adds `elapsed_minutes` for the running period.
- **Timestamps** (`check_in`, `check_out`) are RFC 3339 with the local UTC
  offset, e.g. `2026-04-08T08:30:00+02:00`. Dates are `YYYY-MM-DD`.
- **An empty result is `[]`**, never empty output, also in `jsonl`.
- **Errors** are one envelope on stdout; see [Errors](30-errors.md).

## Output per command

| Command | JSON output | Schema |
| --- | --- | --- |
| `projects` | array of projects | `project.schema.json` |
| `tasks [project-id]` | array of tasks | `task.schema.json` |
| `entries`, `timesheets` | array of entries | `entry.schema.json` |
| `entries add` / `update ID` | the created / updated entry | `entry.schema.json` |
| `entries delete ID` | the entry as it was before deletion | `entry.schema.json` |
| `entries add` / `update` with `--dry-run` | the entry that would be written, plus `dry_run` and `duplicates` | `entry_dry_run.schema.json` |
| `whoami` | the current user | `user.schema.json` |
| `clock status` | clock state with today's periods | `clock_status.schema.json` |
| `clock list` | array of attendance records, oldest first | `attendance.schema.json` |
| `clock in` / `clock out` | the created or closed record | `attendance.schema.json` |
| `clock add` | the created record | `attendance.schema.json` |
| `clock edit` | the record as it is now, plus `before` (`null` if created) | `attendance_edit.schema.json` |
| any command, on failure | the error envelope | `error.schema.json` |

The schemas (JSON Schema draft 2020-12) are in
[`internal/output/schemas/`](https://github.com/seletz/odoo-work-cli/tree/develop/internal/output/schemas).
They are the contract; the test suite checks every command against them.

## Objects

### project

```console
$ odoo-work-cli projects --search portal --json
--8<-- "docs/assets/cli/projects-search-json.txt"
```

`extra_fields` holds the [configured extra fields](40-configuration.md#models)
by name.

### entry

```console
$ odoo-work-cli entries --date 2026-04-09 --json
--8<-- "docs/assets/cli/entries-date-json.txt"
```

### entry_dry_run

```console
$ odoo-work-cli entries add --project-id 1 --task-id 13 --date 2026-04-09 --hours 1:30 --description "Review fuzzy matching" --dry-run --json
--8<-- "docs/assets/cli/entries-add-dry-run-json.txt"
```

### attendance

```console
$ odoo-work-cli clock list --date 2026-04-08 --json
--8<-- "docs/assets/cli/clock-list-date-json.txt"
```

`running` is `true` and `check_out` `null` while the record is open.

### Other objects

| Object | Fields |
| --- | --- |
| `task` | id, name, project_id, project, company, stage, active |
| `user` | id, name, login, email, company |
| `clock_status` | clocked_in, current_id, check_in, elapsed_minutes, periods, total_hours, total_minutes |
| `attendance_edit` | the `attendance` fields plus `before` |
