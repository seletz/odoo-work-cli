# odoo-work-cli completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(odoo-work-cli completion zsh)

To load completions for every new session, execute once:

#### Linux:

	odoo-work-cli completion zsh > "${fpath[1]}/_odoo-work-cli"

#### macOS:

	odoo-work-cli completion zsh > $(brew --prefix)/share/zsh/site-functions/_odoo-work-cli

You will need to start a new shell for this setup to take effect.


```
odoo-work-cli completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --backend string   backend: odoo (default) or fake (offline, made-up data; env ODOO_WORK_CLI_BACKEND)
      --config string    config file path (skip discovery)
```

### SEE ALSO

* [odoo-work-cli completion](odoo-work-cli_completion.md)	 - Generate the autocompletion script for the specified shell

