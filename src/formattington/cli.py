
import argparse
import sys
from importlib.metadata import PackageNotFoundError, version

from formattington.clang_format import clang_format
from formattington.shfmt import shfmt


FORMATTERS = {
    "shfmt": shfmt,
    "clang-format": clang_format,
}


def main() -> int:
    argv = sys.argv[1:]

    # Formattington owns global options before the formatter name.
    formatter_index = next(
        (i for i, arg in enumerate(argv) if arg in {*FORMATTERS, "all"}),
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
        (i for i, arg in enumerate(argv) if arg in {*FORMATTERS, "all"}),
        None,
    )

    if formatter_index is None:
        parser = argparse.ArgumentParser(
            prog="formattington",
            description="One formatter interface for your entire project.",
        )
        parser.add_argument(
            "--version",
            action="version",
            version=formattington_version(),
        )
        parser.add_argument("--check", action="store_true")
        parser.add_argument("--dry-run", action="store_true")
        parser.parse_args(argv)
        return 0

    formattington_args = argv[:formatter_index]
    formatter_name = argv[formatter_index]
    formatter_args = argv[formatter_index + 1:]

    parser = argparse.ArgumentParser(
        prog="formattington",
        description="One formatter interface for your entire project.",
    )
    parser.add_argument(
        "--version",
        action="version",
        version=formattington_version(),
    )
    modes = parser.add_mutually_exclusive_group()
    modes.add_argument("--check", action="store_true")
    modes.add_argument("--dry-run", action="store_true")
    args = parser.parse_args(formattington_args)

    if formatter_name == "all":
        if formatter_args:
            parser.error("extra arguments are not supported with 'all'")
        names = list(FORMATTERS)
    else:
        names = [formatter_name]

    exit_code = 0

    for name in names:
        try:
            result = FORMATTERS[name](
                check=args.check,
                dry_run=args.dry_run,
                args=formatter_args,
            )
        except (FileNotFoundError, RuntimeError) as exc:
            print(f"formattington: {exc}", file=sys.stderr)
            exit_code = 127
            continue
        if result != 0:
            exit_code = result

    return exit_code


def formattington_version() -> str:
    try:
        return version("formattington")
    except PackageNotFoundError:
        return "unknown (package metadata unavailable)"


if __name__ == "__main__":
    sys.exit(main())
