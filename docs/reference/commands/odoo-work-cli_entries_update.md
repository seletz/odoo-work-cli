# odoo-work-cli entries update

Update an existing timesheet entry

### Synopsis

Update an existing timesheet entry.

With --json (or --format json|jsonl) the updated entry is printed.

With --dry-run the entry is read, the flags are merged into it and validated
(project and task are resolved), and the resulting entry is printed. Nothing
is written. With --json it is the same object as the real call plus
"dry_run": true and "duplicates".

```
odoo-work-cli entries update ID [flags]
```

### Options

```
      --date string          entry date YYYY-MM-DD
      --description string   work description
      --dry-run              validate and print the entry that would be written, without writing it
  -h, --help                 help for update
      --hours string         hours worked (e.g. 2.5 or 2:30)
      --project-id int       Odoo project ID
      --task-id int          Odoo task ID
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

