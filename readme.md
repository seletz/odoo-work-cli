# odoo-work-cli

CLI tool for managing Odoo 17 timesheets and projects from the terminal, as well as attendance (clock-in and clock-out).

## Features

- **CLI:** CLI commands for scripting
- **TUI:** Terminal UI for fast interactive usage
- **Config bootstrapping:** `config install` creates a default config file

### TUI Features

- Remembers last week's tasks and projects you worked
- Search for projects and tasks to add new rows (`/` key), with filter toggle (`Ctrl+A`)
- German Holidays are marked and coloured
- Shows attendance state prominently
- Color coded work hour day and week summaries with configurable limits
- Configurable key bindings via `[keys]` section in config file
- Company-based color coding for project/task labels via `[company_colors]` config
- Add, edit and delete time entries
- Hours input accepts both `H:MM` (e.g. `1:30`) and decimal (e.g. `1.5`) formats
- Help overlay (`?` key) showing all key bindings grouped by context
- It's pretty fast

## Usage

### TUI

Just do:

```bash
odoo-work-cli tui
```

### CLI

Commands:

| Command                                                  | Description                                                                                                         |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `whoami`                                                 | Show current Odoo user info (ID, name, login, email, company)                                                       |
| `projects`                                               | List Odoo projects (with customer, company, phase, project manager)                                                 |
| `tasks [project-id]`                                     | List Odoo tasks, optionally filtered by project ID                                                                  |
| `timesheets [--week YYYY-Www]`                           | List timesheets for a week (defaults to current week)                                                               |
| `entries [--week\|--date] [--project\|--task\|--status]` | List individual timesheet entries with full detail (description, hours, validation status)                          |
| `entries add --project-id N --hours H --description "…"` | Create a new timesheet entry (hours: `2.5` or `2:30`, date defaults to today, task-id optional)                     |
| `entries update ID [--hours H] [--description "…"] …`    | Partially update a timesheet entry (hours: `2.5` or `2:30`, only set flags are sent)                                |
| `entries delete ID`                                      | Delete a timesheet entry by ID                                                                                      |
| `clock in`                                               | Clock in (start attendance period)                                                                                  |
| `clock out`                                              | Clock out (end attendance period, shows duration)                                                                   |
| `clock status`                                           | Show current attendance state and today's periods                                                                   |
| `tui`                                                    | Weekly timesheet TUI with detail view (auto-reloads from Odoo), inline editing/adding, and live clock-in/out status |
| `fields <model>`                                         | Inspect field metadata for any Odoo model                                                                           |
| `config`                                                 | Show discovered config file paths (merge order)                                                                     |
| `config --merged`                                        | Print the fully merged TOML config (password omitted)                                                               |
| `config install`                                         | Create a default config file at the platform config directory                                                       |

Examples:

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
./odoo-work-cli entries add --project-id 42 --hours 2:30 --description "Dev work"
./odoo-work-cli entries add --project-id 42 --task-id 10 --date 2026-03-09 --hours 1.5 --description "Code review"
./odoo-work-cli entries update 100 --hours 1:15
./odoo-work-cli entries update 100 --description "Updated description" --hours 2.5
./odoo-work-cli entries delete 100
./odoo-work-cli clock in
./odoo-work-cli clock out
./odoo-work-cli clock status
./odoo-work-cli config
./odoo-work-cli config --merged
./odoo-work-cli config install
```

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

Then run `mise run inject-env` to generate `.env`. Passwords must come from the
`ODOO_PASSWORD` env var -- config files that contain a `password` field are
rejected.

### Config file example

Run `odoo-work-cli config install` to create a default config file with all
options documented. The generated file looks like:

```toml
url = "https://odoo.example.com"
database = "odoo"
username = "user@example.com"
bundesland = "Baden-Württemberg"

# NOTE: password/API key must be set via ODOO_PASSWORD env var, not here.

[hours]
daily_low = 6.0    # below this: yellow
daily_high = 9.0   # above this: red
weekly_low = 35.0  # below this: yellow
weekly_high = 40.0 # above this: red

[company_colors]
"My Company" = "5"       # purple/magenta
"Partner Corp" = "2"     # green

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

### Company-based row colors

Project/task labels in the TUI can be colored by company name using the
`[company_colors]` section. Values are ANSI 256-color codes (as strings).
Companies not listed use the default terminal color.

```toml
[company_colors]
"digitalgedacht GmbH" = "5"    # purple/magenta
"nexiles GmbH" = "2"           # green
```

### Custom fields per model

Extra Odoo fields can be fetched per model via `extra_fields`. Each entry
specifies a display name, the Odoo field name, and its type (`many2one`, `char`,
`boolean`, `integer`, `float`, etc.). These appear as additional columns in
command output.

### Default query filters per model

Filters scope queries automatically. They are defined per model under
`[models.<name>]` with `field`, `op`, and `value`. Supported operators include
`=`, `!=`, `ilike`, `>`, `<`, `>=`, `<=`, etc.

### Configurable key bindings

TUI key bindings can be overridden in the `[keys]` section. Action names are
prefixed with the context they apply to. Only overridden keys change; others
keep their defaults. Values can be a single string or an array of strings.

```toml
[keys]
# Cursor movement (shared across grid, detail, search views)
cursor_up = ["up", "k"]
cursor_down = ["down", "j"]

# Grid view
grid_next_col = ["tab"]
grid_prev_col = ["shift+tab"]
grid_enter = ["enter"]
grid_search = ["/"]

# Detail view
detail_edit = ["e"]
detail_add = ["a"]
detail_delete = ["d"]

# Search view
search_toggle = ["ctrl+a"]

# Global (available in all non-modal views)
global_prev_week = ["left", "h"]
global_next_week = ["right", "l"]
global_back = ["esc"]
global_refresh = ["r"]
global_help = ["?"]
global_quit = ["q", "ctrl+c"]
```

Filters **accumulate** across config levels (AND semantics). If a child config
defines a filter on the same field as a parent, the child's entry overrides the
parent's. This lets you set a company-wide filter in a parent directory and add
project-specific filters in subdirectories.

## Development

### Prerequisites

- [Go](https://go.dev/) 1.21+
- [mise](https://mise.jdx.dev/) for task running and tool management
- [1Password CLI](https://developer.1password.com/docs/cli/) (`op`) for secrets injection

### Mise Tasks

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

## Releasing

Releases are managed via `mise run release` and built automatically by GitHub Actions.

### Creating a release

```bash
mise run release
```

This will:

1. Show the current version (latest `v*` tag, or `v0.0.0` if none)
2. Prompt for bump type (patch / minor / major)
3. Create an annotated git tag
4. Push the tag to origin
5. Create a GitHub prerelease with auto-generated notes

### CI build

When a release is created, GitHub Actions automatically:

- Cross-compiles for macOS (amd64, arm64), Linux (amd64, arm64), and Windows (amd64)
- Packages binaries as `.tar.gz` (Unix) or `.zip` (Windows)
- Uploads all artifacts to the release
- Generates a `checksums-<tag>.txt` with SHA-256 hashes

### Version embedding

The build injects the version via `-ldflags`:

```bash
mise run build
./odoo-work-cli --version
```
