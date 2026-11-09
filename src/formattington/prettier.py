import subprocess

from formattington._tools import ensure_valid_mode, find_tool, is_informational

FILE_EXTENSIONS: tuple[str, ...] = (
    ".js",
    ".ts",
    ".tsx",
    ".jsx",
    ".json",
    ".html",
    ".css",
    ".scss",
    ".yaml",
    ".yml",
    ".md",
    ".markdown",
)

__all__ = ["prettier", "FILE_EXTENSIONS"]


def prettier(
    *,
    check: bool = False,
    dry_run: bool = False,
    args: list[str] | None = None,
    verbose: bool = False,
) -> int:
    """Format JS/TS/CSS/JSON/HTML/YAML/Markdown files using Prettier.

    Respects project ``.prettierrc`` (and other Prettier configuration) --
    no ``--config`` flag is added unless the caller forwards one via
    ``args``. Prettier is resolved via ``find_tool`` (bundled lookup then
    ``PATH``); it is not bundled and is installed via npm (for example
    ``npm install -g prettier`` or ``npx prettier``) with the
    ``prettier`` executable on ``PATH``.

    Formattington mode mapping (Prettier has no unified-diff flag such as
    ``--diff`` -- see https://prettier.io/docs/en/cli):

    - ``check`` -> ``--check``: fail if any file needs formatting;
      do not modify files.
    - ``dry_run`` -> ``--list-different``: list files that differ from
      Prettier formatting; do not modify files. This is the dry-run
      equivalent for Prettier.
    - Default -> ``--write``: format in place (unless ``--write`` (or its
      ``-w`` alias) is already given via ``args``). If the caller forwards
      ``--check``/``-c`` or ``--list-different``/``-l`` via ``args`` the
      default also omits ``--write`` so the forwarded mode is preserved.

    The extension list above covers Prettier-supported types for future
    file discovery (so the map lives in one place here). ``.md`` and
    ``.markdown`` are included because Prettier also formats Markdown;
    there is no separate Markdown integration -- the ``md`` and
    ``markdown`` CLI aliases resolve to this Prettier integration and
    Prettier's native configuration (for example ``.prettierrc``) is
    respected.
    """
    ensure_valid_mode(check=check, dry_run=dry_run)

    executable = find_tool("prettier", verbose=verbose)
    arguments = list(args or [])

    # Preserve native informational commands without adding formatting flags.
    if is_informational(arguments):
        return subprocess.run(
            [executable, *arguments],
            check=False,
        ).returncode

    if check:
        # Fail if any file needs formatting; don't modify files.
        arguments = ["--check", *arguments]
    elif dry_run:
        # List files that differ without modifying files.
        arguments = ["--list-different", *arguments]
    elif not any(
        arg in {"--write", "-w", "--check", "-c", "--list-different", "-l"} for arg in arguments
    ):
        # Formattington formats in place by default.
        arguments.insert(0, "--write")

    return subprocess.run(
        [executable, *arguments],
        check=False,
    ).returncode
