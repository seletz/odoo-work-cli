---
title: Customise the TUI
icon: lucide/palette
---
# Customise the TUI

How to change keys, colours, hour limits and holidays. All of it goes into
your config file (`~/.config/odoo-work-cli/config.toml`, or a
`.odoo-work-cli.toml` in a project directory); see
[Configuration](../reference/40-configuration.md).

## Rebind keys

Override single actions in `[keys]`; the others keep their defaults. A value
is a key or a list of keys:

```toml
[keys]
global_quit = ["q", "ctrl+c"]
grid_search = ["/", "f"]
detail_delete = "x"
```

All actions and their defaults: [TUI keys and views](../reference/50-tui-keys.md).
Bindings never steal typing: while a text field has focus, printable keys go
into the field.

## Colour rows by company

The row prefix (`[EXA]`) is the first three letters of the project's
company. Give each company a colour, as an ANSI 256 colour number or a hex
value:

```toml
[company_colors]
"Example Consulting" = "5"     # magenta
"Wildflower Labs" = "#5fd75f"
```

## Set the hour limits

The colours of the day and week totals follow these limits:

```toml
[hours]
daily_low = 6.0     # below: yellow
daily_high = 9.0    # above: red
weekly_low = 35.0
weekly_high = 40.0
```

## Pick the holidays

Public holidays are German, per federal state. The default is
Baden-Württemberg:

```toml
bundesland = "Bayern"
```

Valid values: Baden-Württemberg, Bayern, Berlin, Brandenburg, Bremen,
Hamburg, Hessen, Mecklenburg-Vorpommern, Niedersachsen, Nordrhein-Westfalen,
Rheinland-Pfalz, Saarland, Sachsen, Sachsen-Anhalt, Schleswig-Holstein,
Thüringen.
