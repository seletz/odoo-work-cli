# odoo-work-cli

CLI tool for managing Odoo 17 timesheets from the terminal.

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

## Configuration

### Secrets (via 1Password)

Copy `.env.1p` and fill in your 1Password references:

```
ODOO_URL={{ op://your-vault/odoo/url }}
ODOO_DATABASE={{ op://your-vault/odoo/database }}
ODOO_USERNAME={{ op://your-vault/odoo/username }}
ODOO_PASSWORD={{ op://your-vault/odoo/password }}
```

Then run `mise run inject-env` to generate `.env`.

### Config file (optional)

Non-secret configuration can go in `~/.config/odoo-work-cli/config.toml`:

```toml
url = "https://odoo.example.com"
database = "mydb"
```

## Usage

```bash
./odoo-work-cli --help
./odoo-work-cli projects
./odoo-work-cli tasks
./odoo-work-cli timesheets
./odoo-work-cli fields
./odoo-work-cli whoami
```

## Development

```bash
mise run fmt     # format code
mise run lint    # run linters
mise run test    # run tests
mise run build   # compile binary
```
