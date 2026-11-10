---
title: Use it from a coding agent
icon: lucide/bot
---
# Use it from a coding agent

How to let a coding agent (Claude Code, Codex, …) book time and attendance
with `odoo-work-cli`. The agent runs the CLI through its shell tool, reads
JSON and branches on error kinds.

## Give the agent the guide

`odoo-work-cli llm --skill` prints a Markdown guide for agents: every command
and flag, the JSON fields, the error kinds, workflows (booking entries,
backfilling attendance) and the pitfalls. It needs no config and no
connection, and it always matches the installed binary: the field lists and
error kinds are generated from the code, and a test fails when a command or
flag is missing from it.

Save it as a skill, for example for Claude Code:

```sh
mkdir -p ~/.claude/skills/odoo-work-cli
odoo-work-cli llm --skill > ~/.claude/skills/odoo-work-cli/SKILL.md
```

The file starts with the skill front matter (`name`, `description`), so the
agent loads it when the user asks to book hours or check working time.
Regenerate it after an update.

## The rules the guide sets

- **Always `--json`**, and read stdout only.
- **Exit code** `0` on success, `1` on any error. With `--json` the error is
  one envelope on stdout: `{"error":{"kind":"…","message":"…"}}`. Branch on
  `kind`; see [Errors and exit codes](../reference/30-errors.md).
- **Look IDs up, never guess them**: `projects --search`, `tasks`, `entries`,
  `clock list`.
- **Preview writes**: `entries add … --dry-run --json`, show the proposal to
  the user, book only after approval. See [Preview a write](50-dry-run.md).
- **Never overwrite silently**: backfill attendance with
  `clock edit --create-only`; on `conflict`, ask.
- **No retry loops**: `auth`, `access_denied`, `config` and `secrets_locked`
  need the user.
- **No TUI**: `tui` is interactive.

## A typical session

```console
$ odoo-work-cli projects --search portal --json
--8<-- "docs/assets/cli/projects-search-json.txt"
$ odoo-work-cli entries --date 2026-04-09 --json
--8<-- "docs/assets/cli/entries-date-json.txt"
```

The agent sees that *Review fuzzy matching* is already booked, and its dry
run reports it as a duplicate:

```console
$ odoo-work-cli entries add --project-id 1 --task-id 13 --date 2026-04-09 --hours 1:30 --description "Review fuzzy matching" --dry-run --json
--8<-- "docs/assets/cli/entries-add-dry-run-json.txt"
```

## Let the agent experiment safely

Point it at the [fake backend](70-test-without-odoo.md#the-fake-backend):

```sh
export ODOO_WORK_CLI_BACKEND=fake
export ODOO_WORK_CLI_FAKE_STATE="$(mktemp -d)/state.json"
```

Same commands, same JSON, same error kinds, made-up data.

## 1Password

If your config uses `[op_secrets]` and 1Password is locked, every command
fails after about 5 seconds with `secrets_locked`. The agent should stop and
ask you to unlock it. To run without 1Password, see
[Test without Odoo](70-test-without-odoo.md#real-odoo-no-1password).
