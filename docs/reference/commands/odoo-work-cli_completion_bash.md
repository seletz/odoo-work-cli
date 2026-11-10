# odoo-work-cli completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(odoo-work-cli completion bash)

To load completions for every new session, execute once:

#### Linux:

	odoo-work-cli completion bash > /etc/bash_completion.d/odoo-work-cli

#### macOS:

	odoo-work-cli completion bash > $(brew --prefix)/etc/bash_completion.d/odoo-work-cli

You will need to start a new shell for this setup to take effect.


```
odoo-work-cli completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --backend string   backend: odoo (default) or fake (offline, made-up data; env ODOO_WORK_CLI_BACKEND)
      --config string    config file path (skip discovery)
```

### SEE ALSO

* [odoo-work-cli completion](odoo-work-cli_completion.md)	 - Generate the autocompletion script for the specified shell

