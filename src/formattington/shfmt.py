
from pathlib import Path

from formattington._tools import find_tool, run_tool


def shfmt(
    paths: str | Path | list[str | Path] = ".",
    *,
    check: bool = False,
    dry_run: bool = False,
    args: list[str] | None = None,
    verbose: bool = False,
) -> int:
    """Format shell scripts with shfmt."""
    if check and dry_run:
        raise ValueError("check and dry_run cannot both be enabled")

    executable = find_tool("shfmt", verbose=verbose)
    extra_args = args or []

    # Formattington owns the mode; shfmt owns all other flags.
    if check:
        return run_tool(
            executable,
            ["-l", *extra_args],
            check_output=True,
        )

    if dry_run:
        return run_tool(executable, ["-d", *extra_args])

    return run_tool(
        executable,
        ["-w", *extra_args],
    )
