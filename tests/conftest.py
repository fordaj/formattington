"""Shared fixtures for Formattington tests."""

from __future__ import annotations

import os
import stat
import sys
from pathlib import Path

import pytest


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


@pytest.fixture
def fake_executable_dir(tmp_path: Path) -> Path:
    """Provide a temp dir for fake tools with docs."""
    path = tmp_path / "fake-tools"
    path.mkdir()
    return path


@pytest.fixture
def posix_only():
    """Skip on Windows where shell-script fakes are not executable."""
    if os.name == "nt":
        pytest.skip("fake shell executables require POSIX")


@pytest.fixture
def set_argv(monkeypatch):
    """Helper to set sys.argv for CLI tests."""

    def _set(argv: list[str]):
        monkeypatch.setattr(sys, "argv", argv)

    return _set


# Re-export for callers that prefer a direct import in test modules.
__all__ = ["make_fake_executable"]
