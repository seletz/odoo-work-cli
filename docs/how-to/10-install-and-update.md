---
title: Install and update
icon: lucide/download
---
# Install and update

`odoo-work-cli` is a single static binary for macOS, Linux and Windows on
amd64 and arm64 (no Windows arm64).

## Homebrew (macOS, Linux)

```sh
brew tap seletz/tap
brew install odoo-work-cli
```

Homebrew also installs the shell completions.

### Update

Refresh the tap first, then upgrade. `brew upgrade` alone does not see a new
release until `brew update` has run:

```sh
brew update && brew upgrade odoo-work-cli
odoo-work-cli --version
```

If that still shows the old version, tap again:

```sh
brew untap seletz/tap
brew tap seletz/tap
brew install odoo-work-cli
```

## Release binaries

Download the archive for your platform from the
[releases page](https://github.com/seletz/odoo-work-cli/releases), unpack it
and put `odoo-work-cli` on your `PATH`. To update, replace the binary.

## From source

With Go installed:

```sh
go install github.com/seletz/odoo-work-cli/cmd/odoo-work-cli@latest
```

## Shell completions

For installs without Homebrew, add one of these:

```sh
# zsh (in .zshrc)
eval "$(odoo-work-cli completion zsh)"

# bash
odoo-work-cli completion bash > /usr/local/etc/bash_completion.d/odoo-work-cli

# fish
odoo-work-cli completion fish | source
```

## Next

Set up the connection: [Your first booking](../tutorials/20-first-booking.md).
