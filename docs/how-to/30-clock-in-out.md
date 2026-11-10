---
title: Clock in and out
icon: lucide/timer
---
# Clock in and out

How to record attendance: now, backdated, fixed afterwards, or added for a
day you forgot. Background: [The attendance model](../explanation/attendance-model.md).

## Clock in and out now

```sh
odoo-work-cli clock in
odoo-work-cli clock out
odoo-work-cli clock status
```

These use Odoo's own attendance button (the systray), which every employee
may use. They need your **login password** and, with 2FA, your TOTP secret
in addition to the API key; see [Secrets](../explanation/secrets.md). In the
TUI, press ++c++.

## See what is recorded

```sh
odoo-work-cli clock list                    # today
odoo-work-cli clock list --date 2026-04-08
odoo-work-cli clock list --week 2026-W14
```

```console
$ odoo-work-cli clock list --week 2026-W14
--8<-- "docs/assets/cli/clock-list-week.txt"
```

A record counts on the day of its check-in. A shift from 22:00 to 01:30 shows
its check-out with a date.

## Clock in or out at another time

```sh
odoo-work-cli clock in --at 08:30                 # today, 08:30
odoo-work-cli clock out --at 17:15
odoo-work-cli clock in --at "2026-09-14 08:30"    # explicit date
```

Times are local. Future times are rejected (with one minute of grace for
clock skew).

## Fix a record

```sh
odoo-work-cli clock edit --in 08:00                       # today's record
odoo-work-cli clock edit --in 08:00 --out 12:30
odoo-work-cli clock edit --date 2026-09-14 --in 08:15
odoo-work-cli clock edit --id 4606 --out 12:15            # a day with several records
```

`clock edit` prints the old and the new values:

```
Updated attendance #4667
  was: 2026-09-14 08:00 - 12:00 (4:00)
  now: 2026-09-14 08:15 - 12:00 (3:45)
```

Which record it changes:

- **One record** on that day: that one.
- **No record**: it creates one, if you give both `--in` and `--out`.
- **Several records**: it lists them and fails; repeat with `--id N`.

## Fill in a forgotten day

Use `--create-only`. It never overwrites; if the day has a record already,
it changes nothing and fails with `conflict`:

```sh
odoo-work-cli clock list --week 2026-W38
odoo-work-cli clock edit --date 2026-09-14 --in 08:00 --out 16:30 --create-only
```

Do this day by day when you backfill a week.

## Record a day with a break

`clock add` always creates a new record, also on a day that has records:

```sh
odoo-work-cli clock add --date 2026-09-14 --in 08:00 --out 12:00
odoo-work-cli clock add --date 2026-09-14 --in 12:30 --out 17:00
```

It refuses spans that overlap an existing record and changes nothing:

```console
$ odoo-work-cli clock add --date 2026-04-08 --in 11:00 --out 13:00
--8<-- "docs/assets/cli/clock-add-conflict.txt"
```

Records that only touch (12:00 out, 12:00 in) do not overlap.

| Goal | Command |
| --- | --- |
| Add a record and keep what is there | `clock add` |
| Fill a day that must be empty | `clock edit --create-only` |
| Correct an existing record | `clock edit` (`--id N` on a day with several) |

## Permissions

`--at`, `clock edit` and `clock add` write `hr.attendance` over the API. Odoo
17 allows that only with the right **Officer: Manage attendances**, and only
for employees whose attendance manager you are. Without it you get the error
kind `access_denied` and this hint:

```
hint: editing attendance over the API requires the 'Attendance Officer' access right with you set as your own attendance manager; ask your administrator, or use plain 'clock in'/'clock out'
```

The officer group is hidden; Odoo grants it through the employee record. An
administrator opens *Employees → your employee → Work Information →
Approvers → Attendance* and selects **your own user**. Saving that field adds
the group, clearing it removes it again.

Plain `clock in` and `clock out` keep working without these rights.
