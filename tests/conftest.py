"""Shared fixtures and helpers for Formattington tests."""

from __future__ import annotations

import os
import stat
import sys
from pathlib import Path
from unittest.mock import MagicMock

import pytest

import formattington._tools as tools
import formattington.cli as cli


def make_fake_executable(directory: Path, name: str) -> Path:
    """Create a fake executable that records argv and exits 0.

    The script appends each argument (one per line) to the file named by
    the FAKE_ARGV_FILE environment variable, creating parent directories
    as needed. It is a POSIX shell script, so callers should skip on Windows
    or mock a non-Windows platform when invoking it.
    """
    directory.mkdir(parents=True, exist_ok=True)
    script = directory / name
    script.write_text(
        "#!/bin/sh\n"
        'printf \'%s\\n\' "$@" >> "$FAKE_ARGV_FILE"\n'
        "exit 0\n"
    )
    script.chmod(script.stat().st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
    return script


@pytest.fixture
def fake_argv_file(tmp_path: Path) -> Path:
    path = tmp_path / "argv.txt"
    path.touch()
    return path


def set_argv(monkeypatch, argv):
    """Set ``sys.argv`` for CLI tests.

    Single canonical helper replacing the duplicated ``_set_argv`` copies
    previously defined in ``test_cli.py`` and ``test_config.py``.
    """
    monkeypatch.setattr(sys, "argv", argv)


# Alias preserving the previous private name during migration.
_set_argv = set_argv


@pytest.fixture
def mock_formatters(monkeypatch):
    """Replace FORMATTERS with mocks and restore after.

    File discovery is stubbed to return no files so the existing
    formatter-call assertions (``args=[]`` unless the test forwards
    arguments) observe argv construction rather than real filesystem
    state. Discovery behavior itself is covered in test_discovery.py.

    Returns a dict of mocks keyed by formatter name.
    """
    mocks = {
        name: MagicMock(return_value=0)
        for name in ("shfmt", "clang-format", "black", "prettier", "powershell", "taplo")
    }
    monkeypatch.setattr(cli, "discover_files", lambda extensions, root=None: [])
    original = dict(cli.FORMATTERS)
    cli.FORMATTERS.update(mocks)
    try:
        yield mocks
    finally:
        cli.FORMATTERS.clear()
        cli.FORMATTERS.update(original)


def read_recorded(argv_file: Path) -> list[str]:
    """Read argv recorded by a fake executable, one argument per line."""
    if not argv_file.exists():
        return []
    text = argv_file.read_text()
    # Fake scripts write one argument per line with a trailing newline.
    # An empty write means the formatter was invoked with no extra args.
    if text == "":
        return []
    return text.splitlines()


_read_recorded = read_recorded


def make_tools_layout(
    tools_dir: Path,
    name: str,
    version: str,
    platform_dir: str,
    exe_name: str,
) -> Path:
    """Create ``tools/<name>/v<version>/<platform>/<exe>`` as a fake script."""
    exe_dir = tools_dir / name / f"v{version}" / platform_dir
    exe_dir.mkdir(parents=True, exist_ok=True)
    return make_fake_executable(exe_dir, exe_name)


_make_tools_layout = make_tools_layout


def install_fake(monkeypatch, tmp_path: Path, fake_argv_file: Path, name: str) -> None:
    """Point discovery at PATH fakes and ``chdir`` into the fixture root.

    Sets ``TOOLS_DIR`` to an empty directory (forcing PATH lookup),
    creates a fake ``name`` executable on ``PATH``, records argv to
    ``fake_argv_file``, and changes into ``tmp_path``.
    """
    empty_tools = tmp_path / "empty-tools"
    empty_tools.mkdir(exist_ok=True)
    monkeypatch.setattr(tools, "TOOLS_DIR", empty_tools)

    bin_dir = tmp_path / "bin"
    make_fake_executable(bin_dir, name)
    monkeypatch.setenv("PATH", f"{bin_dir}{os.pathsep}{os.environ.get('PATH', '')}")
    monkeypatch.setenv("FAKE_ARGV_FILE", str(fake_argv_file))
    monkeypatch.chdir(tmp_path)


_install_fake = install_fake


# Re-export for callers that prefer a direct import in test modules.
__all__ = [
    "make_fake_executable",
    "set_argv",
    "_set_argv",
    "read_recorded",
    "_read_recorded",
    "make_tools_layout",
    "_make_tools_layout",
    "install_fake",
    "_install_fake",
]
