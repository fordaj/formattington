
import argparse
import concurrent.futures
import sys
from importlib.metadata import PackageNotFoundError, version

from formattington.clang_format import clang_format
from formattington.format_python import black
from formattington.prettier import prettier
from formattington.shfmt import shfmt


FORMATTERS = {
    "shfmt": shfmt,
    "clang-format": clang_format,
    "black": black,
    "prettier": prettier,
}

LANGUAGE_ALIASES = {
    "python": "black",
    "py": "black",
    "md": "prettier",
    "markdown": "prettier",
}

_EPILOG = (
    "Language aliases: python/py use black; md/markdown use prettier "
    "(Markdown formatting is provided by Prettier; ``.md`` and "
    "``.markdown`` files are covered).\n\n"
    "Bundled tools live under tools/<name>/v<version>/"
    "<platform>/; use --verbose (or set "
    "FORMATTINGTON_VERBOSE=1) to show the resolved executable.\n\n"
    "With 'all', formatters run concurrently by default; "
    "use --series (before or after 'all', e.g. "
    "'formattington --series all' or 'formattington all --series') "
    "to run them sequentially in shfmt, clang-format, black, prettier "
    "order. Exit code is the last non-zero code in that order "
    "(missing tool counts as 127, printed to stderr, and does not "
    "prevent remaining formatters from running); 0 only if all succeed."
)


def _run_formatter(
    name: str,
    *,
    check: bool,
    dry_run: bool,
    args: list[str],
    verbose: bool,
) -> tuple[str, int, str | None]:
    """Run one formatter, translating missing-tool errors to 127.

    Returns ``(name, returncode, error_message_or_None)`` so callers can
    report errors and aggregate exit codes in registration order regardless
    of completion order (important for the concurrent default).
    """
    try:
        call_kwargs: dict[str, object] = {
            "check": check,
            "dry_run": dry_run,
            "args": list(args),
        }
        if verbose:
            call_kwargs["verbose"] = True
        result = FORMATTERS[name](**call_kwargs)  # type: ignore[arg-type]
    except (FileNotFoundError, RuntimeError) as exc:
        return (name, 127, str(exc))
    return (name, result, None)


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
        names = [LANGUAGE_ALIASES.get(formatter_name, formatter_name)]

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
