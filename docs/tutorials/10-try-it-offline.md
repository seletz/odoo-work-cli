---
title: Try it offline
icon: lucide/flask-conical
---
# Try it offline

In this tutorial you run `odoo-work-cli` without an Odoo server. You list
projects, look at a week, preview an entry, book it and open the TUI. All of
it runs against the built-in fake backend with made-up data. You need about
ten minutes and nothing but the binary.

## 1. Install the binary

```sh
brew tap seletz/tap
brew install odoo-work-cli
```

No Homebrew? See [Install and update](../how-to/10-install-and-update.md) for
release binaries and `go install`.

## 2. Load the demo data

Download the demo state, [`demo-state.json`](../assets/demo-state.json), and
point the CLI at it. The second variable switches to the fake backend:

```sh
curl -sSLo /tmp/demo-state.json https://seletz.github.io/odoo-work-cli/assets/demo-state.json
export ODOO_WORK_CLI_FAKE_STATE=/tmp/demo-state.json
export ODOO_WORK_CLI_BACKEND=fake
```

The fake backend reads no config file and never calls 1Password. Writes go
into the state file, so what you book stays there until you download it
again.

!!! note "Without the download"
    Skip the `curl` line and the fake backend creates its own, smaller data
    set on the first call. The outputs then differ from the ones below.

## 3. Who are you?

```console
$ odoo-work-cli whoami
--8<-- "docs/assets/cli/whoami.txt"
```

## 4. Find something to book on

Projects, then the tasks of project 1:

```console
$ odoo-work-cli projects
--8<-- "docs/assets/cli/projects.txt"
$ odoo-work-cli tasks 1
--8<-- "docs/assets/cli/tasks.txt"
```

## 5. Look at a day

The demo world is frozen in ISO week 2026-W15. Thursday, 9 April, is "today".
Look at Wednesday:

```console
$ odoo-work-cli entries --date 2026-04-08
--8<-- "docs/assets/cli/entries-date.txt"
```

## 6. Preview an entry

Before you write anything, ask for a dry run. It validates the input, resolves
project and task, and writes nothing:

```console
$ odoo-work-cli entries add --project-id 1 --task-id 13 --date 2026-04-09 --hours 1:30 --description "Review fuzzy matching" --dry-run
--8<-- "docs/assets/cli/entries-add-dry-run.txt"
```

The warning tells you that the same entry already exists. Good thing you
looked.

## 7. Book an entry

Book something new instead, on the support task of project 2:

```console
$ odoo-work-cli entries add --project-id 2 --task-id 22 --date 2026-04-09 --hours 0:45 --description "Answer support mail"
--8<-- "docs/assets/cli/entries-add.txt"
```

Check it with `odoo-work-cli entries --date 2026-04-09`. The new entry is
listed with ID 5000.

## 8. The same as JSON

Scripts and agents ask for JSON:

```console
$ odoo-work-cli projects --search portal --json
--8<-- "docs/assets/cli/projects-search-json.txt"
```

## 9. Open the TUI

```sh
odoo-work-cli tui --week 2026-W15
```

You see the week from the screenshot on the [home page](../index.md), with
your new entry in the row *Orchard Mobile App / Support*. Press ++question++
for help and ++q++ to quit. The [TUI tour](30-tui-tour.md) shows the rest.

## Clean up

```sh
unset ODOO_WORK_CLI_BACKEND ODOO_WORK_CLI_FAKE_STATE
rm /tmp/demo-state.json
```

## Next

- Connect to your own Odoo: [Your first booking](20-first-booking.md).
- Everything about the fake backend: [Test without Odoo](../how-to/70-test-without-odoo.md).
