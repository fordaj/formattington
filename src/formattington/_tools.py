import platform
import shutil
from pathlib import Path
import subprocess
import sys

PROJECT_ROOT = Path(__file__).resolve().parents[2]
TOOLS_DIR = PROJECT_ROOT / "tools"

def run_tool(
    executable: str,
    arguments: list[str],
    *,
    check_output: bool = False,
) -> int:
    """Run a formatter and return its exit code."""
    result = subprocess.run(
        [executable, *arguments],
        capture_output=check_output,
        text=True,
        check=False,
    )

    if check_output:
        if result.stdout:
            print(result.stdout, end="")
        if result.stderr:
            print(result.stderr, end="", file=sys.stderr)

        if result.returncode != 0:
            return result.returncode

        # For shfmt -l, output means files need formatting.
        if result.stdout.strip():
            print(
                "formattington: formatting changes required.",
                file=sys.stderr,
            )
            return 1

    return result.returncode



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
        return f"{systems[system]}-{architectures[machine]}"
    except KeyError as exc:
        raise RuntimeError(
            f"Unsupported platform: {system}-{machine}"
        ) from exc


def find_tool(name: str, version: str | None = None) -> str:
    """Find a bundled formatter, falling back to PATH."""
    executable = (
        f"{name}.exe"
        if platform.system() == "Windows"
        else name
    )

    tool_dir = TOOLS_DIR / name

    if version:
        versions = [version.removeprefix("v")]
    else:
        # Prefer the newest version directory by name.
        versions = sorted(
            (
                path.name.removeprefix("v")
                for path in tool_dir.iterdir()
                if path.is_dir()
            ),
            reverse=True,
        ) if tool_dir.is_dir() else []

    platform_dir = get_platform_dir()

    for candidate_version in versions:
        bundled = (
            tool_dir
            / f"v{candidate_version}"
            / platform_dir
            / executable
        )

        if bundled.is_file():
            return str(bundled)

    system_tool = shutil.which(name)
    if system_tool:
        return system_tool

    expected = tool_dir / "<version>" / platform_dir / executable
    raise FileNotFoundError(
        f"Could not find {name!r}. "
        f"Expected a bundled executable under {expected} "
        "or an installation on PATH."
    )