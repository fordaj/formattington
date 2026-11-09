"""Dynamic package version computed with versioner at build time.

Setuptools imports this module to resolve the ``version`` attribute
(see ``[tool.setuptools.dynamic]`` in ``pyproject.toml``). The version is
computed by the bundled ``versioner`` binary (``tools/versioner/``) from
git history, matching ``release.bash`` and ``scripts/env.sh``.

Resolution order:
  1. ``FORMATTINGTON_VERSION`` or ``SETUPTOOLS_SCM_PRETEND_VERSION`` env
     (explicit override, e.g. for sdist/wheel builds without git).
  2. ``PKG-INFO`` ``Version:`` line (sdist builds, where git and the
     bundled binary are unavailable).
  3. Bundled ``versioner`` binary (or ``versioner`` on ``PATH``) run
     against the repository root.
  4. Fallback ``0.0.0`` with a stderr warning.

The installed package's ``importlib.metadata`` version (used by
``formattington --version``) is baked at build time; this module is
only imported by setuptools during the build. At runtime in an
installed wheel the bundled binary and git history are absent, so
steps 2-4 do not apply.
"""

import json
import os
import platform
import shutil
import subprocess
import sys
from pathlib import Path

__all__ = ["__version__"]


def _get_platform_dir() -> str:
    """Hyphenated ``<os>-<arch>`` matching ``_tools.get_platform_dir()``."""
    system = platform.system().lower()
    machine = platform.machine().lower()
    systems = {"windows": "windows", "darwin": "macos", "linux": "linux"}
    architectures = {
        "amd64": "x86_64",
        "x86_64": "x86_64",
        "arm64": "arm64",
        "aarch64": "arm64",
    }
    try:
        return f"{systems[system]}-{architectures[machine]}"
    except KeyError as exc:
        raise RuntimeError(f"Unsupported platform: {system}-{machine}") from exc


def _version_key(version: str) -> tuple[int, ...]:
    """Numeric sort key so ``v10`` wins over ``v9`` (mirrors ``_tools``)."""
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


def _repo_root() -> Path:
    """Repository root derived from this file (``src/formattington/_version.py``)."""
    return Path(__file__).resolve().parents[2]


def _bundled_versioner(root: Path) -> str | None:
    """Newest bundled versioner path for this platform, or ``None``."""
    tool_dir = root / "tools" / "versioner"
    if not tool_dir.is_dir():
        return None
    try:
        platform_dir = _get_platform_dir()
    except RuntimeError:
        return None
    is_windows = platform.system() == "Windows"
    exe_names = ["versioner.exe"] if is_windows else ["versioner"]
    candidates = sorted(tool_dir.iterdir(), key=lambda p: _version_key(p.name), reverse=True)
    for version_dir in candidates:
        if not version_dir.is_dir() or not version_dir.name.startswith("v"):
            continue
        for exe_name in exe_names:
            candidate = version_dir / platform_dir / exe_name
            if candidate.is_file():
                return str(candidate)
    return None


def _version_from_pkg_info() -> str | None:
    """Version from ``PKG-INFO`` for sdist builds, or ``None``."""
    candidates = [Path.cwd() / "PKG-INFO", _repo_root() / "PKG-INFO"]
    for pkg_info in candidates:
        if not pkg_info.is_file():
            continue
        try:
            for line in pkg_info.read_text(encoding="utf-8").splitlines()[:50]:
                if line.startswith("Version:"):
                    ver = line.partition(":")[2].strip()
                    if ver:
                        return ver
        except OSError:
            continue
    return None


def _version_from_versioner(binary: str, root: Path) -> str | None:
    """Version from the ``versioner`` binary, or ``None`` on failure."""
    try:
        result = subprocess.run(
            [binary, "--quiet", "--format", "json"],
            cwd=root,
            capture_output=True,
            text=True,
            timeout=30,
            check=False,
        )
    except OSError, subprocess.TimeoutExpired:
        return None
    if result.returncode != 0:
        return None
    try:
        data = json.loads(result.stdout)
    except json.JSONDecodeError, TypeError:
        return None
    ver = data.get("version") if isinstance(data, dict) else None
    if isinstance(ver, str) and ver:
        return ver
    return None


def _resolve_version() -> str:
    """Resolve the package version per the module docstring order."""
    for env_name in ("FORMATTINGTON_VERSION", "SETUPTOOLS_SCM_PRETEND_VERSION"):
        override = os.environ.get(env_name, "").strip()
        if override:
            return override

    pkg_info_version = _version_from_pkg_info()
    if pkg_info_version:
        return pkg_info_version

    root = _repo_root()
    binary = _bundled_versioner(root) or shutil.which("versioner")
    if binary:
        ver = _version_from_versioner(binary, root)
        if ver:
            return ver

    print(
        "warning: versioner unavailable; falling back to 0.0.0",
        file=sys.stderr,
    )
    return "0.0.0"


__version__ = _resolve_version()
