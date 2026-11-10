---
title: Book time
icon: lucide/notebook-pen
---
# Book time

How to add, change and delete timesheet entries on the command line. In the
TUI the same works with ++a++, ++e++ and ++d++ in the detail view; see the
[TUI tour](../tutorials/30-tui-tour.md).

## Add an entry

```sh
odoo-work-cli entries add --project-id 42 --task-id 10 --hours 2:30 \
  --description "Code review"
```

- `--project-id`, `--hours` and `--description` are required.
- `--task-id` is optional. The task must belong to the project.
- `--date YYYY-MM-DD` books on another day; the default is today.
- Hours are `H:MM` (`2:30`) or decimal (`2.5`).

Don't know the IDs? [Find projects and tasks](40-find-projects-and-tasks.md).
Not sure? Add `--dry-run` first: [Preview a write](50-dry-run.md).

## See what is booked

```sh
odoo-work-cli entries                           # current week
odoo-work-cli entries --week 2026-W15
odoo-work-cli entries --date 2026-04-08
odoo-work-cli entries --project lighthouse --status draft
odoo-work-cli timesheets --week 2026-W15        # weekly summary with total
```

```console
$ odoo-work-cli timesheets --week 2026-W15
--8<-- "docs/assets/cli/timesheets.txt"
```

`--project` and `--task` match a part of the name, ignoring case. `--status`
is Odoo's validation status (`draft`, `validated`).

## Change an entry

Only the flags you pass are sent:

```sh
odoo-work-cli entries update 4019 --hours 1:15
odoo-work-cli entries update 4019 --description "Customer call (pricing)" --date 2026-04-08
```

`update` also takes `--project-id` and `--task-id`. Preview with `--dry-run`:
it shows the entry with your changes merged in.

## Delete an entry

```sh
odoo-work-cli entries delete 4019
```

A wrong ID fails with `not_found` before anything is deleted. With `--json`
the output is the entry as it was, so you can book it again if needed.

## Book many entries from a script

Use `--json` and stop on the first error; see
[Script with JSON output](60-json-output.md).
