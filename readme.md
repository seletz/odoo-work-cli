# odoo-work-cli

CLI tool for managing Odoo 17 timesheets and projects from the terminal, as well as attendance (clock-in and clock-out).

## Features

- **CLI:** CLI commands for scripting
- **TUI:** Terminal UI for fast interactive usage
- **Clock in/out:** Works for non-admin users, including accounts with 2FA (TOTP) enabled
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
- Clock in/out toggle directly from TUI (`c` key)
- Help overlay (`?` key) showing all key bindings grouped by context
- Cursor starts on today's column when viewing the current week
- It's pretty fast

## Installation

### Homebrew (macOS / Linux)

```bash
brew tap seletz/tap
brew install odoo-work-cli
```

> [!TIP]
> If `brew upgrade` doesn't pick up a new version, untap and re-tap:
>
> ```bash
> brew untap seletz/tap
> brew tap seletz/tap
> brew install odoo-work-cli
> ```

### From GitHub Releases

Download the latest binary from
[Releases](https://github.com/seletz/odoo-work-cli/releases) and place it on
your `PATH`.

### From Source

```bash
go install github.com/seletz/odoo-work-cli/cmd/odoo-work-cli@latest
```

### Shell Completions

Homebrew installs completions automatically. For manual installs, add one of:

```bash
# zsh (add to .zshrc)
eval "$(odoo-work-cli completion zsh)"

# bash (add to .bashrc, or drop into completions dir)
odoo-work-cli completion bash > /usr/local/etc/bash_completion.d/odoo-work-cli

# fish
odoo-work-cli completion fish | source
```

## Quick Start

1. **Install** the CLI (see [Installation](#installation)).

2. **Create an Odoo API key**: in Odoo, open your user's preferences
   (*Settings → Users → your user → API Keys* tab, or avatar → *My Profile →
   Account Security → New API Key*). Odoo 17 **requires API keys** for
   XML-RPC — your login password will not work for most commands (see
   [Credentials](#credentials)).

3. **Create a config file**:

   ```bash
   odoo-work-cli config install
   ```

   Then edit the generated `config.toml` and set `url`, `database` and
   `username`.

4. **Provide the API key** — via 1Password (see
   [Secrets](#secrets-via-1password)) or as an environment variable:

   ```bash
   export ODOO_PASSWORD="your-api-key"   # yes, the API key — not your login password
   ```

5. **Verify the connection**:

   ```bash
   odoo-work-cli whoami
   ```

6. **Optional — clock in/out**: `clock in|out` additionally needs your real
   login password (`password` / `ODOO_WEB_PASSWORD`) and, if 2FA is enabled,
   your TOTP secret (`totp_secret` / `ODOO_TOTP_SECRET`). See
   [Credentials](#credentials) for why.

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
| `clock in`                                               | Clock in (start attendance period). Works for non-admin users via JSON-RPC; supports 2FA (TOTP).                    |
| `clock out`                                              | Clock out (end attendance period, shows duration). Works for non-admin users via JSON-RPC; supports 2FA (TOTP).     |
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

### Credentials

The CLI talks to Odoo over two different channels, and they need **different
credentials**. This is the most common setup stumbling block, so here is the
full picture:

| Credential         | Config key (`[op_secrets]`) | Environment variable | Needed for                                                                                                   | Where to get it                                                            |
| ------------------ | --------------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------- |
| **API key**        | `api-key`                   | `ODOO_PASSWORD`      | Everything that reads or writes models: `projects`, `tasks`, `timesheets`, `entries`, `whoami`, `fields`, `clock status`, the TUI | Odoo: *Settings → Users → your user → API Keys* (or *My Profile → Account Security*) |
| **Login password** | `password`                  | `ODOO_WEB_PASSWORD`  | `clock in` / `clock out` only (JSON-RPC web session)                                                           | Your normal Odoo login password                                             |
| **TOTP secret**    | `totp_secret`               | `ODOO_TOTP_SECRET`   | Completing the 2FA challenge during clock in/out — only if your account has 2FA enabled                        | The base32 secret (or `otpauth://` URL) shown when enabling 2FA in Odoo     |

Why three credentials?

- Odoo 17 **requires API keys for XML-RPC**. Login passwords are not
  accepted — and the failure is silent: `authenticate` just returns `false`
  with no error message.
- The attendance systray controller used by `clock in|out` cannot be reached
  via XML-RPC by regular employees, so the CLI opens a JSON-RPC **web
  session** for those two commands. Odoo's web session login is the exact
  opposite of XML-RPC: it **rejects API keys** and wants the real login
  password.
- If the account has 2FA enabled, the web session login is followed by a
  TOTP challenge. With `totp_secret` configured the CLI completes it
  automatically — no interactive prompt.

Note the naming: `ODOO_PASSWORD` holds the **API key**, and
`ODOO_WEB_PASSWORD` holds the actual password.

If you never use `clock in|out`, the API key is all you need.

### Layered config discovery

Configuration is loaded in layers, with later layers overriding earlier ones:

1. **Global config**: `$XDG_CONFIG_HOME/odoo-work-cli/config.toml` (defaults to `~/.config/odoo-work-cli/config.toml`)
2. **Directory walk**: `.odoo-work-cli.toml` files from filesystem root down to cwd (root-most first, like `.editorconfig`)
3. **`[op_secrets]`**: resolved via 1Password CLI at runtime (see below)
4. **Environment variables**: `ODOO_URL`, `ODOO_DATABASE`, `ODOO_USERNAME`, `ODOO_PASSWORD`, `ODOO_WEB_PASSWORD`, `ODOO_TOTP_SECRET` (highest priority)
5. **`--config` flag**: Skip discovery entirely, load only the specified file + op_secrets + env vars

### Secrets (via 1Password)

Add an `[op_secrets]` section to your config file with `op://` vault references.
At startup, if the `op` CLI is installed and authenticated, the CLI resolves each
reference automatically. No manual injection step needed.

```toml
[op_secrets]
url         = "op://Employee/odoo/url"
database    = "op://Employee/odoo/database"
username    = "op://Employee/odoo/username"
api-key     = "op://Employee/odoo/api-key"        # Odoo API key (for XML-RPC)
password    = "op://Employee/odoo/password"        # Odoo login password (for clock in/out)
totp_secret = "op://Employee/odoo/totp_secret"     # TOTP secret (for 2FA, if enabled)
```

See [Credentials](#credentials) for what each key is and when it is needed.

Values without the `op://` prefix are used as-is (useful for non-secret fields
like database name). If `op` is not installed or the `[op_secrets]` section is
absent, the section is skipped and the CLI falls back to environment variables.
If `op` **is** installed but not signed in, resolution fails with an error.

Plain-text passwords in config files are rejected at load time — a `password`
key outside `[op_secrets]` is an error. Secrets must come from `[op_secrets]`
or environment variables.

### Environment variables

Every connection setting can also be provided as an environment variable.
Env vars have the **highest priority** and override both config file values
and resolved `[op_secrets]`:

| Variable            | Meaning                                        |
| ------------------- | ---------------------------------------------- |
| `ODOO_URL`          | Server URL, e.g. `https://odoo.example.com`    |
| `ODOO_DATABASE`     | Database name                                  |
| `ODOO_USERNAME`     | Login (usually your email address)             |
| `ODOO_PASSWORD`     | **API key** — XML-RPC auth                     |
| `ODOO_WEB_PASSWORD` | Login password — web session (clock in/out)    |
| `ODOO_TOTP_SECRET`  | Base32 TOTP secret or `otpauth://` URL (2FA)   |

### Config file example

Run `odoo-work-cli config install` to create a default config file with all
options documented. The generated file looks like:

```toml
url = "https://odoo.example.com"
database = "odoo"
username = "user@example.com"
bundesland = "Baden-Württemberg"

# NOTE: password/API key must NOT be stored as plain text here.
# Use [op_secrets] below for 1Password, or set ODOO_PASSWORD env var.

# [op_secrets]
# url         = "op://vault/item/url"
# database    = "op://vault/item/database"
# username    = "op://vault/item/username"
# api-key     = "op://vault/item/api-key"        # Odoo API key (for XML-RPC)
# password    = "op://vault/item/password"        # Odoo login password (for clock in/out)
# totp_secret = "op://vault/item/totp_secret"     # TOTP secret (if 2FA enabled)

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

Note: the TUI weekly grid ignores `[models.timesheet]` filters and always
shows all of your own bookings. Timesheet entries booked on another
company's project carry that company's `company_id`, so a company filter
would silently hide hours you just booked (see issue #58). The `entries`
and `timesheets` commands still apply `[models.timesheet]` filters.

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
global_clock_toggle = ["c"]
global_refresh = ["r"]
global_help = ["?"]
global_quit = ["q", "ctrl+c"]
```

Filters **accumulate** across config levels (AND semantics). If a child config
defines a filter on the same field as a parent, the child's entry overrides the
parent's. This lets you set a company-wide filter in a parent directory and add
project-specific filters in subdirectories.

## Troubleshooting

**`whoami` / `projects` / `entries` fail with an authentication error even
though the credentials look right.**
XML-RPC needs an **API key**, not your login password (see
[Credentials](#credentials)). This failure is silent on the Odoo side:
password auth against XML-RPC — especially with 2FA enabled — simply returns
`false` without an error message. Create an API key and put it in `api-key` /
`ODOO_PASSWORD`.

**Authentication worked before and suddenly fails (dev environment).**
If the dev database was recreated, all API keys stored in it are gone and the
key in 1Password / `.env` is stale. Run `mise run odoo:prepare-db` (or
`mise run odoo:create-api-key` followed by `mise run prepare_env`).

**`clock in` fails with "authentication failed".**
The web session needs your real login **password** (`password` /
`ODOO_WEB_PASSWORD`) — Odoo's `/web/session/authenticate` rejects API keys.

**`clock in` fails with "2FA is enabled but no TOTP secret configured".**
Set `totp_secret` in `[op_secrets]` or `ODOO_TOTP_SECRET`. The value is the
base32 secret (or the full `otpauth://` URL) shown when enabling 2FA in Odoo.

**"TOTP verification failed".**
The `totp_secret` is wrong, or your machine's clock is skewed (TOTP codes are
time-based).

**`clock in|out` fails against a multi-database server.**
Known limitation — the web session is not reliably bound to the requested
database on multi-db servers. Tracked in
[#48](https://github.com/seletz/odoo-work-cli/issues/48). `clock status` and
all other commands are unaffected (they use XML-RPC).

**`op inject` errors at startup.**
The config has an `[op_secrets]` section and the 1Password CLI is installed
but not signed in. Run `op signin`, or remove the section and use environment
variables instead.

## Development

### Prerequisites

- [Go](https://go.dev/) 1.21+
- [mise](https://mise.jdx.dev/) for task running and tool management
- [1Password CLI](https://developer.1password.com/docs/cli/) (`op`) — optional, for `[op_secrets]` resolution

### Mise Tasks

```bash
# Install tools via mise
mise install

# Build
mise run build

# Run tests
mise run test

# Lint
mise run lint
```

### Dev environment setup

> [!IMPORTANT]
> Local development and automated tests target the **dev** Odoo environment
> **only**. The test and prod environments must never be touched by local
> development or tests.

Dev credentials live in a single 1Password item and flow into the project in
two ways:

- **`.env` for mise tasks** — [`1p.env`](1p.env) is a 1Password template
  containing `op://` references. `mise run prepare_env` runs `op inject` on it
  to generate a local `.env`, which mise auto-loads for every task (including
  `mise run run -- ...` and the `odoo:*` tasks below).
- **`[op_secrets]` for the CLI** — the checked-in
  [`.odoo-work-cli.toml`](.odoo-work-cli.toml) resolves the same 1Password
  item at runtime, so running the built binary from the repo directory picks
  up dev credentials automatically.

Both expect a 1Password item with the fields `url`, `database-name`, `login`,
`api-key`, `password`, and `totp_secret`.

Bootstrap tasks:

```bash
mise run prepare_env          # generate .env from 1p.env via op inject
mise run odoo:create-api-key  # create an XML-RPC API key via the web login
                              # flow (handles 2FA) and write it back to 1Password
mise run odoo:add-test-data   # idempotent: seed the dev db with the modules,
                              # company, project, tasks and employee the
                              # checked-in config filters expect
mise run odoo:ensure-test-user # idempotent: provision the non-admin test
                              # user (see below)
mise run odoo:prepare-db      # all of the above in order — bootstraps a
                              # freshly recreated dev db in one command
```

When the dev database is recreated ("nuked"), all previously issued API keys
are gone. `mise run odoo:prepare-db` fixes that end-to-end: it creates a new
API key, refreshes `.env`, and re-seeds the test data.

### Non-admin test user

The CLI user provisioned above is an **administrator**, so ACL/permission
bugs never surface with it — [#40](https://github.com/seletz/odoo-work-cli/issues/40)
(`whoami` faulting for regular users) is exactly the class of bug that hides.
The bootstrap therefore also provisions a permanent **non-admin** test user
(issue [#55](https://github.com/seletz/odoo-work-cli/issues/55)):

- `mise run odoo:ensure-test-user` — idempotent; creates/repairs a user with
  login `test-user`, the regular-employee groups only (**Internal User**,
  **Project/User**, **Timesheets/User: own timesheets only** — verified not
  to be in Administration/Settings), and a linked `hr.employee` record.
  Strictly Internal-User-only does not work: Odoo denies reading
  `account.analytic.line` / `project.project` without the per-app user
  groups every real employee has. A fresh random
  password is set on every run and written to the `test-user-password` field
  of the "ODOO Work CLI" 1Password item (printed once to stdout instead if
  the `op` CLI is unavailable). The user has no 2FA, so XML-RPC accepts the
  plain password — no API key needed.
- `mise run odoo:run-as-user -- <args...>` — run any CLI/TUI command as that
  user, e.g. `mise run odoo:run-as-user -- whoami`. Overrides
  `ODOO_USERNAME`/`ODOO_PASSWORD`/`ODOO_WEB_PASSWORD` for the invocation
  (password read from 1Password, or from `ODOO_TEST_USER_PASSWORD` if set).
- `mise run odoo:smoke-test-user` — smoke check: runs `whoami`,
  `clock status` and `entries` as the non-admin user and fails loudly on
  ACL faults. Run it after changes that touch Odoo model reads/writes.

### Pre-upgrade check: web-login version-drift canary

The 2FA web session used by `clock in|out` authenticates through Odoo's HTML
login flow (`/web/login` → `/web/login/totp`) and therefore encodes
web-controller internals that a new Odoo version may change
([#57](https://github.com/seletz/odoo-work-cli/issues/57)):

- `GET /web/login?db=<db>` binds the session and serves a `csrf_token` input
- `POST /web/login` accepts `csrf_token`, `login`, `password`, `redirect`
- 303 redirect = accepted (to `/web/login/totp` when 2FA is pending),
  200 = rejected
- `POST /web/login/totp` accepts `csrf_token`, `totp_token`

```bash
mise run odoo:login-canary
```

exercises the full flow against the **dev** instance — login, TOTP challenge,
and a session-authenticated JSON-RPC call — and on failure names the encoded
assumption that broke (e.g. "login page markup may have changed; check the
Odoo version").

**This is the mandatory pre-upgrade check.** Dev receives new Odoo versions
first, so run the canary against dev **before any Odoo upgrade reaches test
or prod**. If it fails, `clock in|out` (and `mise run odoo:create-api-key`,
which mirrors the same flow) will break on that Odoo version.

**Known limitation:** `clock in|out` currently fails against multi-database
servers — the web session is not reliably bound to the requested database.
Tracked in [#48](https://github.com/seletz/odoo-work-cli/issues/48).

## Releasing

Releases are managed via `mise run release` and built by
[GoReleaser](https://goreleaser.com/) in GitHub Actions.

### Creating a release

```bash
mise run release
```

This will:

1. Show the current version (latest `v*` tag, or `v0.0.0` if none)
2. Prompt for bump type (patch / minor / major)
3. Create an annotated git tag
4. Push the tag to origin

GoReleaser then automatically:

- Cross-compiles for macOS (amd64, arm64), Linux (amd64, arm64), and Windows (amd64)
- Packages binaries as `.tar.gz` (Unix) or `.zip` (Windows)
- Creates a GitHub Release with all artifacts and SHA-256 checksums
- Updates the [Homebrew tap](https://github.com/seletz/homebrew-tap) cask so `brew upgrade` picks up the new version

### Local testing

```bash
mise run goreleaser-check       # validate .goreleaser.yml
mise run goreleaser-snapshot    # full build without publishing
```

### Version embedding

The build injects the version via `-ldflags`:

```bash
mise run build
./odoo-work-cli --version
```
