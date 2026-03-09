# odoo-work-cli

CLI tool for managing Odoo 17 timesheets and projects from the terminal.

## Prerequisites

- [Go](https://go.dev/) 1.21+
- [mise](https://mise.jdx.dev/) for task running and tool management
- [1Password CLI](https://developer.1password.com/docs/cli/) (`op`) for secrets injection

## Setup

```bash
# Install tools via mise
mise install

# Inject secrets from 1Password (edit .env.1p with your vault/item references first)
mise run inject-env

# Build
mise run build

# Run tests
mise run test

# Lint
mise run lint
```

## Commands

| Command                                                  | Description                                                                                |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| `whoami`                                                 | Show current Odoo user info (ID, name, login, email, company)                              |
| `projects`                                               | List Odoo projects (with customer, company, phase, project manager)                        |
| `tasks [project-id]`                                     | List Odoo tasks, optionally filtered by project ID                                         |
| `timesheets [--week YYYY-Www]`                           | List timesheets for a week (defaults to current week)                                      |
| `entries [--week\|--date] [--project\|--task\|--status]` | List individual timesheet entries with full detail (description, hours, validation status) |
| `entries add --project-id N --hours H --description "…"` | Create a new timesheet entry (date defaults to today, task-id optional)                    |
| `entries update ID [--hours H] [--description "…"] …`    | Partially update a timesheet entry (only set flags are sent)                               |
| `entries delete ID`                                      | Delete a timesheet entry by ID                                                             |
| `fields <model>`                                         | Inspect field metadata for any Odoo model                                                  |
| `config`                                                 | Show discovered config file paths (merge order)                                            |
| `config --merged`                                        | Print the fully merged TOML config (password omitted)                                      |

## Configuration

### Layered config discovery

Configuration is loaded in layers, with later layers overriding earlier ones:

1. **Global config**: `$XDG_CONFIG_HOME/odoo-work-cli/config.toml` (defaults to `~/.config/odoo-work-cli/config.toml`)
2. **Directory walk**: `.odoo-work-cli.toml` files from filesystem root down to cwd (root-most first, like `.editorconfig`)
3. **Environment variables**: `ODOO_URL`, `ODOO_DATABASE`, `ODOO_USERNAME`, `ODOO_PASSWORD` (highest priority)
4. **`--config` flag**: Skip discovery entirely, load only the specified file + env vars

### Secrets (via 1Password)

Copy `.env.1p` and fill in your 1Password references:

```
ODOO_URL={{ op://your-vault/odoo/url }}
ODOO_DATABASE={{ op://your-vault/odoo/database }}
ODOO_USERNAME={{ op://your-vault/odoo/username }}
ODOO_PASSWORD={{ op://your-vault/odoo/password }}
```

Then run `mise run inject-env` to generate `.env`. Passwords must come from the `ODOO_PASSWORD` env var -- config files that contain a `password` field are rejected.

### Config file example

```toml
url = "https://odoo.example.com"
database = "mydb"
username = "admin"

[models.project]
extra_fields = [
  { name = "product_owner", field = "x_studio_productowner", type = "many2one" },
]
filters = [
  { field = "company_id.name", op = "=", value = "My Company" },
]

[models.task]
filters = [
  { field = "project_id.name", op = "=", value = "My Project" },
  { field = "stage_id.name", op = "=", value = "In Progress" },
]
```

### Custom fields per model

Extra Odoo fields can be fetched per model via `extra_fields`. Each entry specifies a display name, the Odoo field name, and its type (`many2one`, `char`, `boolean`, `integer`, `float`, etc.). These appear as additional columns in command output.

### Default query filters per model

Filters scope queries automatically. They are defined per model under `[models.<name>]` with `field`, `op`, and `value`. Supported operators include `=`, `!=`, `ilike`, `>`, `<`, `>=`, `<=`, etc.

Filters **accumulate** across config levels (AND semantics). If a child config defines a filter on the same field as a parent, the child's entry overrides the parent's. This lets you set a company-wide filter in a parent directory and add project-specific filters in subdirectories.

## Usage

```bash
./odoo-work-cli whoami
./odoo-work-cli projects
./odoo-work-cli tasks
./odoo-work-cli tasks 42
./odoo-work-cli timesheets
./odoo-work-cli timesheets --week 2026-W10
./odoo-work-cli fields project.project
./odoo-work-cli entries
./odoo-work-cli entries --week 2026-W10
./odoo-work-cli entries --date 2026-03-02
./odoo-work-cli entries --project "Acme" --status draft
./odoo-work-cli entries add --project-id 42 --hours 2.5 --description "Dev work"
./odoo-work-cli entries add --project-id 42 --task-id 10 --date 2026-03-09 --hours 1.5 --description "Code review"
./odoo-work-cli entries update 100 --hours 3.0
./odoo-work-cli entries update 100 --description "Updated description" --hours 2.5
./odoo-work-cli entries delete 100
./odoo-work-cli config
./odoo-work-cli config --merged
```

## Development

```bash
mise run fmt     # format code
mise run lint    # run linters (golangci-lint v2)
mise run test    # run tests
mise run build   # compile binary
```
