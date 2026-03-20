#!/bin/sh
set -e
rm -rf completions
mkdir completions
go run ./cmd/odoo-work-cli completion bash > "completions/odoo-work-cli.bash"
go run ./cmd/odoo-work-cli completion zsh > "completions/_odoo-work-cli"
go run ./cmd/odoo-work-cli completion fish > "completions/odoo-work-cli.fish"
