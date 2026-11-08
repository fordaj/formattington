"""Integration-style tests with fake executables.

These tests never invoke the bundled or real formatter binaries. Instead they
point tool discovery at temporary directories containing stub scripts (or PATH
stubs) and prove that arguments — including paths with spaces — arrive as
correct argv batches in a single subprocess invocation.

On POSIX the fakes are shell scripts executed for real. On Windows they are
skipped (see conftest.posix_only); the mocked-subprocess tests in
test_formatters cover the same argv contracts there.
"""

from __future__ import annotations

import os
import stat
import sys
from pathlib import Path

import pytest

import formattington._tools as tools
from formattington import _tools as tools_module
from formattington.clang_format import clang_format
from formattington.shfmt import shfmt
from tests.conftest import make_fake_executable


def _read_recorded(argv_file: Path) -> list[str]:
    if not argv_file.exists():
        return []
    text = argv_file.read_text()
    # Fake scripts write one argument per line with a trailing newline.
    # An empty write means the formatter was invoked with no extra args.
    if text == "":
        return []
    return text.splitlines()


def _make_tools_layout(
    tools_dir: Path,
    name: str,
    version: str,
    platform_dir: str,
    exe_name: str,
) -> Path:
    """Create tools/<name>/v<version>/<platform>/<exe> as a fake script."""
    exe_dir = tools_dir / name / f"v{version}" / platform_dir
    exe_dir.mkdir(parents=True, exist_ok=True)
    return make_fake_executable(exe_dir, exe_name)


@pytest.mark.skipif(os.name == "nt", reason="shell fakes require POSIX")
class TestBundledFakeExecutables:
    def test_shfmt_finds_bundled_fake_and_records_write_mode(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        tools_dir = tmp_path / "tools"
        _make_tools_layout(tools_dir, "shfmt", "3.14.1", "macos-x86_64", "shfmt")
        monkeypatch.setattr(tools, "TOOLS_DIR", tools_dir)
        monkeypatch.setenv("FAKE_ARGV_FILE", str(fake_argv_file))
        with pytest.MonkeyPatch.context() as mp:
            # Force the darwin layout the fake was created for, regardless
            # of the host platform, so discovery is deterministic.
            mp.setattr("formattington._tools.platform.system", lambda: "Darwin")
            mp.setattr("formattington._tools.platform.machine", lambda: "x86_64")
            assert shfmt() == 0
        # One bare "-w" argument: the mode flag and nothing else.
        assert _read_recorded(fake_argv_file) == ["-w"]

    def test_shfmt_batches_many_paths_in_one_invocation(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        tools_dir = tmp_path / "tools"
        _make_tools_layout(tools_dir, "shfmt", "3.14.1", "macos-x86_64", "shfmt")
        monkeypatch.setattr(tools, "TOOLS_DIR", tools_dir)
        monkeypatch.setenv("FAKE_ARGV_FILE", str(fake_argv_file))
        paths = [
            "a.sh",
            "b with spaces.sh",
            "very/long/path/to/c.sh",
            "d.sh",
        ]
        with pytest.MonkeyPatch.context() as mp:
            mp.setattr("formattington._tools.platform.system", lambda: "Darwin")
            mp.setattr("formattington._tools.platform.machine", lambda: "x86_64")
            assert shfmt(args=paths) == 0
        # A single invocation carried every path (batch, not per-file).
        # The fake appends across invocations, so len(recorded) == len(paths)+1
        # proves one call happened; spaces stayed intact as single entries.
        recorded = _read_recorded(fake_argv_file)
        assert recorded[0] == "-w"
        assert recorded[1:] == paths
        assert "b with spaces.sh" in recorded

    def test_shfmt_check_mode_with_fake(self, monkeypatch, tmp_path, fake_argv_file):
        tools_dir = tmp_path / "tools"
        _make_tools_layout(tools_dir, "shfmt", "3.14.1", "macos-x86_64", "shfmt")
        monkeypatch.setattr(tools, "TOOLS_DIR", tools_dir)
        monkeypatch.setenv("FAKE_ARGV_FILE", str(fake_argv_file))
        with pytest.MonkeyPatch.context() as mp:
            mp.setattr("formattington._tools.platform.system", lambda: "Darwin")
            mp.setattr("formattington._tools.platform.machine", lambda: "x86_64")
            assert shfmt(check=True) == 0
        assert _read_recorded(fake_argv_file) == ["-l"]

    def test_clang_format_batches_paths_with_spaces(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        tools_dir = tmp_path / "tools"
        _make_tools_layout(
            tools_dir, "clang-format", "23.1.3", "macos-x86_64", "clang-format"
        )
        monkeypatch.setattr(tools, "TOOLS_DIR", tools_dir)
        monkeypatch.setenv("FAKE_ARGV_FILE", str(fake_argv_file))
        with pytest.MonkeyPatch.context() as mp:
            mp.setattr("formattington._tools.platform.system", lambda: "Darwin")
            mp.setattr("formattington._tools.platform.machine", lambda: "x86_64")
            assert clang_format(args=["my file.cpp", "another file.cpp"]) == 0
        recorded = _read_recorded(fake_argv_file)
        assert recorded == ["-i", "my file.cpp", "another file.cpp"]

    def test_run_tool_batch_chunking_is_a_single_call(
        self, monkeypatch, tmp_path, fake_argv_file
    ):
        """run_tool sends all arguments in one subprocess.run call.

        This is the batching contract that future file discovery relies on:
        many paths in, one invocation. The fake executable records exactly one
        argv batch, so a second read (via integration with shfmt) would double
        the lines; here we assert each path appears exactly once.
        """
        tools_dir = tmp_path / "tools"
        _make_tools_layout(tools_dir, "shfmt", "3.14.1", "macos-x86_64", "shfmt")
        monkeypatch.setattr(tools, "TOOLS_DIR", tools_dir)
        monkeypatch.setenv("FAKE_ARGV_FILE", str(fake_argv_file))
        with pytest.MonkeyPatch.context() as mp:
            mp.setattr("formattington._tools.platform.system", lambda: "Darwin")
            mp.setattr("formattington._tools.platform.machine", lambda: "x86_64")
            exe = tools.find_tool("shfmt")
            assert tools.run_tool(exe, ["-w", "a.sh", "b with spaces.sh"]) == 0
        recorded = _read_recorded(fake_argv_file)
        assert recorded == ["-w", "a.sh", "b with spaces.sh"]
        # Each path counted once => single batched invocation.
        assert recorded.count("b with spaces.sh") == 1


@pytest.mark.skipif(os.name == "nt", reason="shell fakes require POSIX")
class TestPathStubs:
    def test_falls_back_to_path_stub(self, monkeypatch, tmp_path, fake_argv_file):
        # Empty TOOLS_DIR forces PATH lookup; place a fake `shfmt` on PATH.
        empty_tools = tmp_path / "empty-tools"
        empty_tools.mkdir()
        monkeypatch.setattr(tools, "TOOLS_DIR", empty_tools)

        bin_dir = tmp_path / "bin"
        make_fake_executable(bin_dir, "shfmt")
        monkeypatch.setenv("PATH", f"{bin_dir}{os.pathsep}{os.environ.get('PATH', '')}")
        monkeypatch.setenv("FAKE_ARGV_FILE", str(fake_argv_file))
        with pytest.MonkeyPatch.context() as mp:
            mp.setattr("formattington._tools.platform.system", lambda: "Darwin")
            mp.setattr("formattington._tools.platform.machine", lambda: "x86_64")
            assert shfmt(args=["file with spaces.sh"]) == 0
        assert _read_recorded(fake_argv_file) == ["-w", "file with spaces.sh"]

    def test_cli_uses_path_stub_end_to_end(self, monkeypatch, tmp_path, fake_argv_file):
        empty_tools = tmp_path / "empty-tools"
        empty_tools.mkdir()
        monkeypatch.setattr(tools, "TOOLS_DIR", empty_tools)

        bin_dir = tmp_path / "bin"
        make_fake_executable(bin_dir, "shfmt")
        make_fake_executable(bin_dir, "clang-format")
        monkeypatch.setenv("PATH", f"{bin_dir}{os.pathsep}{os.environ.get('PATH', '')}")
        monkeypatch.setenv("FAKE_ARGV_FILE", str(fake_argv_file))
        monkeypatch.setattr(sys, "argv", ["formattington", "--check", "shfmt"])
        with pytest.MonkeyPatch.context() as mp:
            mp.setattr("formattington._tools.platform.system", lambda: "Darwin")
            mp.setattr("formattington._tools.platform.machine", lambda: "x86_64")
            from formattington.cli import main

            assert main() == 0
        # --check before the formatter maps to shfmt's "-l" batch.
        assert _read_recorded(fake_argv_file) == ["-l"]


class TestFileDiscoveryContract:
    """Discovery-agnostic checks that hold on every platform.

    The current implementation has no separate per-file discovery module;
    formatters receive whole argument batches from the CLI. These tests lock
    in the contract future discovery must preserve: one subprocess call per
    formatter, arguments as discrete argv entries, spaces intact.
    """

    def test_shfmt_single_call_for_many_paths(self):
        from unittest.mock import patch

        paths = [f"dir/file {i}.sh" for i in range(5)]
        calls = []

        def fake_run_tool(executable, arguments, *, check_output=False):
            calls.append((executable, list(arguments)))
            return 0

        with patch(
            "formattington.shfmt.find_tool", return_value="/fake/shfmt"
        ), patch("formattington.shfmt.run_tool", side_effect=fake_run_tool):
            assert shfmt(args=paths) == 0
        assert len(calls) == 1
        _, arguments = calls[0]
        assert arguments == ["-w", *paths]

    def test_clang_format_single_call_for_many_paths(self):
        from unittest.mock import patch

        paths = [f"dir/file {i}.cpp" for i in range(5)]
        calls = []

        def fake_run(cmd, **kwargs):
            calls.append(list(cmd))
            m = type("R", (), {"returncode": 0})()
            return m

        with patch(
            "formattington.clang_format.find_tool", return_value="/fake/clang-format"
        ), patch("formattington.clang_format.subprocess.run", side_effect=fake_run):
            assert clang_format(args=paths) == 0
        assert len(calls) == 1
        assert calls[0] == ["/fake/clang-format", "-i", *paths]
