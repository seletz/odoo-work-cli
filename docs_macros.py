"""Build-time macros for the docs site (Zensical, ``zensical.toml``).

Registered through the Markdown extension ``zensical.extensions.macros``.
Only pages with ``render_macros: true`` in their front matter run through
Jinja (``render_by_default = false``); ``on_error_fail = true`` makes a macro
error fail the build.

- :func:`version_banner` renders an admonition with the version the site
  documents, and a link to its GitHub release when the version is a release.
- :func:`changelog` renders ``CHANGELOG.md`` (the "Release notes" page).

The version comes from the environment variable ``DOCS_VERSION`` (the release
workflow sets it from the tag); without it, from ``git describe`` (for
example ``0.8.5-3-gabc123``), and ``dev`` outside a git checkout.
"""

from __future__ import annotations

import os
import re
import subprocess
from pathlib import Path

#: Repository slug used when ``repo_url`` is not configured.
_DEFAULT_REPO = "seletz/odoo-work-cli"

#: A released version: no commits after the tag, no ``-dirty`` marker.
_RELEASE = re.compile(r"^\d+\.\d+\.\d+$")


def docs_version(root: Path, environ: dict[str, str] | None = None) -> str:
    """Return the version the site documents.

    Order: ``DOCS_VERSION``, then ``git describe --tags --match 'v*' --always
    --dirty``, then ``dev``. A leading ``v`` is dropped.

    :param root: project root (the git checkout).
    :param environ: environment to read ``DOCS_VERSION`` from, default
        ``os.environ``.
    :returns: the version, e.g. ``0.8.6`` or ``0.8.5-3-gabc123``.
    """
    env = os.environ if environ is None else environ
    version = (env.get("DOCS_VERSION") or "").strip()
    if not version:
        try:
            proc = subprocess.run(
                ["git", "describe", "--tags", "--match", "v*", "--always", "--dirty"],
                cwd=root,
                capture_output=True,
                text=True,
                timeout=20,
                check=True,
            )
            version = proc.stdout.strip()
        except Exception:
            version = ""
    return version.removeprefix("v") or "dev"


def _repo_url(env) -> str:
    """Return the repository URL from ``repo_url``, else the default."""
    try:
        url = env.conf.get("repo_url", "") or ""
    except Exception:
        url = ""
    return (url or f"https://github.com/{_DEFAULT_REPO}").rstrip("/")


def define_env(env) -> None:
    """Register the macros with the Zensical macro environment.

    :param env: the environment provided by the extension; ``env.conf``
        carries the resolved project configuration.
    """
    root = Path(env.conf.get("root_dir", ".")).resolve()
    repo_url = _repo_url(env)

    @env.macro
    def version_banner() -> str:
        """Render the version admonition.

        :returns: a Markdown ``info`` admonition naming the version; a release
            links to its GitHub release, anything else is marked as a
            development build.
        """
        version = docs_version(root)
        if _RELEASE.match(version):
            body = f"Documentation for odoo-work-cli [**{version}**]({repo_url}/releases/tag/v{version})."
        else:
            body = f"Documentation for odoo-work-cli **{version}** (development build, not a release)."
        return f'!!! info "Version"\n\n    {body}'

    @env.macro
    def changelog() -> str:
        """Render ``CHANGELOG.md`` without its intro paragraph's page title.

        :returns: the changelog Markdown.
        """
        return (root / "CHANGELOG.md").read_text(encoding="utf-8")
