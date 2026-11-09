
import argparse
import concurrent.futures
import json
import os
import sys
from importlib.metadata import PackageNotFoundError, version
from pathlib import Path

from formattington._tools import (
    EXIT_NOT_FOUND,
    EXIT_OK,
    EXIT_USAGE,
    discover_files,
    is_informational,
    iter_chunks,
)
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
from formattington.taplo import FILE_EXTENSIONS as TAPLO_EXTENSIONS
from formattington.taplo import taplo


FORMATTERS = {
    "shfmt": shfmt,
    "clang-format": clang_format,
    "black": black,
    "prettier": prettier,
    "powershell": powershell,
    "taplo": taplo,
}

FORMATTER_EXTENSIONS: dict[str, tuple[str, ...]] = {
    "shfmt": SHFMT_EXTENSIONS,
    "clang-format": CLANG_FORMAT_EXTENSIONS,
    "black": BLACK_EXTENSIONS,
    "prettier": PRETTIER_EXTENSIONS,
    "powershell": POWERSHELL_EXTENSIONS,
    "taplo": TAPLO_EXTENSIONS,
}

# Upper bound on discovered files per formatter invocation, to respect
# operating-system command-line limits on very large repositories.
MAX_FILES_PER_INVOCATION = 256

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
    "toml": "taplo",
}

CONFIG_FILENAME = ".formattington.json"
CONFIG_ENV_VAR = "FORMATTINGTON_CONFIG_PATH"

_EPILOG = (
    "Language aliases (same --check/--dry-run and forwarded-argument "
    "semantics as formatter names): c/cpp/c++ use clang-format; bash "
    "uses shfmt; python/py use black; md/markdown use prettier "
    "(Markdown formatting is provided by Prettier; ``.md`` and "
    "``.markdown`` files are covered); html uses prettier; ps1 uses powershell "
    "(PSScriptAnalyzer linting of PowerShell scripts); toml uses taplo "
    "(TOML formatting via taplo fmt).\n\n"
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
    "powershell, taplo "
    "order. Exit code is the last non-zero code in that order "
    "(missing tool counts as 127, printed to stderr, and does not "
    "prevent remaining formatters from running); 0 only if all succeed.\n\n"
    "Configuration file for 'all' (issue #31): a JSON file with a "
    '\'commands\' array of argv arrays, e.g. {"commands": '
    '[["black", "--check", "src"], ["prettier", "docs/**/*.md"]]}. '
    "Element 0 of each command is a formatter name or language alias; the "
    "rest is forwarded verbatim. Commands run concurrently by default and "
    "sequentially in file order with --series; --check/--dry-run/--verbose "
    "propagate to each command and the exit code is the last non-zero "
    "code in file order (missing tool is 127 per command and does not "
    "stop remaining commands; 0 only if all succeed). An empty "
    "'commands': [] runs nothing and exits 0. Discovery precedence "
    "(highest first): --config PATH (before or after 'all'), then "
    "FORMATTINGTON_CONFIG_PATH, then ./.formattington.json in the current "
    "directory (cwd only, no parent-directory search), then no config "
    "(full-registry 'all' as before). Shell strings, object-form "
    "entries, and bare top-level arrays are rejected; unknown top-level "
    "keys are rejected. Missing/invalid config exits 2 with a "
    "'formattington: ...' message naming the file. Single-tool runs "
    "ignore the config file and environment variable; --config before a "
    "single formatter is a usage error (exit 2). Relative paths are "
    "resolved against the current directory."
)


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
    if is_informational(user_args) or _has_user_file_args(user_args, extensions):
        batches = [user_args]
    else:
        discovered = discover_files(extensions)
        batches = [
            user_args + chunk
            for chunk in iter_chunks(discovered, MAX_FILES_PER_INVOCATION)
        ] or [user_args]

    exit_code = EXIT_OK
    for batch in batches:
        try:
            result = FORMATTERS[name](
                check=check, dry_run=dry_run, args=list(batch), verbose=verbose
            )
        except (FileNotFoundError, RuntimeError) as exc:
            return (name, EXIT_NOT_FOUND, str(exc))
        if result != EXIT_OK:
            exit_code = result
    return (name, exit_code, None)


def _aggregate_exit_code(results: dict[str, int], names: list[str]) -> int:
    """Last non-zero code in ``names`` order wins; 0 only if all succeed."""
    exit_code = EXIT_OK
    for name in names:
        if results[name] != EXIT_OK:
            exit_code = results[name]
    return exit_code


def _aggregate_codes(codes: list[int]) -> int:
    """Last non-zero code in command order wins; 0 only if all succeed."""
    exit_code = EXIT_OK
    for code in codes:
        if code != EXIT_OK:
            exit_code = code
    return exit_code


def _resolve_config_path(
    explicit: str | None,
) -> tuple[Path | None, str | None, str | None]:
    """Resolve the config path by precedence.

    Precedence (highest first): ``explicit`` (``--config``), then
    ``FORMATTINGTON_CONFIG_PATH``, then ``./.formattington.json`` in cwd
    only (no parent search). Returns ``(path, raw, source)`` where
    ``raw`` is the display string and ``source`` is the label used in
    the could-not-find message, or ``(None, None, None)`` when no
    config applies (backwards compatible full-registry ``all``).
    An empty ``FORMATTINGTON_CONFIG_PATH`` counts as unset.
    """
    if explicit is not None:
        return (Path(explicit), explicit, "--config")

    env_value = os.environ.get(CONFIG_ENV_VAR)
    if env_value:
        return (Path(env_value), env_value, CONFIG_ENV_VAR)

    auto = Path.cwd() / CONFIG_FILENAME
    if not auto.exists():
        return (None, None, None)
    return (auto, str(auto), "auto-discovered .formattington.json")


def _read_config_data(
    path: Path,
    *,
    raw: str,
    source: str,
) -> tuple[object | None, str | None]:
    """Read ``path`` and parse JSON.

    Returns ``(data, None)`` on success or ``(None, message)`` where
    ``message`` is the text after ``formattington: `` for the caller
    to print and exit 2. ``raw`` and ``source`` preserve
    byte-identical could-not-find wording (see
    :func:`_resolve_config_path`).
    """
    if not path.is_file():
        return (None, f"Could not find config file '{raw}' (via {source})")

    try:
        text = path.read_text(encoding="utf-8")
    except OSError as exc:
        return (None, f"Could not read config file '{path}': {exc}")

    try:
        data = json.loads(text)
    except json.JSONDecodeError as exc:
        return (None, f"Invalid config file '{path}': invalid JSON: {exc}")

    return (data, None)


def _validate_config(
    data: object,
    path: Path,
) -> tuple[list[list[str]] | None, str | None]:
    """Validate top-level shape and each command entry.

    Returns ``(commands, None)`` when valid (including an empty list)
    or ``(None, message)`` with the same wording as before.
    """
    if not isinstance(data, dict):
        return (
            None,
            f"Invalid config file '{path}': top-level must be an "
            "object with a 'commands' array",
        )

    unknown = sorted(set(data) - {"commands"})
    if unknown:
        return (
            None,
            f"Invalid config file '{path}': unknown key(s) {unknown!r}; "
            "only 'commands' is supported",
        )

    if "commands" not in data:
        return (
            None,
            f"Invalid config file '{path}': missing required 'commands' array",
        )

    commands_raw = data["commands"]
    if not isinstance(commands_raw, list):
        return (
            None,
            f"Invalid config file '{path}': 'commands' must be a list",
        )

    commands: list[list[str]] = []
    for index, entry in enumerate(commands_raw):
        prefix = f"Invalid config file '{path}': commands[{index}]"
        if isinstance(entry, str):
            return (
                None,
                f"{prefix} must be a list of strings, not a shell "
                f"string: {entry!r}",
            )
        if isinstance(entry, dict):
            return (
                None,
                f"{prefix} must be a list of strings, not an object: "
                f"{entry!r}",
            )
        if not isinstance(entry, list):
            return (
                None,
                f"{prefix} must be a list of strings, got "
                f"{type(entry).__name__}: {entry!r}",
            )
        if not entry:
            return (None, f"{prefix} must not be empty")
        if not all(isinstance(item, str) for item in entry):
            return (
                None,
                f"{prefix} must contain only strings: {entry!r}",
            )
        tool = entry[0]
        if tool in FORMATTERS:
            pass
        elif tool in LANGUAGE_ALIASES:
            if LANGUAGE_ALIASES[tool] not in FORMATTERS:
                return (None, f"{prefix}: '{tool}' is not yet supported")
        else:
            return (None, f"{prefix}: unknown formatter or alias '{tool}'")
        commands.append(list(entry))

    return (commands, None)


def _load_config(
    explicit: str | None,
) -> tuple[list[list[str]] | None, str | None]:
    """Resolve, read, and validate the Formattington config file.

    Precedence (highest first): ``explicit`` (``--config``), then
    ``FORMATTINGTON_CONFIG_PATH``, then ``./.formattington.json`` in cwd
    only (no parent search). Returns ``(commands, None)`` when a config
    is in effect, ``(None, None)`` when no config applies (backwards
    compatible full-registry ``all``), or ``(None, message)`` on error
    where ``message`` is the text after ``formattington: `` and the
    caller prints it and exits 2.
    """
    # Resolve
    path, raw, source = _resolve_config_path(explicit)
    if path is None or raw is None or source is None:
        return (None, None)

    # Read
    data, read_error = _read_config_data(path, raw=raw, source=source)
    if read_error is not None:
        return (None, read_error)

    # Validate
    return _validate_config(data, path)


def _run_config_series(
    commands: list[list[str]],
    *,
    check: bool,
    dry_run: bool,
    verbose: bool,
) -> int:
    """Run config commands sequentially in file order."""
    codes: list[int] = []
    for entry in commands:
        target = LANGUAGE_ALIASES.get(entry[0], entry[0])
        _, code, error = _run_formatter(
            target,
            check=check,
            dry_run=dry_run,
            args=list(entry[1:]),
            verbose=verbose,
        )
        if error is not None:
            print(f"formattington: {error}", file=sys.stderr)
        codes.append(code)
    return _aggregate_codes(codes)


def _run_config_parallel(
    commands: list[list[str]],
    *,
    check: bool,
    dry_run: bool,
    verbose: bool,
) -> int:
    """Run config commands concurrently (default for ``all``).

    Results and missing-tool messages are aggregated in file order
    regardless of completion order, mirroring :func:`_run_parallel`.
    Duplicate tool names are supported because results are keyed by
    command index rather than tool name. Empty ``commands`` runs
    nothing and exits 0.
    """
    if not commands:
        return EXIT_OK
    results: dict[int, int] = {}
    errors: dict[int, str] = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=len(commands) or 1) as executor:
        future_by_index: dict[
            concurrent.futures.Future[tuple[str, int, str | None]], int
        ] = {}
        for index, entry in enumerate(commands):
            target = LANGUAGE_ALIASES.get(entry[0], entry[0])
            future_by_index[
                executor.submit(
                    _run_formatter,
                    target,
                    check=check,
                    dry_run=dry_run,
                    args=list(entry[1:]),
                    verbose=verbose,
                )
            ] = index
        for future in future_by_index:
            index = future_by_index[future]
            _, code, error = future.result()
            results[index] = code
            if error is not None:
                errors[index] = error
    for index in sorted(errors):
        print(f"formattington: {errors[index]}", file=sys.stderr)
    return _aggregate_codes([results[index] for index in sorted(results)])


def _find_formatter_index(argv: list[str]) -> int | None:
    """First formatter/alias/``all`` index, skipping ``--config`` values."""
    selectors = {*FORMATTERS, *LANGUAGE_ALIASES, "all"}
    index = 0
    while index < len(argv):
        token = argv[index]
        if token == "--config":
            index += 2
            continue
        if token.startswith("--config="):
            index += 1
            continue
        if token in selectors:
            return index
        index += 1
    return None


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


def _build_parser() -> argparse.ArgumentParser:
    """Build Formattington's shared ``ArgumentParser``.

    Both the no-formatter and with-formatter paths use this single
    parser so shared flags (``--check``/``--dry-run``, ``--series``,
    ``--verbose``, ``--config``, ``--version``) are defined once and
    cannot drift.
    """
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
    parser.add_argument(
        "--config",
        metavar="PATH",
        default=None,
        help=(
            "with 'all', run commands from this JSON config file "
            "(also via FORMATTINGTON_CONFIG_PATH or ./.formattington.json)"
        ),
    )
    return parser


def _parse_all_trailing(
    formatter_args: list[str],
    parser: argparse.ArgumentParser,
) -> tuple[bool, str | None]:
    """Parse trailing options after ``all``.

    ``--series`` may appear after ``all`` (``formattington all --series``)
    and ``--config PATH`` is honored after ``all`` (including the
    ``--config=PATH`` spelling). Any other trailing argument is a usage
    error. Returns ``(series_seen, after_config)`` where ``series_seen``
    records whether ``--series`` appeared after ``all`` and
    ``after_config`` is the trailing ``--config`` value (or ``None``).
    Errors exit via ``parser.error`` (exit 2).
    """
    series_seen = False
    after_config: str | None = None
    remaining: list[str] = []
    pos = 0
    while pos < len(formatter_args):
        arg = formatter_args[pos]
        if arg == "--series":
            series_seen = True
            pos += 1
        elif arg == "--config":
            if pos + 1 >= len(formatter_args):
                parser.error("--config requires a path")
            if after_config is not None:
                parser.error("--config may only be given once")
            after_config = formatter_args[pos + 1]
            pos += 2
        elif arg.startswith("--config="):
            if after_config is not None:
                parser.error("--config may only be given once")
            after_config = arg[len("--config=") :]
            pos += 1
        else:
            remaining.append(arg)
            pos += 1
    if remaining:
        parser.error("extra arguments are not supported with 'all'")
    return (series_seen, after_config)


def main() -> int:
    argv = sys.argv[1:]

    # Formattington owns global options before the formatter name.
    # ``--config`` values are skipped during the scan so a path that
    # happens to equal a formatter name is not mistaken for the selector.
    formatter_index = _find_formatter_index(argv)

    # Only Formattington's arguments count here.
    formattington_args = (
        argv if formatter_index is None else argv[:formatter_index]
    )

    if "--version" in formattington_args:
        print(formattington_version())
        return EXIT_OK

    # Global flags: one shared parser for every path.
    parser = _build_parser()

    if formatter_index is None:
        parser.parse_args(argv)
        return EXIT_OK

    formatter_name = argv[formatter_index]
    formatter_args = argv[formatter_index + 1:]
    args = parser.parse_args(formattington_args)

    series = args.series
    explicit_config: str | None = args.config
    verbose = args.verbose

    if formatter_name == "all":
        # ``formattington all --series`` is the documented spelling: --series
        # may appear after 'all' (and, for consistency with --check/--dry-run,
        # also before 'all' via the global parser above). ``--config PATH``
        # is likewise honored after 'all'. Any other trailing argument is
        # still an error.
        series_after_all, after_config = _parse_all_trailing(
            formatter_args, parser
        )
        if series_after_all:
            series = True
        if explicit_config is not None and after_config is not None:
            parser.error("--config may only be given once")
        if explicit_config is None:
            explicit_config = after_config
        formatter_args = []

        commands, config_error = _load_config(explicit_config)
        if config_error is not None:
            print(f"formattington: {config_error}", file=sys.stderr)
            return EXIT_USAGE
        if commands is not None:
            if series:
                return _run_config_series(
                    commands,
                    check=args.check,
                    dry_run=args.dry_run,
                    verbose=verbose,
                )
            return _run_config_parallel(
                commands,
                check=args.check,
                dry_run=args.dry_run,
                verbose=verbose,
            )
        names = list(FORMATTERS)
    else:
        # Config only changes what 'all' means: an explicit --config
        # before a single formatter is a usage error, while the env
        # variable and ./.formattington.json are ignored here.
        if explicit_config is not None:
            parser.error("--config is only supported with 'all'")
        target = LANGUAGE_ALIASES.get(formatter_name, formatter_name)
        if target not in FORMATTERS:
            # A documented alias whose backend has not landed yet must
            # fail loudly, never silently succeed.
            parser.error(f"'{formatter_name}' is not yet supported")
        names = [target]

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
