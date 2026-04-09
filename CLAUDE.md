# odoo-work-cli

CLI tool for managing Odoo 17 timesheets, written in Go.

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

## Project Structure

```
cmd/odoo-work-cli/main.go    # Cobra CLI entrypoint
internal/config/              # Config loading (env vars, TOML, layered discovery)
internal/odoo/                # Odoo client interface + XML-RPC implementation
internal/display/             # Output formatting
```

## Architecture

- **Interface-driven**: `odoo.Client` interface in `internal/odoo/client.go`
- **go-odoo**: use low-level API only (`ExecuteKw`, `SearchRead`, `Create`) — do NOT use generated model wrappers for timesheets (missing custom fields, missing `validated_status`)
- **Config**: layered discovery (editorconfig-style): global → dir walk root→cwd → env vars
- **TDD**: write failing tests first (RED), then implement (GREEN). Table-driven tests.

## Key Dependencies

- `github.com/spf13/cobra` — CLI
- `github.com/skilld-labs/go-odoo` — Odoo XML-RPC (low-level only)
- `github.com/BurntSushi/toml` — config
- `charm.land/bubbletea/v2` — TUI (vanity domain — NOT `github.com/charmbracelet`)
- `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`, `charm.land/huh/v2`

## Odoo API Notes

- Odoo 17: **API keys** required for XML-RPC auth (not plain passwords)
- API keys: Settings → Users → API Keys tab. Passed as password to `authenticate()`
- Database: `odoo.170`
- go-odoo types: `*String`, `*Int`, `*Bool` use `.Get()`; `*Many2One` has `.ID` and `.Name`
- Many2One in raw `searchReadRaw`: `[]interface{}{int64, string}` or `false` → use `extractMany2OneName()`
- Attendance clock-in/out: JSON-RPC session cookie only — `POST /hr_attendance/systray_check_in_out`
  XML-RPC blocked for regular employees (ACL: officer group required)
- Attendance midnight wrap: two-pass query — (1) check_in today, (2) check_in < today AND check_out = false

## Secrets Management

- `[op_secrets]` in config: `op://` references resolved at runtime via `op read` (shell-out, no CGO)
- Falls back to env vars: `ODOO_URL`, `ODOO_DATABASE`, `ODOO_USERNAME`, `ODOO_PASSWORD`
- Priority: env vars > op_secrets > config fields
- Plain `password` field in config is rejected at load time

## Development

```bash
mise run build        # compile
mise run test         # all tests
mise run lint         # golangci-lint v2
mise run fmt          # gofmt
```

## Conventions

- Conventional commits: `feat:`, `fix:`, `test:`, `refactor:`
- No personal data, credentials, or company-specific URLs in committed code
- golangci-lint v2, `version: "2"`, `default: standard`
