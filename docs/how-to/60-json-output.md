---
title: Script with JSON output
icon: lucide/braces
---
# Script with JSON output

How to use `odoo-work-cli` from shell scripts. Never parse the tables: long
names shift the columns.

## Ask for JSON

Every reading and writing command accepts `--json` (= `--format json`) and
`--format jsonl`:

```sh
odoo-work-cli entries --week 2026-W15 --json | jq '.[] | {id, project, minutes}'
odoo-work-cli clock status --json | jq .clocked_in
odoo-work-cli entries add --project-id 42 --hours 1:30 --description "Review" --json | jq .id
```

```console
$ odoo-work-cli clock list --date 2026-04-08 --json
--8<-- "docs/assets/cli/clock-list-date-json.txt"
```

An empty result is `[]`, never empty output. Field names never change; new
fields may appear. Details per object: [JSON output](../reference/20-json-output.md).

## Handle errors

With `--json`, a failure prints exactly one object on **stdout** and exits
with code `1`:

```console
$ odoo-work-cli entries delete 999 --json
--8<-- "docs/assets/cli/error-not-found.txt"
```

Branch on `kind`, not on `message`:

```sh
out=$(odoo-work-cli clock edit --date 2026-09-14 --in 08:00 --out 16:30 --create-only --json)
case $? in
  0) echo "created #$(jq .id <<<"$out")" ;;
  *) case $(jq -r .error.kind <<<"$out") in
       conflict) echo "day already has a record, skipped" ;;
       *)        jq -r .error.message <<<"$out" >&2; exit 1 ;;
     esac ;;
esac
```

All kinds: [Errors and exit codes](../reference/30-errors.md).

## jsonl

`--format jsonl` prints one compact object per line, handy for `while read`:

```sh
odoo-work-cli projects --format jsonl | while read -r p; do
  jq -r '"\(.id)\t\(.name)"' <<<"$p"
done
```

## Validate against the schemas

The JSON Schemas (draft 2020-12) are in
[`internal/output/schemas/`](https://github.com/seletz/odoo-work-cli/tree/develop/internal/output/schemas)
and are the contract. The test suite checks every command's output against
them.
