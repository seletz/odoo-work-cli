#!/bin/sh
set -e
rm -rf completions
mkdir completions
for sh in bash zsh fish; do
    go run ./cmd/odoo-work-cli completion "$sh" > "completions/odoo-work-cli.$sh"
done
