---
title: Your first booking
icon: lucide/circle-play
---
# Your first booking

In this tutorial you connect `odoo-work-cli` to your Odoo 17, check the
connection and book your first timesheet entry. You need an Odoo account
that can book timesheets and about fifteen minutes.

## 1. Install

```sh
brew tap seletz/tap
brew install odoo-work-cli
```

Other ways: [Install and update](../how-to/10-install-and-update.md).

## 2. Create an Odoo API key

Odoo 17 does not accept your login password for API access. It wants an
**API key**. In Odoo, open your avatar → *My Profile* → *Account Security* →
*New API Key* (or *Settings → Users → your user → API Keys*). Give it a name
such as `odoo-work-cli` and copy the key; Odoo shows it only once.

Why a key and not your password: [Two transports](../explanation/transports.md).

## 3. Create the config file

```sh
odoo-work-cli config install
```

This writes `~/.config/odoo-work-cli/config.toml`. Open it and set three
values:

```toml
url = "https://odoo.example.com"   # your Odoo, without a path
database = "odoo"                  # the database name
username = "you@example.com"       # your login
```

Do not put the API key in this file; it would be rejected.

## 4. Provide the API key

For this tutorial, an environment variable is enough:

```sh
export ODOO_PASSWORD="paste-the-api-key-here"
```

Yes, the variable is called `ODOO_PASSWORD`, and it holds the API key. To keep
the key in 1Password instead, see [Secrets](../explanation/secrets.md).

## 5. Check the connection

```sh
odoo-work-cli whoami
```

You see your user ID, name, login and company. An error of kind `auth` means
the key is wrong or is a password; see
[Troubleshoot](../how-to/95-troubleshoot.md).

## 6. Find the project

Search for part of the project name or the customer:

```sh
odoo-work-cli projects --search portal
odoo-work-cli tasks 42
```

Note the project ID and, if you book on a task, the task ID.

## 7. Preview the entry

```sh
odoo-work-cli entries add --project-id 42 --task-id 10 --hours 1:30 \
  --description "Code review" --dry-run
```

Hours are `1:30` or `1.5`. The date defaults to today; add `--date 2026-09-14`
for another day. The dry run prints the entry and writes nothing. It warns
when the same entry already exists.

## 8. Book it

Run the same command without `--dry-run`:

```sh
odoo-work-cli entries add --project-id 42 --task-id 10 --hours 1:30 \
  --description "Code review"
odoo-work-cli entries
```

`entries` without flags lists the current week. Your entry is there, with
status `draft`.

## Next

- Book and fix entries in the grid: [A tour of the TUI](30-tui-tour.md).
- Clock in and out: [Clock in and out](../how-to/30-clock-in-out.md). It needs
  your login password as well; [Secrets](../explanation/secrets.md) says why.
