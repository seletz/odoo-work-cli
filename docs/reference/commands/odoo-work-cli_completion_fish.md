# odoo-work-cli completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	odoo-work-cli completion fish | source

To load completions for every new session, execute once:

	odoo-work-cli completion fish > ~/.config/fish/completions/odoo-work-cli.fish

You will need to start a new shell for this setup to take effect.


```
odoo-work-cli completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --backend string   backend: odoo (default) or fake (offline, made-up data; env ODOO_WORK_CLI_BACKEND)
      --config string    config file path (skip discovery)
```

### SEE ALSO

* [odoo-work-cli completion](odoo-work-cli_completion.md)	 - Generate the autocompletion script for the specified shell

