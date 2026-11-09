import os
import platform
import shutil
from collections.abc import Iterator, Sequence
from pathlib import Path
import subprocess
import sys

# Bundled root, derived from ``__file__``. ``find_tool()`` resolves
# ``tools/<name>/v<version>/<platform>/<executable>`` under ``TOOLS_DIR``.
PROJECT_ROOT = Path(__file__).resolve().parents[2]
TOOLS_DIR = PROJECT_ROOT / "tools"

# Directory names discovery never descends into. Besides version-control
# metadata, these are the usual places generated, vendored, minified, or
# dependency files live (see AGENTS.md). Hidden directories (starting
# with ".") are also skipped, along with the bundled-tools directory,
# which holds pinned binaries rather than user source.
IGNORED_DIR_NAMES: frozenset[str] = frozenset(
    {
        "__pycache__",
        "node_modules",
        "vendor",
        "venv",
        "dist",
        "build",
        "tools",
    }
)


# Bare help/version tokens that pass through without formatting flags.
INFORMATIONAL_ARGS: frozenset[str] = frozenset(
    {"--help", "-help", "--version", "-version", "-h", "-V"}
)

# Exit codes (see README "Exit codes"): named policy constants so
# success / formatting-required / usage-error / tool-not-found
# decisions read by purpose instead of bare literals.
EXIT_OK = 0
EXIT_FORMAT_REQUIRED = 1
EXIT_USAGE = 2
EXIT_NOT_FOUND = 127


def is_informational(args: list[str]) -> bool:
    """Whether ``args`` are bare help/version passthroughs (no formatting flags)."""
    return any(arg in INFORMATIONAL_ARGS for arg in args)


def ensure_valid_mode(*, check: bool, dry_run: bool) -> None:
    """Raise ``ValueError`` if ``check`` and ``dry_run`` are both set."""
    if check and dry_run:
        raise ValueError("check and dry_run cannot both be enabled")


def _is_ignored_dirname(name: str) -> bool:
    """Whether discovery should prune a directory by name."""
    return name.startswith(".") or name in IGNORED_DIR_NAMES


def discover_files(
    extensions: Sequence[str],
    root: Path | None = None,
) -> list[str]:
    """Discover matching files under ``root`` (default: current directory).

    ``extensions`` are dotted suffixes such as ``".sh"``; matching is
    case-insensitive. Hidden directories, :data:`IGNORED_DIR_NAMES`, and
    anything under them are skipped. Results are sorted and returned as
    strings relative to ``root`` so they can be appended to a formatter
    argv directly (paths with spaces stay single entries).
    """
    base = Path(root) if root is not None else Path.cwd()
    if not base.is_dir():
        return []
    wanted = {ext.lower() for ext in extensions}
    found: list[str] = []
    for dirpath, dirnames, filenames in os.walk(base, topdown=True):
        dirnames[:] = [d for d in dirnames if not _is_ignored_dirname(d)]
        for filename in filenames:
            if Path(filename).suffix.lower() in wanted:
                absolute = Path(dirpath) / filename
                found.append(str(absolute.relative_to(base)))
    return sorted(found)


def iter_chunks(items: Sequence[str], size: int) -> Iterator[list[str]]:
    """Split ``items`` into consecutive chunks of at most ``size`` entries.

    Used to respect operating-system command-line limits when passing
    many discovered files to a single formatter invocation.
    """
    if size <= 0:
        raise ValueError("chunk size must be positive")
    for start in range(0, len(items), size):
        yield list(items[start : start + size])

def _formatting_required(stdout: str) -> bool:
    """Whether captured ``shfmt -l`` output means files need formatting."""
    return bool(stdout.strip())


def run_tool(
    executable: str,
    arguments: list[str],
    *,
    capture_output: bool = False,
) -> int:
    """Run a formatter and return its exit code.

    When ``capture_output`` is set, stdout/stderr are printed and
    non-empty ``shfmt -l`` stdout is reported as
    ``EXIT_FORMAT_REQUIRED`` (formatting changes required).
    """
    result = subprocess.run(
        [executable, *arguments],
        capture_output=capture_output,
        text=True,
        check=False,
    )

    if capture_output:
        if result.stdout:
            print(result.stdout, end="")
        if result.stderr:
            print(result.stderr, end="", file=sys.stderr)

        if result.returncode != 0:
            return result.returncode

        # For shfmt -l, output means files need formatting.
        if _formatting_required(result.stdout):
            print(
                "formattington: formatting changes required.",
                file=sys.stderr,
            )
            return EXIT_FORMAT_REQUIRED

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
    components instead so the newest bundle wins. Non-numeric suffixes
    stop the digit scan; a component with no digits counts as 0.
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


def _bundled_versions(tool_dir: Path) -> list[str]:
    """Version labels (without ``v``) sorted newest-first, or ``[]`` if no dir."""
    if not tool_dir.is_dir():
        return []
    # Prefer the newest version by numeric version key.
    return sorted(
        (
            path.name.removeprefix("v")
            for path in tool_dir.iterdir()
            if path.is_dir()
        ),
        key=_version_key,
        reverse=True,
    )


def _find_bundled_executable(
    name: str,
    version: str | None,
    *,
    executable: str,
) -> str | None:
    """Return the bundled path if found, else ``None`` (no PATH lookup, no raise)."""
    tool_dir = TOOLS_DIR / name
    if version:
        versions = [version.removeprefix("v")]
    else:
        versions = _bundled_versions(tool_dir)

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
                return str(bundled)
    return None


def _find_on_path(name: str, *, is_windows: bool) -> str | None:
    """Return the PATH-resolved tool (including ``.exe`` fallback on Windows), else ``None``."""
    system_tool = shutil.which(name)
    if not system_tool and is_windows:
        system_tool = shutil.which(f"{name}.exe")
    return system_tool


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

    bundled = _find_bundled_executable(name, version, executable=executable)
    if bundled is not None:
        if show_verbose:
            print(
                f"formattington: {name} resolved to {bundled} (bundled)",
                file=sys.stderr,
            )
        return bundled

    system_tool = _find_on_path(name, is_windows=is_windows)
    if system_tool is not None:
        if show_verbose:
            print(
                f"formattington: {name} resolved to {system_tool} (PATH)",
                file=sys.stderr,
            )
        return system_tool

    tool_dir = TOOLS_DIR / name
    platform_dir = get_platform_dir()
    expected = tool_dir / "<version>" / platform_dir / executable
    raise FileNotFoundError(
        f"Could not find {name!r}. "
        f"Expected a bundled executable under {expected} "
        "or an installation on PATH."
    )