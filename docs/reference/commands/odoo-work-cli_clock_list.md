# odoo-work-cli clock list

List attendance records of a day or week

### Synopsis

List the attendance records of a day (--date, default today) or an ISO
week (--week YYYY-Www) with ID, check-in, check-out and duration.

Records are attributed to the day of their check-in. A record that crosses
midnight is listed on its check-in day and shows the check-out date. Use it
before 'clock edit' to see what is already recorded.

```
odoo-work-cli clock list [flags]
```

### Options

```
      --date string   day to list (YYYY-MM-DD, default today)
  -h, --help          help for list
      --week string   ISO week to list (YYYY-Www)
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

