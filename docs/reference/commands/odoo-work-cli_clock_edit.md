# odoo-work-cli clock edit

Edit an attendance record's check-in/check-out time

### Synopsis

Edit the check-in and/or check-out time of an attendance record.

By default the record of the given day (--date, default today) is edited.
If the day has several records, select one with --id. If the day has no
record yet, one is created when both --in and --out are given. The old and
the new values are printed. Use 'clock list' to see the existing records first.

With --create-only the command only ever creates: if the day already has a
record it aborts with a "conflict" error and changes nothing. Use it when
backfilling days, so that an existing record is never overwritten.

Editing attendance over the API requires the 'Attendance Officer' access
right, with you set as your own attendance manager.

```
odoo-work-cli clock edit [flags]
```

### Options

```
      --create-only   only create; abort with a conflict error if the day already has a record
      --date string   day of the record to edit (YYYY-MM-DD, default today)
  -h, --help          help for edit
      --id int        attendance record ID (required if the day has several records)
      --in string     new check-in time (HH:MM or YYYY-MM-DD HH:MM)
      --out string    new check-out time (HH:MM or YYYY-MM-DD HH:MM)
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

