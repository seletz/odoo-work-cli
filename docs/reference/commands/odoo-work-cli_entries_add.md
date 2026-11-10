# odoo-work-cli entries add

Create a new timesheet entry

### Synopsis

Create a new timesheet entry.

With --json (or --format json|jsonl) the created entry is printed.

With --dry-run the input is validated and project and task are resolved, then
the entry that would be created is printed and nothing is written. With --json
it is the same object as the real call, with "id" null and "dry_run": true,
plus "duplicates" (entries of the same day with the same project, task and
description).

```
odoo-work-cli entries add [flags]
```

### Options

```
      --date string          entry date YYYY-MM-DD (defaults to today)
      --description string   work description (required)
      --dry-run              validate and print the entry that would be written, without writing it
  -h, --help                 help for add
      --hours string         hours worked (e.g. 2.5 or 2:30)
      --project-id int       Odoo project ID (required)
      --task-id int          Odoo task ID (optional)
```

### Options inherited from parent commands

```
      --backend string            backend: odoo (default) or fake (offline, made-up data; env ODOO_WORK_CLI_BACKEND)
      --config string             config file path (skip discovery)
      --format table|json|jsonl   output format: table, json or jsonl (default table)
      --json                      shorthand for --format json
```

### SEE ALSO

* [odoo-work-cli entries](odoo-work-cli_entries.md)	 - List individual timesheet entries with full detail

