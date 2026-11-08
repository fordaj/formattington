
import argparse
import platform
import shutil
import subprocess
import sys
from pathlib import Path


PROJECT_ROOT = Path(__file__).resolve().parents[2]


def get_platform_dir() -> str:
    system = platform.system().lower()
    machine = platform.machine().lower()

    systems = {
        "windows": "windows",
        "darwin": "macos",
        "linux": "linux",
    }
    architectures = {
        "amd64": "x86_64",
        "x86_64": "x86_64",
        "arm64": "arm64",
        "aarch64": "arm64",
    }

    try:
        return (
            f"{systems[system]}-"
            f"{architectures[machine]}"
        )
    except KeyError as exc:
        raise RuntimeError(
            f"Unsupported platform: {system}-{machine}"
        ) from exc


def find_tool(name: str) -> str:
    """Prefer a bundled tool, falling back to PATH."""
    executable = (
        f"{name}.exe"
        if platform.system() == "Windows"
        else name
    )

    bundled = (
        PROJECT_ROOT
        / "tools"
        / get_platform_dir()
        / executable
    )

    if bundled.is_file():
        return str(bundled)

    system_tool = shutil.which(name)
    if system_tool:
        return system_tool

    raise FileNotFoundError(
        f"Could not find {name!r}. "
        f"Expected a bundled executable at {bundled} "
        "or an installation on PATH."
    )


def run_shfmt(mode: str | None = None) -> int:
    try:
        executable = find_tool("shfmt")
    except (FileNotFoundError, RuntimeError) as exc:
        print(f"formattington: {exc}", file=sys.stderr)
        return 127

    if mode == "check":
        # List files that would change without modifying them.
        result = subprocess.run(
            [executable, "-l", "."],
            text=True,
            capture_output=True,
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
                "shfmt: formatting changes required.",
                file=sys.stderr,
            )
            return 1

        print("shfmt: all files are formatted.")
        return 0

    if mode == "dry-run":
        # Print a diff, but leave files untouched.
        result = subprocess.run(
            [executable, "-d", "."],
            text=True,
            capture_output=True,
            check=False,
        )

        if result.stdout:
            print(result.stdout, end="")
        if result.stderr:
            print(result.stderr, end="", file=sys.stderr)

        # A diff is informational in dry-run mode.
        if result.returncode != 0 and result.stderr:
            return result.returncode

        if not result.stdout.strip():
            print("shfmt: no changes needed.")

        return 0

    # Default mode: format files in place.
    return subprocess.run(
        [executable, "-w", "."],
        check=False,
    ).returncode


FORMATTERS = {
    "shfmt": run_shfmt,
}


def run_formatter(name: str, mode: str | None) -> int:
    formatter = FORMATTERS.get(name)

    if formatter is None:
        print(
            f"formattington: unknown formatter {name!r}",
            file=sys.stderr,
        )
        return 2

    return formatter(mode)


def main() -> int:
    parser = argparse.ArgumentParser(
        prog="formattington",
        description="One formatter interface for your entire project.",
    )

    modes = parser.add_mutually_exclusive_group()
    modes.add_argument(
        "--check",
        action="store_true",
        help="check formatting without modifying files",
    )
    modes.add_argument(
        "--dry-run",
        action="store_true",
        help="preview formatting changes without modifying files",
    )

    parser.add_argument(
        "formatter",
        choices=["shfmt", "all"],
        help="formatter to run, or all registered formatters",
    )

    args = parser.parse_args()

    mode = (
        "check" if args.check
        else "dry-run" if args.dry_run
        else None
    )

    names = list(FORMATTERS) if args.formatter == "all" else [
        args.formatter
    ]

    exit_code = 0

    for name in names:
        result = run_formatter(name, mode)
        if result != 0:
            exit_code = result

    return exit_code
