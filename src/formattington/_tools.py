import os
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


def _is_verbose_env() -> bool:
    """Whether FORMATTINGTON_VERBOSE enables verbose resolution hints."""
    return os.environ.get(
        "FORMATTINGTON_VERBOSE", ""
    ).strip().lower() in {"1", "true", "yes", "on"}


def _version_key(version: str) -> tuple[int, ...]:
    """Parse a version string into a comparable numeric tuple.

    Sorting lexicographically fails at v9 vs v10; compare numeric
    components instead so the newest bundle wins.
    """
    text = version.strip().removeprefix("v").lower()
    key: list[int] = []
    for part in text.replace("-", ".").split("."):
        digits = ""
        for char in part:
            if char.isdigit():
                digits += char
            else:
                break
        key.append(int(digits) if digits else 0)
    return tuple(key)


def _platform_candidates(platform_dir: str) -> list[str]:
    """Hyphenated platform dir first, then the legacy underscored form.

    Bundled dirs use the hyphenated convention (for example
    ``linux-x86_64``). Underscored names (for example ``linux_x86_64``)
    are a migration fallback so older layouts still resolve.
    """
    if "-" in platform_dir:
        system, arch = platform_dir.split("-", 1)
        legacy = f"{system}_{arch}"
        return [platform_dir, legacy]
    return [platform_dir]


def find_tool(
    name: str, version: str | None = None, *, verbose: bool = False
) -> str:
    """Find a bundled formatter, falling back to PATH.

    Bundled executables live under
    ``tools/<name>/v<version>/<platform>/<executable>`` where
    ``<platform>`` is the hyphenated ``get_platform_dir()`` value.
    When verbose is enabled explicitly or via ``FORMATTINGTON_VERBOSE``,
    the resolved executable is printed to stderr.
    """
    show_verbose = verbose or _is_verbose_env()
    is_windows = platform.system() == "Windows"
    executable = f"{name}.exe" if is_windows else name

    tool_dir = TOOLS_DIR / name

    if version:
        versions = [version.removeprefix("v")]
    else:
        # Prefer the newest version by numeric version key.
        versions = sorted(
            (
                path.name.removeprefix("v")
                for path in tool_dir.iterdir()
                if path.is_dir()
            ),
            key=_version_key,
            reverse=True,
        ) if tool_dir.is_dir() else []

    platform_dir = get_platform_dir()

    for candidate_version in versions:
        for candidate_platform in _platform_candidates(platform_dir):
            bundled = (
                tool_dir
                / f"v{candidate_version}"
                / candidate_platform
                / executable
            )

            if bundled.is_file():
                if show_verbose:
                    print(
                        f"formattington: {name} resolved to {bundled} (bundled)",
                        file=sys.stderr,
                    )
                return str(bundled)

    system_tool = shutil.which(name)
    if not system_tool and is_windows:
        system_tool = shutil.which(f"{name}.exe")
    if system_tool:
        if show_verbose:
            print(
                f"formattington: {name} resolved to {system_tool} (PATH)",
                file=sys.stderr,
            )
        return system_tool

    expected = tool_dir / "<version>" / platform_dir / executable
    raise FileNotFoundError(
        f"Could not find {name!r}. "
        f"Expected a bundled executable under {expected} "
        "or an installation on PATH."
    )