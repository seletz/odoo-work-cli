# odoo-work-cli clock out

Clock out (end attendance)

### Synopsis

Clock out (end attendance).

With --json (or --format json|jsonl) the closed attendance record is
printed.

```
odoo-work-cli clock out [flags]
```

### Options

```
      --at string   clock out at a past time (HH:MM or YYYY-MM-DD HH:MM) instead of now; requires attendance officer rights
  -h, --help        help for out
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

