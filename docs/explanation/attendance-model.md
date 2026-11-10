---
title: The attendance model
icon: lucide/calendar-clock
weight: 40
---
# The attendance model

Odoo records attendance as `hr.attendance` records: one check-in, one
check-out, and the hours in between. This page explains how `odoo-work-cli`
maps them onto days.

## Records, not days

A record has `check_in`, `check_out` (empty while you are clocked in) and
`worked_hours`. A day can have none, one or several records: a day with a
lunch break taken off the clock is two records.

That is why there are three ways to write:

- `clock edit` changes *the* record of a day. With several records it needs
  `--id`.
- `clock edit --create-only` fills a day that must be empty and refuses a day
  that has a record. Safe for backfilling.
- `clock add` adds a record next to the existing ones and refuses overlaps.

## A record belongs to the day of its check-in

A shift from 22:00 to 01:30 counts on the first day. `clock list` shows its
check-out with a date, the TUI with `HH:MM+`. The same rule decides what
`--create-only` considers "the day already has a record". To find a record
that runs past midnight, the CLI asks for records that checked in today and
for records that checked in earlier and are still open.

## Local time in, UTC in the database

Odoo stores datetimes in UTC. You type local times (`08:00`), the CLI
converts them (08:00 CEST is stored as 06:00), and every output converts
back. What you type is what you see. JSON timestamps carry the local offset
(`2026-04-08T08:30:00+02:00`).

## worked_hours excludes lunch

Odoo computes `worked_hours` from the working calendar and subtracts the
lunch break. A record from 08:00 to 16:30 can have 8:00 `worked_hours`.
`odoo-work-cli` shows Odoo's value, so the TUI's daily difference between
attendance and booked hours is `0:00` on a fully booked day. While a record
is running there is no `worked_hours` yet; the CLI counts the time elapsed
so far, without lunch deduction.

The fake backend does not know working calendars; its `worked_hours` is the
plain difference. The demo data in these docs uses it.

## Future times are rejected

A check-in or check-out in the future is an error (with one minute of grace
for clock skew). Attendance records what happened; to clock in now, use
`clock in` without `--at`.
