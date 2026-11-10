# odoo-work-cli clock add

Add an attendance record (e.g. the second half of a day with a break)

### Synopsis

Add a new attendance record with check-in --in and check-out --out on the
given day (--date, default today).

Unlike 'clock edit', it always creates a new record, also on days that
already have records, so a day with a break is two 'clock add' calls.
If the new span overlaps an existing record (including a record from the
day before that runs past midnight, or a running record) the command
aborts with a "conflict" error, names the overlapping record and changes
nothing. Adjacent records (12:00 out, 12:00 in) do not overlap.

Adding attendance over the API requires the 'Attendance Officer' access
right, with you set as your own attendance manager.

```
odoo-work-cli clock add [flags]
```

### Options

```
      --date string   day of the new record (YYYY-MM-DD, default today)
  -h, --help          help for add
      --in string     check-in time (HH:MM or YYYY-MM-DD HH:MM)
      --out string    check-out time (HH:MM or YYYY-MM-DD HH:MM)
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

