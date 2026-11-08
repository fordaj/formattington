
import subprocess

from formattington._tools import find_tool


def clang_format(
    *,
    check: bool = False,
    dry_run: bool = False,
    args: list[str] | None = None,
    verbose: bool = False,
) -> int:
    """Format C/C++ files using clang-format."""
    if check and dry_run:
        raise ValueError("check and dry_run cannot both be enabled")

    executable = find_tool("clang-format", verbose=verbose)
    arguments = list(args or [])

    # Preserve native informational commands without adding formatting flags.
    if any(arg in {"--help", "-help", "--version", "-version"} for arg in arguments):
        return subprocess.run(
            [executable, *arguments],
            check=False,
        ).returncode

    if check:
        # Fail if any file needs formatting; don't modify files.
        arguments = ["--dry-run", "--Werror", *arguments]
    elif dry_run:
        # Validate without modifying files.
        arguments = ["--dry-run", *arguments]
    elif "-i" not in arguments and "--in-place" not in arguments:
        # Formattington formats in place by default.
        arguments.insert(0, "-i")

    return subprocess.run(
        [executable, *arguments],
        check=False,
    ).returncode
