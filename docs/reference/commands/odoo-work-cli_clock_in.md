# odoo-work-cli clock in

Clock in (start attendance)

### Synopsis

Clock in (start attendance).

With --json (or --format json|jsonl) the new, running attendance record is
printed.

```
odoo-work-cli clock in [flags]
```

### Options

```
      --at string   clock in at a past time (HH:MM or YYYY-MM-DD HH:MM) instead of now; requires attendance officer rights
  -h, --help        help for in
```

### Options inherited from parent commands

```
      --backend string            backend: odoo (default) or fake (offline, made-up data; env ODOO_WORK_CLI_BACKEND)
      --config string             config file path (skip discovery)
      --format table|json|jsonl   output format: table, json or jsonl (default table)
      --json                      shorthand for --format json
```

### SEE ALSO

* [odoo-work-cli clock](odoo-work-cli_clock.md)	 - Clock in/out and attendance status

