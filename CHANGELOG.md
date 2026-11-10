All notable changes of odoo-work-cli are listed here, newest first. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

Every pull request with a user-visible change adds a line under `Unreleased`
and updates the pages it affects. `mise run release` turns `Unreleased` into
the section of the new version, and that section is the text of the GitHub
release. See [Releases and docs](explanation/50-releases-and-docs.md).

## [Unreleased]

## [0.9.1] - 2026-11-10

### Fixed

- **MCP: 1Password is asked at most once per server (#117).** The server keeps
  the resolved secrets once `op inject` succeeded, even when the Odoo login
  fails afterwards; before, the next tool call asked again. An `auth` error
  discards them, and the next call resolves once more (rotated key or
  password).

### Added

- `odoo-work-cli mcp --secrets eager|lazy` (env `ODOO_WORK_CLI_MCP_SECRETS`,
  default `lazy`): `eager` resolves the secrets at server start, so the prompt
  comes when the client starts, except with the fake backend, which never
  needs secrets (#117).
- How-to "1Password prompts": when prompts come, and three setups (eager MCP,
  `op run --env-file`, `ODOO_WORK_CLI_NO_OP`) (#117).

## [0.9.0] - 2026-11-10

### Added

- **MCP server (#102).** `odoo-work-cli mcp` serves the CLI as MCP tools on
  stdin/stdout: 14 tools, reads (`whoami`, `projects`, `tasks`, `entries`,
  `timesheets`, `clock_status`, `clock_list`) and writes (`entries_add`,
  `entries_update`, `entries_delete`, `clock_in`, `clock_out`, `clock_add`,
  `clock_edit`). Every tool runs one CLI command with `--json`, so validation,
  fields and error kinds are the CLI's. `--read-only` serves only the reads.
  `odoo-work-cli mcp print` prints the client configuration or the
  `claude mcp add` line.
- The documentation site shows the version it documents (a banner on the home
  page and on the release notes, set from the release tag) and has a "Release
  notes" page, this changelog (#112).
- CI fails a pull request that changes `cmd/` or non-test code in `internal/`
  without an entry in `CHANGELOG.md`, unless it carries the label
  `no-changelog` (#112).
- Documentation page "Releases and docs": the release chain, the version on
  the site and the changelog rule (#112).

### Changed

- `mise run release` refuses to release with an empty `Unreleased` section,
  turns it into the section of the new version and commits it before tagging.
  The release verification fails when the changelog has no section for the tag
  (#112).
- The GitHub release text is the changelog section of the version. The commit
  list below it no longer repeats every change (merge commits and branch
  commits repeated by their PR are left out) and includes `docs:` and `ci:`
  commits, grouped by type (#112).
- The TUI docs describe the 80-column layout and the 68-column minimum, and
  the troubleshooting page covers "terminal too narrow" (#112).
- The TUI screenshots in the docs look like the TUI in a real terminal: set in
  Fira Code with Ghostty's cell metrics and the Catppuccin Mocha palette
  instead of Go Mono with raw ANSI colours (#115).

## [0.8.5] - 2026-11-10

### Changed

- **TUI fits 80 columns (#107).** In narrow terminals the weekly grid squeezes
  the project/task column and shortens long names in the middle
  (`[EXA] …pha: Design`), so rows stay distinct. Below 68 columns the TUI shows
  a "terminal too narrow" notice instead of a broken grid.
- **Docs are part of the release (#110).** Every release runs tests, lint, the
  docs check and the strict docs build before GoReleaser, then deploys the
  docs site from the tag. Tests and lint also run in CI on every pull request.

## [0.8.4] - 2026-11-10

### Added

- **Man pages (#103).** `man odoo-work-cli` and one page per command (for
  example `man odoo-work-cli-entries-add`), installed by the Homebrew cask.
  Each page has NAME, SYNOPSIS, DESCRIPTION, OPTIONS, OUTPUT (JSON fields),
  ENVIRONMENT, FILES, EXIT STATUS with error kinds, EXAMPLES and SEE ALSO. The
  same reference is on the site. `--help` texts are more complete too.

### Changed

- **Legible screenshots on the docs site (#101).** TUI screenshots are PNGs
  with an embedded font, so they look the same in every browser.

### Fixed

- **TUI: overlays stay within the terminal (#105).** The search overlay no
  longer extends past a 120-column terminal; long content is truncated.

## [0.8.3] - 2026-11-10

### Added

- **Documentation site** at <https://seletz.github.io/odoo-work-cli/>:
  tutorials, how-to guides, a full command reference and explanations, for the
  CLI, the TUI and agent use (JSON output, error kinds, `llm --skill`, use as
  an MCP tool). The TUI screenshots and CLI output samples are generated from
  the fake backend, and CI fails when they are out of date (#99).

### Changed

- The readme is short and points to the site.

### Fixed

- **TUI:** detail, edit and help overlays no longer leave escape-sequence
  garbage to the right of the box, and they are centred by visible width.

## [0.8.2] - 2026-11-10

### Fixed

- **Expired web sessions recover on their own (#73).** After a few days Odoo
  expires the web session used for clock in/out. The CLI now logs in again once
  when Odoo reports an expired session or redirects to the login page. Only if
  that fails does it report an `auth` error.
- **The TUI shows clock errors.** A failed clock in/out in the TUI shows the
  error instead of failing silently.

## [0.8.1] - 2026-11-09

### Fixed

- An unknown project, task or other record ID gives one clean message, such as
  `project 67 does not exist`, instead of Odoo's raw "Record does not exist or
  has been deleted" text (#95). The message is the same with and without
  `--dry-run`, and in table and JSON output. The error kind is unchanged.
- The readme and `llm --skill` say to run `brew update` before
  `brew upgrade odoo-work-cli`, because the upgrade alone does not see a new
  release (#96).

## [0.8.0] - 2026-11-09

### Added

- **Several attendance records per day (#85).** `clock add --in --out [--date]`
  creates an additional record and never overwrites. If the new span overlaps
  an existing record, including a running one or one from the night before, it
  fails with `conflict` and changes nothing. Records that only touch (12:00
  out, 12:00 in) are fine.
- **Find projects and tasks without listing everything (#86).**
  `projects --search TEXT` matches the project name or customer, and
  `tasks [project-id] --search TEXT` matches the task name. Odoo does the
  case-insensitive matching, combined with the configured filters.
- **`--dry-run` for `entries add` and `entries update` (#87).** It validates
  the input, resolves project and task, and prints the entry that would be
  written, without writing it. With `--json` the output also lists possible
  duplicates on the same day.
- **Test without 1Password and without Odoo (#88).** Credentials already set
  through environment variables are no longer requested from 1Password, and
  `ODOO_WORK_CLI_NO_OP=1` ignores `[op_secrets]` entirely.
  `ODOO_WORK_CLI_BACKEND=fake` (or `--backend fake`) runs every command against
  an offline fake with made-up data. It keeps its state in a file, so
  workflows such as list → add → list can be tested end to end.

## [0.7.1] - 2026-11-09

### Fixed

- Invalid input is reported before 1Password is asked (#84). Configuration and
  secrets load only when a command needs Odoo, after it has checked its flags
  and arguments. A wrong call such as `clock list --date … --week …`, a
  malformed date or ID, or a future `--at` time gives `kind: validation`
  immediately, instead of `secrets_locked` and a 5-second wait while
  1Password is locked.

## [0.7.0] - 2026-11-09

### Added

- **Scripts and agents (#75, #74).** Every reading command takes `--json` or
  `--format table|json|jsonl`. Output has stable snake_case fields, numeric
  IDs and durations in hours and minutes, and an empty result is `[]`. Errors
  become a single `{"error":{"kind":…,"message":…}}` object on stdout with
  exit code 1, using stable kinds (`auth`, `access_denied`, `not_found`,
  `validation`, `conflict`, `network`, `config`, `secrets_locked`, `internal`).
  Write commands return the object they created or changed.
  `odoo-work-cli llm --skill` prints a guide written for agents.
- **Safer attendance backfill (#76).** `clock list --date D` or
  `--week YYYY-Www` shows the existing records first. `clock edit` prints the
  old and the new values instead of overwriting silently.
  `clock edit --create-only` refuses to touch a day that already has a record
  and fails with `conflict`.
- **TUI attendance (#78).** The weekly view shows check-in, check-out and
  attendance per day, with the difference to the booked hours. Past workdays
  without attendance are flagged.

### Changed

- A locked 1Password fails within about 5 s with `secrets_locked` instead of
  hanging for a minute (#77). `ODOO_OP_TIMEOUT` changes the timeout.
