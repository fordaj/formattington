import subprocess

from formattington._tools import find_tool

FILE_EXTENSIONS: tuple[str, ...] = (".toml",)

__all__ = ["taplo", "FILE_EXTENSIONS"]


def taplo(
    *,
    check: bool = False,
    dry_run: bool = False,
    args: list[str] | None = None,
    verbose: bool = False,
) -> int:
    """Format TOML files using Taplo (``taplo fmt``).

    Respects project ``taplo.toml`` configuration (found automatically
    by Taplo); no ``--config`` flag is added unless the caller forwards
    one via ``args``. Taplo is resolved via ``find_tool`` (bundled
    copy under ``tools/taplo/<version>/<platform>/`` first, then
    ``PATH``).

    Formattington mode mapping (verified against ``taplo fmt --help``):

    - ``check`` -> ``taplo fmt --check``: dry-run and report files
      that are not correctly formatted; exit 1 if any differ.
    - ``dry_run`` -> ``taplo fmt --diff``: print the differences in
      patch format to stdout without modifying files.
    - Default -> ``taplo fmt``: format in place.

    ``--colors never`` is forced (except for informational
    passthroughs) so diff/log output is deterministic in captured
    output on every platform.
    """
    if check and dry_run:
        raise ValueError("check and dry_run cannot both be enabled")

    executable = find_tool("taplo", verbose=verbose)
    forwarded = list(args or [])

    # Preserve native informational commands without adding format flags.
    if any(arg in {"--help", "-help", "--version", "-version", "-h", "-V"} for arg in forwarded):
        return subprocess.run(
            [executable, *forwarded],
            check=False,
        ).returncode

    arguments = ["fmt"]
    if "--colors" not in forwarded:
        arguments += ["--colors", "never"]
    if check:
        # Fail if any file needs formatting; don't modify files.
        arguments.append("--check")
    elif dry_run:
        # Show diff without modifying files.
        arguments.append("--diff")
    # Default: taplo fmt writes in place without a flag.
    arguments.extend(forwarded)

    return subprocess.run(
        [executable, *arguments],
        check=False,
    ).returncode
