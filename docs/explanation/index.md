---
title: Explanation
icon: lucide/lightbulb
weight: 4
---
# Explanation

Background and design decisions: why `odoo-work-cli` works the way it does.

- [Config layering](config-layering.md): global file, directory walk,
  `[op_secrets]`, environment.
- [Secrets](secrets.md): three credentials, and where they come from.
- [Two transports](transports.md): XML-RPC with an API key, a web session with
  a password.
- [The attendance model](attendance-model.md): records, days, midnight, lunch.
- [Releases and docs](50-releases-and-docs.md): the release chain, the version
  on the site, the changelog.
