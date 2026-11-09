
from formattington._tools import ensure_valid_mode, find_tool, run_tool

FILE_EXTENSIONS: tuple[str, ...] = (".sh", ".bash")

__all__ = ["shfmt", "FILE_EXTENSIONS"]


def shfmt(
    *,
    check: bool = False,
    dry_run: bool = False,
    args: list[str] | None = None,
    verbose: bool = False,
) -> int:
    """Format shell scripts with shfmt."""
    ensure_valid_mode(check=check, dry_run=dry_run)

    executable = find_tool("shfmt", verbose=verbose)
    extra_args = args or []

    # Formattington owns the mode; shfmt owns all other flags.
    if check:
        return run_tool(
            executable,
            ["-l", *extra_args],
            capture_output=True,
        )

    if dry_run:
        return run_tool(executable, ["-d", *extra_args])

    return run_tool(
        executable,
        ["-w", *extra_args],
    )
