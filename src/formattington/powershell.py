import subprocess
from pathlib import Path

from formattington._tools import ensure_valid_mode, find_tool, is_informational

FILE_EXTENSIONS: tuple[str, ...] = (".ps1", ".psm1")

__all__ = ["powershell", "FILE_EXTENSIONS"]

# PowerShell 7+ (pwsh) is required for PSScriptAnalyzer; on Windows the
# in-box Windows PowerShell (powershell.exe) is tried as a fallback.
SHELL_CANDIDATES: tuple[str, ...] = ("pwsh", "powershell")

# Positional tokens ending in these suffixes (or existing on disk) are
# analyzed scripts; every other token is forwarded to
# Invoke-ScriptAnalyzer as a parameter (e.g. -Severity Error).
_PATH_SUFFIXES = frozenset({".ps1", ".psm1"})


def _find_shell(*, verbose: bool = False) -> str:
    """Resolve ``pwsh`` (preferred) or ``powershell``, else raise.

    The error message follows the ``find_tool`` convention so the CLI
    prints a single ``formattington: Could not find ...`` line and
    exits 127.
    """
    first_error: FileNotFoundError | None = None
    for candidate in SHELL_CANDIDATES:
        try:
            return find_tool(candidate, verbose=verbose)
        except FileNotFoundError as exc:
            if first_error is None:
                first_error = exc
    raise FileNotFoundError(
        "Could not find 'pwsh' or 'powershell'. "
        "Expected PowerShell 7+ (pwsh) on PATH and the PSScriptAnalyzer "
        "module ('Install-Module PSScriptAnalyzer' inside pwsh)."
    ) from first_error


def _is_path_arg(arg: str) -> bool:
    """Whether a forwarded token is a PowerShell file to analyze."""
    if arg.startswith("-"):
        return False
    return Path(arg).suffix.lower() in _PATH_SUFFIXES


def _single_quoted(value: str) -> str:
    """Quote a path for a PowerShell single-quoted string."""
    return "'" + value.replace("'", "''") + "'"


def powershell(
    *,
    check: bool = False,
    dry_run: bool = False,
    args: list[str] | None = None,
    verbose: bool = False,
) -> int:
    """Lint PowerShell files with PSScriptAnalyzer (no in-place formatting).

    PSScriptAnalyzer is a PowerShell module (``Install-Module
    PSScriptAnalyzer``) run through ``pwsh -NoProfile -Command
    "Invoke-ScriptAnalyzer ..."``. It reports findings but never
    rewrites files, so every mode behaves the same: findings exit 1,
    no findings exit 0. ``check`` and ``dry_run`` are accepted for
    CLI uniformity; they do not change the lint invocation.

    Files (``.ps1``/``.psm1`` paths, or existing files) are passed via
    ``-Path``; any other forwarded token becomes an
    ``Invoke-ScriptAnalyzer`` parameter, so e.g.
    ``formattington ps1 -Severity Error`` forwards ``-Severity`` and
    ``Error``. With no files, the current directory is analyzed
    recursively. Module settings files (for example
    ``PSScriptAnalyzerSettings.psd1``) are honored via the forwarded
    ``-Settings`` parameter.
    """
    ensure_valid_mode(check=check, dry_run=dry_run)

    executable = _find_shell(verbose=verbose)
    forwarded = list(args or [])

    # Preserve native informational commands without invoking the analyzer.
    if is_informational(forwarded):
        return subprocess.run(
            [executable, *forwarded],
            check=False,
        ).returncode

    paths = [arg for arg in forwarded if _is_path_arg(arg)]
    params = [arg for arg in forwarded if not _is_path_arg(arg)]

    if paths:
        path_list = ", ".join(_single_quoted(path) for path in paths)
        recurse = ""
    else:
        path_list = "'.'"
        recurse = " -Recurse"
    extra = (" " + " ".join(params)) if params else ""
    script = (
        "Import-Module PSScriptAnalyzer -ErrorAction Stop; "
        f"$issues = @(Invoke-ScriptAnalyzer -Path {path_list}{recurse}{extra}); "
        "$issues; if ($issues.Count -gt 0) { exit 1 }"
    )

    return subprocess.run(
        [executable, "-NoProfile", "-Command", script],
        check=False,
    ).returncode
