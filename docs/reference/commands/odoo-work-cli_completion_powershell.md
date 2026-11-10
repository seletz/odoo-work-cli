# odoo-work-cli completion powershell

Generate the autocompletion script for powershell

### Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	odoo-work-cli completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
odoo-work-cli completion powershell [flags]
```

### Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --backend string   backend: odoo (default) or fake (offline, made-up data; env ODOO_WORK_CLI_BACKEND)
      --config string    config file path (skip discovery)
```

### SEE ALSO

* [odoo-work-cli completion](odoo-work-cli_completion.md)	 - Generate the autocompletion script for the specified shell

