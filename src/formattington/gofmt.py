import subprocess
import sys

from formattington._tools import (
    EXIT_FORMAT_REQUIRED,
    ensure_valid_mode,
    find_tool,
    is_informational,
)

FILE_EXTENSIONS: tuple[str, ...] = (".go",)

__all__ = ["gofmt", "FILE_EXTENSIONS"]


def gofmt(
    *,
    check: bool = False,
    dry_run: bool = False,
    args: list[str] | None = None,
    verbose: bool = False,
) -> int:
    """Format Go files using gofmt.

    Respects Go's native formatting; no flags are added unless the
    caller forwards them via ``args``. gofmt is resolved via
    ``find_tool`` (bundled lookup then ``PATH``); it is part of the
    Go installation (https://go.dev/doc/go1.19#gofmt) with the
    ``gofmt`` executable on ``PATH`` and is not bundled.

    Formattington mode mapping (verified against ``gofmt -h``):

    - ``check`` -> ``gofmt -l``: list files whose formatting differs
      from gofmt; exit 1 if any listed, else 0. ``gofmt -l`` exits 0
      even when files differ, so Formattington captures stdout and
      reports ``EXIT_FORMAT_REQUIRED`` when non-empty (like shfmt
      ``-l``).
    - ``dry_run`` -> ``gofmt -d``: display diffs instead of
      rewriting files; do not modify files.
    - Default -> ``gofmt -w``: write result to the source file
      instead of stdout (format in place).
    """
    ensure_valid_mode(check=check, dry_run=dry_run)

    executable = find_tool("gofmt", verbose=verbose)
    forwarded = list(args or [])

    # Preserve native informational commands without adding format flags.
    if is_informational(forwarded):
        return subprocess.run(
            [executable, *forwarded],
            check=False,
        ).returncode

    if check:
        # gofmt -l lists differing files but exits 0 in all cases.
        result = subprocess.run(
            [executable, "-l", *forwarded],
            capture_output=True,
            text=True,
            check=False,
        )
        if result.stdout:
            print(result.stdout, end="")
        if result.stderr:
            print(result.stderr, end="", file=sys.stderr)
        if result.returncode != 0:
            return result.returncode
        if result.stdout.strip():
            print(
                "formattington: formatting changes required.",
                file=sys.stderr,
            )
            return EXIT_FORMAT_REQUIRED
        return 0

    if dry_run:
        # Show diffs without modifying files.
        return subprocess.run(
            [executable, "-d", *forwarded],
            check=False,
        ).returncode

    # Default: format in place.
    return subprocess.run(
        [executable, "-w", *forwarded],
        check=False,
    ).returncode
