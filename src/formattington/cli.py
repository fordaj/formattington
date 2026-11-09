
import argparse
import concurrent.futures
import sys
from importlib.metadata import PackageNotFoundError, version
from pathlib import Path

from formattington._tools import discover_files, iter_chunks
from formattington.clang_format import FILE_EXTENSIONS as CLANG_FORMAT_EXTENSIONS
from formattington.clang_format import clang_format
from formattington.format_python import FILE_EXTENSIONS as BLACK_EXTENSIONS
from formattington.format_python import black
from formattington.prettier import FILE_EXTENSIONS as PRETTIER_EXTENSIONS
from formattington.powershell import FILE_EXTENSIONS as POWERSHELL_EXTENSIONS
from formattington.powershell import powershell
from formattington.prettier import prettier
from formattington.shfmt import FILE_EXTENSIONS as SHFMT_EXTENSIONS
from formattington.shfmt import shfmt


FORMATTERS = {
    "shfmt": shfmt,
    "clang-format": clang_format,
    "black": black,
    "prettier": prettier,
    "powershell": powershell,
}

FORMATTER_EXTENSIONS: dict[str, tuple[str, ...]] = {
    "shfmt": SHFMT_EXTENSIONS,
    "clang-format": CLANG_FORMAT_EXTENSIONS,
    "black": BLACK_EXTENSIONS,
    "prettier": PRETTIER_EXTENSIONS,
    "powershell": POWERSHELL_EXTENSIONS,
}

# Upper bound on discovered files per formatter invocation, to respect
# operating-system command-line limits on very large repositories.
MAX_FILES_PER_INVOCATION = 256

_INFORMATIONAL_ARGS = frozenset(
    {"--help", "-help", "--version", "-version", "-h", "-V"}
)

LANGUAGE_ALIASES = {
    "c": "clang-format",
    "cpp": "clang-format",
    "c++": "clang-format",
    "bash": "shfmt",
    "python": "black",
    "py": "black",
    "md": "prettier",
    "markdown": "prettier",
    "html": "prettier",
    "ps1": "powershell",
}

_EPILOG = (
    "Language aliases (same --check/--dry-run and forwarded-argument "
    "semantics as formatter names): c/cpp/c++ use clang-format; bash "
    "uses shfmt; python/py use black; md/markdown use prettier "
    "(Markdown formatting is provided by Prettier; ``.md`` and "
    "``.markdown`` files are covered); html uses prettier; ps1 uses powershell "
    "(PSScriptAnalyzer linting of PowerShell scripts).\n\n"
    "With no file arguments, formattington discovers each formatter's "
    "matching files recursively from the current directory (skipping "
    "hidden directories and __pycache__, node_modules, vendor, venv, "
    "dist, build, and tools/) and formats them in batches.\n\n"
    "Bundled tools live under tools/<name>/v<version>/"
    "<platform>/; use --verbose (or set "
    "FORMATTINGTON_VERBOSE=1) to show the resolved executable.\n\n"
    "With 'all', formatters run concurrently by default; "
    "use --series (before or after 'all', e.g. "
    "'formattington --series all' or 'formattington all --series') "
    "to run them sequentially in shfmt, clang-format, black, prettier, "
    "powershell "
    "order. Exit code is the last non-zero code in that order "
    "(missing tool counts as 127, printed to stderr, and does not "
    "prevent remaining formatters from running); 0 only if all succeed."
)


def _is_informational(args: list[str]) -> bool:
    """Whether ``args`` are bare help/version passthroughs (no discovery)."""
    return any(arg in _INFORMATIONAL_ARGS for arg in args)


def _has_user_file_args(args: list[str], extensions: tuple[str, ...]) -> bool:
    """Whether the user supplied explicit file arguments.

    A non-flag argument counts as an explicit file when it exists on
    disk or ends with one of the formatter's extensions.
    """
    wanted = {ext.lower() for ext in extensions}
    for arg in args:
        if arg.startswith("-"):
            continue
        if Path(arg).exists():
            return True
        if Path(arg).suffix.lower() in wanted:
            return True
    return False


def _run_formatter(
    name: str,
    *,
    check: bool,
    dry_run: bool,
    args: list[str],
    verbose: bool,
) -> tuple[str, int, str | None]:
    """Run one formatter, translating missing-tool errors to 127.

    When the user passes no explicit file arguments (and the args are
    not informational help/version passthroughs), matching files are
    discovered recursively and appended to the invocation, in batches
    of at most ``MAX_FILES_PER_INVOCATION`` files.

    Returns ``(name, returncode, error_message_or_None)`` so callers can
    report errors and aggregate exit codes in registration order regardless
    of completion order (important for the concurrent default).
    """
    extensions = FORMATTER_EXTENSIONS[name]
    user_args = list(args)
    if _is_informational(user_args) or _has_user_file_args(user_args, extensions):
        batches = [user_args]
    else:
        discovered = discover_files(extensions)
        batches = [
            user_args + chunk
            for chunk in iter_chunks(discovered, MAX_FILES_PER_INVOCATION)
        ] or [user_args]

    exit_code = 0
    for batch in batches:
        try:
            call_kwargs: dict[str, object] = {
                "check": check,
                "dry_run": dry_run,
                "args": list(batch),
            }
            if verbose:
                call_kwargs["verbose"] = True
            result = FORMATTERS[name](**call_kwargs)  # type: ignore[arg-type]
        except (FileNotFoundError, RuntimeError) as exc:
            return (name, 127, str(exc))
        if result != 0:
            exit_code = result
    return (name, exit_code, None)


def _aggregate_exit_code(results: dict[str, int], names: list[str]) -> int:
    """Last non-zero code in ``names`` order wins; 0 only if all succeed."""
    exit_code = 0
    for name in names:
        if results[name] != 0:
            exit_code = results[name]
    return exit_code


def _run_series(
    names: list[str],
    *,
    check: bool,
    dry_run: bool,
    args: list[str],
    verbose: bool,
) -> int:
    """Run formatters one at a time in registration order."""
    results: dict[str, int] = {}
    for name in names:
        _, code, error = _run_formatter(
            name,
            check=check,
            dry_run=dry_run,
            args=args,
            verbose=verbose,
        )
        if error is not None:
            print(f"formattington: {error}", file=sys.stderr)
        results[name] = code
    return _aggregate_exit_code(results, names)


def _run_parallel(
    names: list[str],
    *,
    check: bool,
    dry_run: bool,
    args: list[str],
    verbose: bool,
) -> int:
    """Run formatters concurrently with a thread pool.

    Results are aggregated in registration order (not completion order) so
    the exit code is deterministic; missing-tool messages are printed in
    registration order after all formatters have run. Formatters that
    write to stdout/stderr may interleave in this mode — use ``--series``
    for ordered output.
    """
    results: dict[str, int] = {}
    errors: dict[str, str] = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=len(names) or 1) as executor:
        future_by_name = {
            executor.submit(
                _run_formatter,
                name,
                check=check,
                dry_run=dry_run,
                args=args,
                verbose=verbose,
            ): name
            for name in names
        }
        # Collect in submission (registration) order; workers still overlap
        # because all futures were submitted before any result is awaited.
        for future in future_by_name:
            name, code, error = future.result()
            results[name] = code
            if error is not None:
                errors[name] = error
    for name in names:
        if name in errors:
            print(f"formattington: {errors[name]}", file=sys.stderr)
    return _aggregate_exit_code(results, names)


def main() -> int:
    argv = sys.argv[1:]

    # Formattington owns global options before the formatter name.
    formatter_index = next(
        (i for i, arg in enumerate(argv) if arg in {*FORMATTERS, *LANGUAGE_ALIASES, "all"}),
        None,
    )

    # Only Formattington's arguments count here.
    formattington_args = (
        argv if formatter_index is None else argv[:formatter_index]
    )

    if "--version" in formattington_args:
        print(formattington_version())
        return 0

    formatter_index = next(
        (i for i, arg in enumerate(argv) if arg in {*FORMATTERS, *LANGUAGE_ALIASES, "all"}),
        None,
    )

    if formatter_index is None:
        parser = argparse.ArgumentParser(
            prog="formattington",
            description="One formatter interface for your entire project.",
            epilog=_EPILOG,
        )
        parser.add_argument(
            "--version",
            action="version",
            version=formattington_version(),
        )
        parser.add_argument("--check", action="store_true")
        parser.add_argument("--dry-run", action="store_true")
        parser.add_argument(
            "--series",
            action="store_true",
            help=(
                "with 'all', run formatters sequentially instead of "
                "concurrently (no effect without 'all')"
            ),
        )
        parser.add_argument(
            "--verbose",
            action="store_true",
            help=(
                "show which formatter executable was resolved "
                "(also via FORMATTINGTON_VERBOSE=1)"
            ),
        )
        parser.parse_args(argv)
        return 0

    formattington_args = argv[:formatter_index]
    formatter_name = argv[formatter_index]
    formatter_args = argv[formatter_index + 1:]

    parser = argparse.ArgumentParser(
        prog="formattington",
        description="One formatter interface for your entire project.",
        epilog=_EPILOG,
    )
    parser.add_argument(
        "--version",
        action="version",
        version=formattington_version(),
    )
    modes = parser.add_mutually_exclusive_group()
    modes.add_argument("--check", action="store_true")
    modes.add_argument("--dry-run", action="store_true")
    parser.add_argument(
        "--series",
        action="store_true",
        help=(
            "with 'all', run formatters sequentially instead of "
            "concurrently (no effect on single-formatter runs)"
        ),
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help=(
            "show which formatter executable was resolved "
            "(also via FORMATTINGTON_VERBOSE=1)"
        ),
    )
    args = parser.parse_args(formattington_args)

    series = getattr(args, "series", False)

    if formatter_name == "all":
        # ``formattington all --series`` is the documented spelling: --series
        # may appear after 'all' (and, for consistency with --check/--dry-run,
        # also before 'all' via the global parser above). Any other trailing
        # argument is still an error.
        remaining: list[str] = []
        for arg in formatter_args:
            if arg == "--series":
                series = True
            else:
                remaining.append(arg)
        if remaining:
            parser.error("extra arguments are not supported with 'all'")
        formatter_args = []
        names = list(FORMATTERS)
    else:
        target = LANGUAGE_ALIASES.get(formatter_name, formatter_name)
        if target not in FORMATTERS:
            # A documented alias whose backend has not landed yet must
            # fail loudly, never silently succeed.
            parser.error(f"'{formatter_name}' is not yet supported")
        names = [target]

    verbose = getattr(args, "verbose", False)

    if formatter_name != "all" or len(names) == 1:
        # Single-formatter runs have no concurrency to order; run directly
        # (equivalently: --series is a no-op here).
        _, code, error = _run_formatter(
            names[0],
            check=args.check,
            dry_run=args.dry_run,
            args=formatter_args,
            verbose=verbose,
        )
        if error is not None:
            print(f"formattington: {error}", file=sys.stderr)
        return code

    if series:
        return _run_series(
            names,
            check=args.check,
            dry_run=args.dry_run,
            args=formatter_args,
            verbose=verbose,
        )
    return _run_parallel(
        names,
        check=args.check,
        dry_run=args.dry_run,
        args=formatter_args,
        verbose=verbose,
    )


def formattington_version() -> str:
    try:
        return version("formattington")
    except PackageNotFoundError:
        return "unknown (package metadata unavailable)"


if __name__ == "__main__":
    sys.exit(main())
