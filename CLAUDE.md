# odoo-work-cli

CLI and TUI tool for managing Odoo 17 timesheets and attendance, written in Go.

## Session Start — Read These First

Before touching any code, read in this order:

1. **Decisions log** — `~/develop/notes/00 Persönlich/00.03 Projekte/odoo-work-cli/decisions.md`
   What was decided, why, what was rejected, current status, open questions.
   Read this to avoid re-litigating settled decisions.

2. **Design doc** — `~/develop/notes/00 Persönlich/00.03 Projekte/odoo-work-cli/odoo-work-cli.md`
   Full spec, tech stack, Odoo model reference.

3. **Progress log** — `~/develop/notes/00 Persönlich/00.03 Projekte/odoo-work-cli/odoo-work-cli progress.md`
   Completed tasks, lessons learned, open tasks per phase.

## Session End — Update decisions.md

When a session produces a significant decision (architectural choice, API constraint discovered,
alternative explicitly rejected, open question resolved), append it to `decisions.md`
in the format already used there: decision → rationale → rejected alternatives → status.
Do not summarise. Record the actual decision.

---

**IMPORTANT:** Always use TDD. Red → Green → Refactor. No exceptions.

---

## Odoo Environments

There are now full **dev**, **test**, and **prod** environments of odoo-erp.

- Development and testing **MUST ONLY** target the **dev** environment.
- **test and prod MUST NOT be touched** by local development or automated tests.

## Project Structure

```
cmd/odoo-work-cli/main.go    # Cobra CLI entrypoint (all commands)
internal/app/                 # Runtime dependency container (Deps: config + client)
internal/config/              # Config loading: TOML discovery, env vars, op secrets, install
internal/odoo/                # Odoo client interface + XML-RPC and JSON-RPC implementations
internal/filter/              # Entry filtering (project/task/status)
internal/parsing/             # Date and hours parsing helpers
internal/display/             # Output formatting
internal/tui/                 # Bubbletea TUI: model, grid, rendering, keymap, styles, holidays
internal/version/             # Version string injected at build time via ldflags
scripts/release.sh            # Interactive semver release script
```

## Commands

- `projects`, `tasks [project-id]` — list projects/tasks (with configurable extra fields and filters)
- `timesheets --week` — weekly summary
- `entries` (+ `add`, `update`, `delete`) — timesheet entries with `--week/--date/--project/--task/--status` filters
- `clock in|out|status` — attendance clock in/out and daily status
- `tui --week` — interactive weekly grid view
- `whoami`, `fields <model>`, `config [--merged]`, `config install`

## Architecture

- **Interface-driven**: `odoo.Client` interface in `internal/odoo/client.go`; implementation is `XMLRPCClient` in `xmlrpc.go` using `github.com/skilld-labs/go-odoo`
- **go-odoo**: use low-level API only (`ExecuteKw`, `SearchRead`, `Create`) — do NOT use generated model wrappers for timesheets (missing custom fields, missing `validated_status`)
- **Dual transport**: XML-RPC (API key auth) for models CRUD; a JSON-RPC web session (`internal/odoo/jsonrpc.go`) for controller endpoints like the attendance systray toggle. The web session authenticates with the login **password** (`WebPassword`) and completes TOTP 2FA challenges automatically when `TOTPSecret` is set (via `pquerna/otp`)
- **Attendance**: `internal/odoo/attendance.go` — clock in/out goes through `/hr_attendance/systray_check_in_out` (sudo'd controller, works for non-admin users); status is read via `hr.attendance` search
- **Deps container**: `internal/app.Deps` holds config + client, initialized by the root command; commands use `RequireConfig()`/`RequireClient()`
- **Config**: `internal/config` — layered discovery (editorconfig-style): global `~/.config/odoo-work-cli/config.toml` → `.odoo-work-cli.toml` dir walk root→cwd → env vars (highest priority). `Merge`, `ApplyDefaults`, `Validate`. Per-model `extra_fields` and `filters`, TUI `hours` limits, `keys` bindings, `bundesland` (holidays), `company_colors`
- **TUI**: `internal/tui` — Bubbletea v2 state machine (`stateLoading/Grid/Detail/Edit/Search/Help/Error`), weekly grid with per-day cells, detail view, add/edit form (shared via `editIsNew` flag), project/task search, async loading via `tea.Cmd` messages, attendance ticker, German public holidays via `wlbr/feiertage`, lipgloss styling in `styles.go`
- **TDD**: write failing tests first (RED), then implement (GREEN). Table-driven tests.

## Key Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/skilld-labs/go-odoo` — Odoo XML-RPC client (low-level only)
- `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2` — TUI framework and styling (vanity domain — NOT `github.com/charmbracelet`)
- `github.com/BurntSushi/toml` — TOML config parsing
- `github.com/wlbr/feiertage` — German public holidays
- `github.com/pquerna/otp` — TOTP code generation for 2FA web sessions

## Odoo API Notes

- Odoo 17: **API keys** required for XML-RPC auth (not plain passwords)
- API keys: Settings → Users → API Keys tab. Passed as password to `authenticate(db, login, api_key, "")`
- Database: `odoo.170` (discoverable via `POST /web/database/list` JSON-RPC)
- Web session auth (`/web/session/authenticate`) rejects API keys — it needs the real login password, plus a TOTP challenge via `/web/login/totp` when 2FA is enabled
- The go-odoo `Client.uid` field is private; search by `login` field to find current user
- go-odoo wrapper types: `*String`, `*Int`, `*Bool` use `.Get()`; `*Many2One` has `.ID` and `.Name` fields
- Many2One in raw `searchReadRaw`: `[]interface{}{int64, string}` or `false` → use `extractMany2OneName()`
- Attendance clock-in/out: JSON-RPC session cookie only — `POST /hr_attendance/systray_check_in_out`
  XML-RPC blocked for regular employees (ACL: officer group required)
- Attendance midnight wrap: two-pass query — (1) check_in today, (2) check_in < today AND check_out = false
- Odoo datetime format over RPC: `2006-01-02 15:04:05` (UTC)

## Secrets Management

Secrets are resolved at runtime via `[op_secrets]` in config files:
- Config files can contain `op://` vault references in `[op_secrets]` section
- Resolved keys: `url`, `database`, `username`, `api-key` (→ XML-RPC password), `password` (→ web session password for clock in/out), `totp_secret`
- At startup, if `op` CLI is installed, references are resolved via a single `op inject` call (shell-out, no CGO)
- Plain values (without `op://` prefix) in `[op_secrets]` are used as-is
- Falls back to environment variables: `ODOO_URL`, `ODOO_DATABASE`, `ODOO_USERNAME`, `ODOO_PASSWORD`, `ODOO_WEB_PASSWORD`, `ODOO_TOTP_SECRET`
- Priority: env vars > op_secrets > config file fields
- Plain-text `password` in config files is rejected at load time

## Development

```bash
mise run build        # compile binary (injects version via ldflags)
mise run run -- ...   # build and run the CLI
mise run test         # run all tests
mise run lint         # golangci-lint v2
mise run fmt          # gofmt
mise run release      # interactive semver release (scripts/release.sh)
```

Releases are built with GoReleaser (`mise run goreleaser-check` / `goreleaser-snapshot`).

## Conventions

- Conventional commits: `feat:`, `fix:`, `test:`, `refactor:`
- Go RST-style docstrings
- No personal data, credentials, or company-specific URLs in committed code
- golangci-lint v2, `version: "2"`, `default: standard`
