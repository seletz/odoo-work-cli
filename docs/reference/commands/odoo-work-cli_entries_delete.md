# odoo-work-cli entries delete

Delete a timesheet entry

### Synopsis

Delete a timesheet entry.

With --json (or --format json|jsonl) the entry as it was before deletion
is printed.

```
odoo-work-cli entries delete ID [flags]
```

### Options

```
  -h, --help   help for delete
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

