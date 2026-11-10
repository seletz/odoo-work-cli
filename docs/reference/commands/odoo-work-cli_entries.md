# odoo-work-cli entries

List individual timesheet entries with full detail

```
odoo-work-cli entries [flags]
```

### Options

```
      --date string               specific date (YYYY-MM-DD), overrides --week
      --format table|json|jsonl   output format: table, json or jsonl (default table)
  -h, --help                      help for entries
      --json                      shorthand for --format json
      --project string            filter by project name (substring, case-insensitive)
      --status string             filter by validation status (e.g. draft, validated)
      --task string               filter by task name (substring, case-insensitive)
      --week string               ISO week (e.g. 2026-W10), defaults to current week
```

### Options inherited from parent commands

```
      --backend string   backend: odoo (default) or fake (offline, made-up data; env ODOO_WORK_CLI_BACKEND)
      --config string    config file path (skip discovery)
```

### SEE ALSO

* [odoo-work-cli](odoo-work-cli.md)	 - CLI for managing Odoo 17 timesheets
* [odoo-work-cli entries add](odoo-work-cli_entries_add.md)	 - Create a new timesheet entry
* [odoo-work-cli entries delete](odoo-work-cli_entries_delete.md)	 - Delete a timesheet entry
* [odoo-work-cli entries update](odoo-work-cli_entries_update.md)	 - Update an existing timesheet entry

