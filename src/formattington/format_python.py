import subprocess

from formattington._tools import ensure_valid_mode, find_tool, is_informational

FILE_EXTENSIONS: tuple[str, ...] = (".py",)

__all__ = ["black", "format_python", "FILE_EXTENSIONS"]


def black(
    *,
    check: bool = False,
    dry_run: bool = False,
    args: list[str] | None = None,
    verbose: bool = False,
) -> int:
    """Format Python files using Black.

    Respects ``[tool.black]`` configuration (for example ``pyproject.toml``
    ``line-length = 200``); no ``--line-length`` flag is added unless the
    caller forwards one via ``args``. Black is resolved via ``find_tool``
    (bundled lookup then ``PATH``); it is installed via pip
    (``pip install black`` or ``uv tool install black``) and is not bundled.
    """
    ensure_valid_mode(check=check, dry_run=dry_run)

    executable = find_tool("black", verbose=verbose)
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
        # Show diff without modifying files.
        arguments = ["--diff", *arguments]
    # Default: Black writes in place without a flag.

    return subprocess.run(
        [executable, *arguments],
        check=False,
    ).returncode


format_python = black
