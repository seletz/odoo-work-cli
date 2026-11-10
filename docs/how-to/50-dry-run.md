---
title: Preview a write with --dry-run
icon: lucide/eye
---
# Preview a write with `--dry-run`

`entries add` and `entries update` accept `--dry-run`. It runs the same
checks as the real call and prints the entry that would be written. Nothing
is written.

```console
$ odoo-work-cli entries add --project-id 1 --task-id 13 --date 2026-04-09 --hours 1:30 --description "Review fuzzy matching" --dry-run
--8<-- "docs/assets/cli/entries-add-dry-run.txt"
```

What it checks:

- the flags and values (hours, date);
- that project and task exist (an unknown ID gives the same error kind as
  the real call) and that the task belongs to the project;
- for `update`: that the entry exists. The output is the entry with your
  changes merged in.

It also looks for **duplicates**: entries on the same day with the same
project, task and description. The table output warns per duplicate.

## As JSON

```console
$ odoo-work-cli entries add --project-id 1 --task-id 13 --date 2026-04-09 --hours 1:30 --description "Review fuzzy matching" --dry-run --json
--8<-- "docs/assets/cli/entries-add-dry-run-json.txt"
```

The object is an `entry` plus `"dry_run": true` and `duplicates` (`[]` if
none). `id` is `null` for `add`. `employee` and `status` are only known once
Odoo stored the entry, so they are `null` for `add`. Schema:
`entry_dry_run.schema.json`, see [JSON output](../reference/20-json-output.md).

## Then book

Run the same command without `--dry-run`. Agents should show the dry run to
the user and wait for approval; see [Use it from a coding agent](80-agents.md).
