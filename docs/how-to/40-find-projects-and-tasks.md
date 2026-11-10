---
title: Find projects and tasks
icon: lucide/search
---
# Find projects and tasks

How to find the project and task IDs you book on.

## Search projects

`--search` keeps projects whose name or customer contains the text, ignoring
case. Odoo does the matching, so this is fast also on large databases:

```sh
odoo-work-cli projects --search portal
odoo-work-cli projects --search "example harbor"
odoo-work-cli projects                    # all projects
```

```console
$ odoo-work-cli projects
--8<-- "docs/assets/cli/projects.txt"
```

## List the tasks of a project

```sh
odoo-work-cli tasks 1
odoo-work-cli tasks 1 --search review     # task name contains "review"
odoo-work-cli tasks --search review       # in all projects
```

```console
$ odoo-work-cli tasks 1
--8<-- "docs/assets/cli/tasks.txt"
```

No match prints a message, or `[]` with `--json`.

## Narrow down what you see by default

Configured filters apply to every `projects` and `tasks` call, for example
only the projects of your company. See
[`[models.*]` in the configuration](../reference/40-configuration.md#models).
`--search` is combined with them (AND).

## In the TUI

Press ++slash++ in the grid; see the [TUI tour](../tutorials/30-tui-tour.md#add-a-row-with-search).
