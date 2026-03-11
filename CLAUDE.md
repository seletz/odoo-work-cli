# odoo-work-cli

CLI tool for managing Odoo 17 timesheets, written in Go.

## Design Doc

- `~/develop/notes/00 Persönlich/00.03 Projekte/odoo-work-cli/odoo-work-cli.md`

**IMPORTANT:** Always use TDD, red/green phase

## Progress Tracking and ODOO API Notes

- `~/develop/notes/00 Persönlich/00.03 Projekte/odoo-work-cli/odoo-work-cli progress.md`

## Project Structure

```
cmd/odoo-work-cli/main.go    # Cobra CLI entrypoint
internal/config/              # Config loading (env vars, TOML)
internal/odoo/                # Odoo client interface + XML-RPC implementation
internal/display/             # Output formatting (placeholder, M4)
```

## Architecture

- **Interface-driven**: `odoo.Client` interface in `internal/odoo/client.go`; real implementation in `xmlrpc.go` using `github.com/skilld-labs/go-odoo`
- **Config**: `internal/config` loads from env vars (`LoadFromEnv`) or TOML file (`LoadFromTOML`), with `Merge` for overlaying
- **TDD**: write failing tests first (RED), then implement (GREEN). Use table-driven tests.

## Key Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/skilld-labs/go-odoo` — Odoo XML-RPC client (generated model types)
- `github.com/BurntSushi/toml` — TOML config parsing

## Odoo API Notes

- Odoo 17 requires **API keys** for XML-RPC authentication (not plain passwords)
- API keys are created in: Odoo Settings > Users > API Keys tab
- The API key is passed as the password parameter to `authenticate(db, login, api_key, "")`
- Database name: `odoo.170` (discoverable via `POST /web/database/list` JSON-RPC)
- The go-odoo `Client.uid` field is private; search by `login` field to find current user
- go-odoo wrapper types: `*String`, `*Int`, `*Bool` use `.Get()`; `*Many2One` has `.ID` and `.Name` fields

## Secrets Management

Secrets are resolved at runtime via `[op_secrets]` in config files:
- Config files can contain `op://` vault references in `[op_secrets]` section
- At startup, if `op` CLI is installed, references are resolved via `op read`
- Plain values (without `op://` prefix) in `[op_secrets]` are used as-is
- Falls back to environment variables: `ODOO_URL`, `ODOO_DATABASE`, `ODOO_USERNAME`, `ODOO_PASSWORD`
- Priority: env vars > op_secrets > config file fields
- Plain-text `password` in config files is still rejected

## Development

```bash
mise run build        # compile binary
mise run test         # run all tests
mise run lint         # golangci-lint v2
mise run fmt          # gofmt
```

## Conventions

- Go RST-style docstrings
- No personal data, credentials, or company-specific URLs in committed code
- golangci-lint v2 with `version: "2"` in `.golangci.yml`, `default: standard` linters
